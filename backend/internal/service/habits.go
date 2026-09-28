package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/appctx"
)

// dateLayout is the calendar-day format used for logged_date keys and for
// passing a day to SQL as $n::date.
const dateLayout = "2006-01-02"

// calendarDay returns t's calendar date (in t's own location) as UTC
// midnight, so it compares cleanly with dates parsed from "YYYY-MM-DD"
// strings — the streak and completion walks rely on that.
func calendarDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// HabitService handles habit business logic.
type HabitService struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewHabitService(db *pgxpool.Pool, logger *zap.Logger) *HabitService {
	return &HabitService{db: db, logger: logger}
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

// MetricField defines a single metric field for metric-type habits.
// IsTarget/TargetValue mark the field the UI pre-fills with a goal — the
// metric is optional enrichment and never blocks the check-in.
type MetricField struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Unit        string   `json:"unit"`
	IsTarget    bool     `json:"is_target,omitempty"`
	TargetValue *float64 `json:"target_value,omitempty"`
}

type HabitDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Color       string    `json:"color"`
	Frequency   string    `json:"frequency"`
	TargetDays  []int32   `json:"target_days"`
	IsActive    bool      `json:"is_active"`
	TimeOfDay   string    `json:"time_of_day"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
	// Check type fields
	CheckType    string        `json:"check_type"`
	TimerMinutes *int          `json:"timer_minutes,omitempty"`
	DeadlineTime *string       `json:"deadline_time,omitempty"`
	MetricConfig []MetricField `json:"metric_config,omitempty"`
	// Computed fields
	CurrentStreak  int  `json:"current_streak"`
	CompletedToday bool `json:"completed_today"`
}

type HabitLogDTO struct {
	ID         string    `json:"id"`
	HabitID    string    `json:"habit_id"`
	LoggedDate string    `json:"logged_date"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
	// Advanced check-in fields
	TimerSeconds *int                   `json:"timer_seconds,omitempty"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	IsManual     bool                   `json:"is_manual"`
	SourceType   string                 `json:"source_type"`
}

type HabitStatsDTO struct {
	TotalHabits      int         `json:"total_habits"`
	CompletedToday   int         `json:"completed_today"`
	TotalCheckIns    int         `json:"total_check_ins"`
	BestStreak       int         `json:"best_streak"`
	CurrentStreak    int         `json:"current_streak"`
	CompletionRate7d float64     `json:"completion_rate_7d"`
	HabitStats       []HabitStat `json:"habit_stats"`
}

type HabitStat struct {
	HabitID        string  `json:"habit_id"`
	HabitName      string  `json:"habit_name"`
	Icon           string  `json:"icon"`
	Color          string  `json:"color"`
	CurrentStreak  int     `json:"current_streak"`
	CompletionRate float64 `json:"completion_rate"`
}

type HeatmapEntry struct {
	Date            string `json:"date"`
	HabitsCompleted int64  `json:"habits_completed"`
}

// CreateHabitInput holds all fields for creating a habit.
type CreateHabitInput struct {
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Icon         string        `json:"icon"`
	Color        string        `json:"color"`
	Frequency    string        `json:"frequency"`
	TargetDays   []int32       `json:"target_days"`
	TimeOfDay    string        `json:"time_of_day"`
	Category     string        `json:"category"`
	CheckType    string        `json:"check_type"`
	TimerMinutes *int          `json:"timer_minutes,omitempty"`
	DeadlineTime *string       `json:"deadline_time,omitempty"`
	MetricConfig []MetricField `json:"metric_config,omitempty"`
}

// UpdateHabitInput carries a partial update: nil pointers keep the current
// value. Check-type dependent fields (timer/deadline/metric_config) are
// replaced together whenever check_type is present.
type UpdateHabitInput struct {
	Name         *string        `json:"name"`
	Description  *string        `json:"description"`
	Icon         *string        `json:"icon"`
	Color        *string        `json:"color"`
	Frequency    *string        `json:"frequency"`
	TargetDays   *[]int32       `json:"target_days"`
	TimeOfDay    *string        `json:"time_of_day"`
	Category     *string        `json:"category"`
	CheckType    *string        `json:"check_type"`
	TimerMinutes *int           `json:"timer_minutes"`
	DeadlineTime *string        `json:"deadline_time"`
	MetricConfig *[]MetricField `json:"metric_config"`
}

// CheckInInput holds the data for an advanced check-in.
type CheckInInput struct {
	Date         string                 `json:"date"`
	Notes        string                 `json:"notes"`
	TimerSeconds *int                   `json:"timer_seconds,omitempty"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	IsManual     bool                   `json:"is_manual"`
	SourceType   string                 `json:"source_type,omitempty"`
	SourceID     *string                `json:"source_id,omitempty"`
}

// ─── Habit columns helper ─────────────────────────────────────────────────────

// habitSelectColumns is the shared SELECT columns for habits.
const habitSelectColumns = `id, name, COALESCE(description, ''), icon, color, frequency::text,
	target_days, is_active, time_of_day, category, created_at,
	check_type::text, timer_minutes, deadline_time::text, metric_config`

const habitLogSelectColumns = `id::text, habit_id::text, logged_date::text, COALESCE(notes, ''), created_at,
	timer_seconds, started_at, completed_at, metrics, is_manual, source_type`

// scanHabit scans a habit row into a HabitDTO. The row must match habitSelectColumns.
func scanHabit(scan func(dest ...any) error) (HabitDTO, error) {
	var h HabitDTO
	var deadlineTime *string
	var metricConfigJSON []byte

	err := scan(
		&h.ID, &h.Name, &h.Description, &h.Icon, &h.Color,
		&h.Frequency, &h.TargetDays, &h.IsActive, &h.TimeOfDay, &h.Category, &h.CreatedAt,
		&h.CheckType, &h.TimerMinutes, &deadlineTime, &metricConfigJSON,
	)
	if err != nil {
		return h, err
	}

	if deadlineTime != nil {
		// Strip seconds from "HH:MM:SS" → "HH:MM"
		t := *deadlineTime
		if len(t) >= 5 {
			t = t[:5]
		}
		h.DeadlineTime = &t
	}

	if len(metricConfigJSON) > 0 {
		json.Unmarshal(metricConfigJSON, &h.MetricConfig) //nolint:errcheck
	}

	return h, nil
}

func scanHabitLog(scan func(dest ...any) error) (HabitLogDTO, error) {
	var l HabitLogDTO
	var metricsOut []byte
	err := scan(&l.ID, &l.HabitID, &l.LoggedDate, &l.Notes, &l.CreatedAt,
		&l.TimerSeconds, &l.StartedAt, &l.CompletedAt, &metricsOut, &l.IsManual, &l.SourceType)
	if err != nil {
		return l, err
	}
	if len(metricsOut) > 0 {
		json.Unmarshal(metricsOut, &l.Metrics) //nolint:errcheck
	}
	return l, nil
}

// ─── Methods ─────────────────────────────────────────────────────────────────

// logDatesByHabit returns, per habit, the set of logged dates within the last
// 400 days — one query for streaks, completion rates and completed-today.
// The window is anchored on today, the user's local calendar day.
func (s *HabitService) logDatesByHabit(ctx context.Context, userID string, today time.Time) (map[string]map[string]bool, error) {
	rows, err := s.db.Query(ctx,
		`SELECT habit_id::text, logged_date::text FROM habit_logs
		 WHERE user_id = $1 AND logged_date >= $2::date - INTERVAL '400 days'`,
		userID, today.Format(dateLayout),
	)
	if err != nil {
		return nil, fmt.Errorf("loading habit logs: %w", err)
	}
	defer rows.Close()

	logs := map[string]map[string]bool{}
	for rows.Next() {
		var habitID, date string
		if err := rows.Scan(&habitID, &date); err != nil {
			return nil, err
		}
		if logs[habitID] == nil {
			logs[habitID] = map[string]bool{}
		}
		logs[habitID][date] = true
	}
	return logs, rows.Err()
}

func (s *HabitService) ListByUser(ctx context.Context, userID string) ([]HabitDTO, error) {
	return s.listByUser(ctx, userID, appctx.Today(ctx))
}

// listByUser computes completed_today and streaks relative to today, the
// user's local calendar day.
func (s *HabitService) listByUser(ctx context.Context, userID string, today time.Time) ([]HabitDTO, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+habitSelectColumns+`
		 FROM habits WHERE user_id = $1 AND is_active = true ORDER BY sort_order ASC, created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing habits: %w", err)
	}
	defer rows.Close()

	var habits []HabitDTO
	for rows.Next() {
		h, err := scanHabit(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scanning habit: %w", err)
		}
		habits = append(habits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	logs, err := s.logDatesByHabit(ctx, userID, today)
	if err != nil {
		return nil, err
	}

	today = calendarDay(today)
	todayKey := today.Format(dateLayout)
	for i := range habits {
		habitLogs := logs[habits[i].ID]
		habits[i].CompletedToday = habitLogs[todayKey]
		habits[i].CurrentStreak, _ = computeStreak(habitLogs, habits[i].TargetDays, today)
	}

	if habits == nil {
		habits = []HabitDTO{}
	}
	return habits, nil
}

func (s *HabitService) Create(ctx context.Context, userID string, input CreateHabitInput) (*HabitDTO, error) {
	if input.TimeOfDay == "" {
		input.TimeOfDay = "anytime"
	}
	if input.Category == "" {
		input.Category = "general"
	}
	if input.CheckType == "" {
		input.CheckType = "simple"
	}
	if input.DeadlineTime != nil && strings.TrimSpace(*input.DeadlineTime) == "" {
		input.DeadlineTime = nil
	}
	if input.TimerMinutes != nil && *input.TimerMinutes <= 0 {
		input.TimerMinutes = nil
	}

	var metricConfigJSON interface{}
	if len(input.MetricConfig) > 0 {
		b, err := json.Marshal(input.MetricConfig)
		if err != nil {
			return nil, fmt.Errorf("marshaling metric_config: %w", err)
		}
		metricConfigJSON = b
	}

	row := s.db.QueryRow(ctx,
		`INSERT INTO habits (user_id, name, description, icon, color, frequency, target_days, time_of_day, category,
		   check_type, timer_minutes, deadline_time, metric_config)
		 VALUES ($1, $2, $3, $4, $5, $6::habit_frequency, $7, $8, $9,
		   $10::habit_check_type, $11, $12::time, $13)
		 RETURNING `+habitSelectColumns,
		userID, input.Name, input.Description, input.Icon, input.Color, input.Frequency,
		input.TargetDays, input.TimeOfDay, input.Category,
		input.CheckType, input.TimerMinutes, input.DeadlineTime, metricConfigJSON,
	)

	h, err := scanHabit(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("creating habit: %w", err)
	}
	return &h, nil
}

func (s *HabitService) GetByID(ctx context.Context, habitID, userID string) (*HabitDTO, error) {
	row := s.db.QueryRow(ctx,
		`SELECT `+habitSelectColumns+`
		 FROM habits WHERE id = $1 AND user_id = $2 AND is_active = true`,
		habitID, userID,
	)
	h, err := scanHabit(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting habit: %w", err)
	}
	return &h, nil
}

// Update merges the partial input over the current habit and persists every
// column — including the check-type dependent ones the old version dropped.
func (s *HabitService) Update(ctx context.Context, habitID, userID string, input UpdateHabitInput) (*HabitDTO, error) {
	current, err := s.GetByID(ctx, habitID, userID)
	if err != nil {
		return nil, err
	}

	merged := *current
	if input.Name != nil {
		merged.Name = *input.Name
	}
	if input.Description != nil {
		merged.Description = *input.Description
	}
	if input.Icon != nil {
		merged.Icon = *input.Icon
	}
	if input.Color != nil {
		merged.Color = *input.Color
	}
	if input.Frequency != nil && *input.Frequency != "" {
		merged.Frequency = *input.Frequency
	}
	if input.TargetDays != nil && len(*input.TargetDays) > 0 {
		merged.TargetDays = *input.TargetDays
	}
	if input.TimeOfDay != nil && *input.TimeOfDay != "" {
		merged.TimeOfDay = *input.TimeOfDay
	}
	if input.Category != nil && *input.Category != "" {
		merged.Category = *input.Category
	}
	if input.CheckType != nil && *input.CheckType != "" {
		// Switching check type replaces the dependent config as a unit.
		merged.CheckType = *input.CheckType
		merged.TimerMinutes = input.TimerMinutes
		merged.DeadlineTime = input.DeadlineTime
		merged.MetricConfig = nil
		if input.MetricConfig != nil {
			merged.MetricConfig = *input.MetricConfig
		}
	} else {
		if input.TimerMinutes != nil {
			merged.TimerMinutes = input.TimerMinutes
		}
		if input.DeadlineTime != nil {
			merged.DeadlineTime = input.DeadlineTime
		}
		if input.MetricConfig != nil {
			merged.MetricConfig = *input.MetricConfig
		}
	}

	if merged.TimerMinutes != nil && *merged.TimerMinutes <= 0 {
		merged.TimerMinutes = nil
	}
	if merged.DeadlineTime != nil && strings.TrimSpace(*merged.DeadlineTime) == "" {
		merged.DeadlineTime = nil
	}
	switch merged.CheckType {
	case "timed":
		merged.DeadlineTime = nil
		merged.MetricConfig = nil
	case "deadline":
		merged.TimerMinutes = nil
		merged.MetricConfig = nil
	case "metric":
		merged.TimerMinutes = nil
		merged.DeadlineTime = nil
	case "simple":
		merged.TimerMinutes = nil
		merged.DeadlineTime = nil
		merged.MetricConfig = nil
	}

	var metricConfigJSON interface{}
	if len(merged.MetricConfig) > 0 {
		b, err := json.Marshal(merged.MetricConfig)
		if err != nil {
			return nil, fmt.Errorf("marshaling metric_config: %w", err)
		}
		metricConfigJSON = b
	}

	row := s.db.QueryRow(ctx,
		`UPDATE habits SET name=$3, description=$4, icon=$5, color=$6, frequency=$7::habit_frequency,
		   target_days=$8, time_of_day=$9, category=$10,
		   check_type=$11::habit_check_type, timer_minutes=$12, deadline_time=$13::time, metric_config=$14
		 WHERE id=$1 AND user_id=$2
		 RETURNING `+habitSelectColumns,
		habitID, userID, merged.Name, merged.Description, merged.Icon, merged.Color, merged.Frequency,
		merged.TargetDays, merged.TimeOfDay, merged.Category,
		merged.CheckType, merged.TimerMinutes, merged.DeadlineTime, metricConfigJSON,
	)
	h, err := scanHabit(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("updating habit: %w", err)
	}
	return &h, nil
}

func (s *HabitService) Delete(ctx context.Context, habitID, userID string) error {
	tag, err := s.db.Exec(ctx,
		"UPDATE habits SET is_active = false WHERE id = $1 AND user_id = $2 AND is_active = true",
		habitID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CheckIn performs a check-in with validation based on the habit's check_type.
func (s *HabitService) CheckIn(ctx context.Context, habitID, userID string, input CheckInInput) (*HabitLogDTO, error) {
	return s.checkIn(ctx, habitID, userID, input, time.Now())
}

// checkIn takes the current instant explicitly so tests can pin the clock.
// Both the default logged_date and the deadline comparison use now as seen
// in the user's timezone, never the server's.
func (s *HabitService) checkIn(ctx context.Context, habitID, userID string, input CheckInInput, now time.Time) (*HabitLogDTO, error) {
	// Get the habit to validate check-in rules
	habit, err := s.GetByID(ctx, habitID, userID)
	if err != nil {
		return nil, ErrNotFound
	}

	now = now.In(appctx.UserTimezone(ctx))

	date := input.Date
	if date == "" {
		date = now.Format(dateLayout)
	}
	if input.SourceType == "" {
		input.SourceType = "manual"
	}

	// Validate based on check_type
	switch habit.CheckType {
	case "timed":
		if !input.IsManual {
			if input.TimerSeconds == nil || *input.TimerSeconds < 0 {
				return nil, fmt.Errorf("timer data required for timed habits")
			}
			requiredSeconds := 0
			if habit.TimerMinutes != nil {
				requiredSeconds = *habit.TimerMinutes * 60
			}
			if *input.TimerSeconds < requiredSeconds {
				return nil, fmt.Errorf("timer must reach at least %d minutes", *habit.TimerMinutes)
			}
		} else {
			// Manual entry — timer_seconds is required but no minimum enforced
			if input.TimerSeconds == nil {
				return nil, fmt.Errorf("timer_seconds required even for manual entries")
			}
		}

	case "deadline":
		if habit.DeadlineTime != nil {
			// The deadline is a wall-clock time in the user's timezone
			// (now.Location() is the user's zone here).
			deadlineStr := fmt.Sprintf("%sT%s:00", date, *habit.DeadlineTime)
			deadline, parseErr := time.ParseInLocation("2006-01-02T15:04:00", deadlineStr, now.Location())
			if parseErr == nil && now.After(deadline) {
				return nil, fmt.Errorf("deadline passed: check-in must be before %s", *habit.DeadlineTime)
			}
		}

	case "metric":
		// Métricas são enriquecimento opcional — nunca bloqueiam o check-in.
		// O usuário pode dar check mesmo sem preencher os campos (ex: correu 3km de 5km, ainda conta).
	}

	// Marshal metrics to JSON
	var metricsJSON interface{}
	if len(input.Metrics) > 0 {
		b, err := json.Marshal(input.Metrics)
		if err != nil {
			return nil, fmt.Errorf("marshaling metrics: %w", err)
		}
		metricsJSON = b
	}

	row := s.db.QueryRow(ctx,
		`INSERT INTO habit_logs (habit_id, user_id, logged_date, notes, timer_seconds, started_at, completed_at, metrics, is_manual, source_type, source_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (habit_id, logged_date)
		 DO UPDATE SET notes = EXCLUDED.notes, timer_seconds = EXCLUDED.timer_seconds,
		   started_at = EXCLUDED.started_at, completed_at = EXCLUDED.completed_at,
		   metrics = EXCLUDED.metrics, is_manual = EXCLUDED.is_manual
		 RETURNING `+habitLogSelectColumns,
		habitID, userID, date, input.Notes, input.TimerSeconds, input.StartedAt, input.CompletedAt,
		metricsJSON, input.IsManual, input.SourceType, input.SourceID,
	)

	log, err := scanHabitLog(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("check-in: %w", err)
	}
	return &log, nil
}

func (s *HabitService) UndoCheckIn(ctx context.Context, habitID, userID, date string) error {
	tag, err := s.db.Exec(ctx,
		"DELETE FROM habit_logs WHERE habit_id = $1 AND user_id = $2 AND logged_date = $3",
		habitID, userID, date,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *HabitService) GetLogs(ctx context.Context, habitID, userID, from, to string) ([]HabitLogDTO, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+habitLogSelectColumns+`
		 FROM habit_logs WHERE habit_id = $1 AND user_id = $2
		 AND logged_date BETWEEN $3 AND $4 ORDER BY logged_date DESC`,
		habitID, userID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []HabitLogDTO
	for rows.Next() {
		l, err := scanHabitLog(rows.Scan)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []HabitLogDTO{}
	}
	return logs, rows.Err()
}

func (s *HabitService) GetStats(ctx context.Context, userID string) (*HabitStatsDTO, error) {
	return s.getStats(ctx, userID, appctx.Today(ctx))
}

// getStats computes today/7-day figures relative to today, the user's local
// calendar day.
func (s *HabitService) getStats(ctx context.Context, userID string, today time.Time) (*HabitStatsDTO, error) {
	stats := &HabitStatsDTO{HabitStats: []HabitStat{}}

	if err := s.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM habit_logs WHERE user_id = $1", userID,
	).Scan(&stats.TotalCheckIns); err != nil {
		return nil, fmt.Errorf("counting check-ins: %w", err)
	}

	habits, err := s.listByUser(ctx, userID, today)
	if err != nil {
		return nil, err
	}

	logs, err := s.logDatesByHabit(ctx, userID, today)
	if err != nil {
		return nil, err
	}

	today = calendarDay(today)
	weekStart := today.AddDate(0, 0, -6)
	stats.TotalHabits = len(habits)

	var scheduled7d, done7d int
	for _, h := range habits {
		habitLogs := logs[h.ID]
		if h.CompletedToday {
			stats.CompletedToday++
		}

		current, best := computeStreak(habitLogs, h.TargetDays, today)
		if current > stats.CurrentStreak {
			stats.CurrentStreak = current
		}
		if best > stats.BestStreak {
			stats.BestStreak = best
		}

		target := make(map[int32]bool, len(h.TargetDays))
		for _, d := range h.TargetDays {
			target[d] = true
		}
		for day := weekStart; !day.After(today); day = day.AddDate(0, 0, 1) {
			if !target[isoWeekday(day)] {
				continue
			}
			scheduled7d++
			if habitLogs[day.Format("2006-01-02")] {
				done7d++
			}
		}

		stats.HabitStats = append(stats.HabitStats, HabitStat{
			HabitID:        h.ID,
			HabitName:      h.Name,
			Icon:           h.Icon,
			Color:          h.Color,
			CurrentStreak:  current,
			CompletionRate: completionRate(habitLogs, h.TargetDays, weekStart, today),
		})
	}

	if scheduled7d > 0 {
		stats.CompletionRate7d = float64(done7d) / float64(scheduled7d)
	}

	return stats, nil
}

func (s *HabitService) GetHeatmap(ctx context.Context, userID string) ([]HeatmapEntry, error) {
	return s.getHeatmap(ctx, userID, appctx.Today(ctx))
}

// getHeatmap covers the 365 days up to today, the user's local calendar day.
func (s *HabitService) getHeatmap(ctx context.Context, userID string, today time.Time) ([]HeatmapEntry, error) {
	rows, err := s.db.Query(ctx,
		`SELECT logged_date::text, COUNT(*) as habits_completed
		 FROM habit_logs WHERE user_id = $1
		   AND logged_date >= $2::date - INTERVAL '365 days'
		 GROUP BY logged_date ORDER BY logged_date ASC`,
		userID, today.Format(dateLayout),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []HeatmapEntry
	for rows.Next() {
		var e HeatmapEntry
		if err := rows.Scan(&e.Date, &e.HabitsCompleted); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []HeatmapEntry{}
	}
	return entries, nil
}
