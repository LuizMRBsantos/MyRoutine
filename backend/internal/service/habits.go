package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

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
type MetricField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Unit  string `json:"unit"`
}

type HabitDTO struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Icon         string    `json:"icon"`
	Color        string    `json:"color"`
	Frequency    string    `json:"frequency"`
	TargetDays   []int32   `json:"target_days"`
	IsActive     bool      `json:"is_active"`
	TimeOfDay    string    `json:"time_of_day"`
	CreatedAt    time.Time `json:"created_at"`
	// Check type fields
	CheckType    string         `json:"check_type"`
	TimerMinutes *int           `json:"timer_minutes,omitempty"`
	DeadlineTime *string        `json:"deadline_time,omitempty"`
	MetricConfig []MetricField  `json:"metric_config,omitempty"`
	// Computed fields
	CurrentStreak  int  `json:"current_streak"`
	CompletedToday bool `json:"completed_today"`
}

type HabitLogDTO struct {
	ID           string                 `json:"id"`
	HabitID      string                 `json:"habit_id"`
	LoggedDate   string                 `json:"logged_date"`
	Notes        string                 `json:"notes"`
	CreatedAt    time.Time              `json:"created_at"`
	// Advanced check-in fields
	TimerSeconds *int                   `json:"timer_seconds,omitempty"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	IsManual     bool                   `json:"is_manual"`
}

type HabitStatsDTO struct {
	TotalHabits      int            `json:"total_habits"`
	CompletedToday   int            `json:"completed_today"`
	TotalCheckIns    int            `json:"total_check_ins"`
	BestStreak       int            `json:"best_streak"`
	CurrentStreak    int            `json:"current_streak"`
	CompletionRate7d float64        `json:"completion_rate_7d"`
	HabitStats       []HabitStat    `json:"habit_stats"`
}

type HabitStat struct {
	HabitID       string  `json:"habit_id"`
	HabitName     string  `json:"habit_name"`
	Icon          string  `json:"icon"`
	Color         string  `json:"color"`
	CurrentStreak int     `json:"current_streak"`
	CompletionRate float64 `json:"completion_rate"`
}

type HeatmapEntry struct {
	Date            string `json:"date"`
	HabitsCompleted int64  `json:"habits_completed"`
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
}

// ─── Habit columns helper ─────────────────────────────────────────────────────

// scanHabitColumns is the shared SELECT columns for habits.
const habitSelectColumns = `id, name, COALESCE(description, ''), icon, color, frequency::text,
	target_days, is_active, time_of_day, created_at,
	check_type::text, timer_minutes, deadline_time::text, metric_config`

// scanHabit scans a habit row into a HabitDTO. The row must match habitSelectColumns.
func scanHabit(scan func(dest ...any) error) (HabitDTO, error) {
	var h HabitDTO
	var deadlineTime *string
	var metricConfigJSON []byte

	err := scan(
		&h.ID, &h.Name, &h.Description, &h.Icon, &h.Color,
		&h.Frequency, &h.TargetDays, &h.IsActive, &h.TimeOfDay, &h.CreatedAt,
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

// ─── Methods ─────────────────────────────────────────────────────────────────

func (s *HabitService) ListByUser(ctx context.Context, userID string) ([]HabitDTO, error) {
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

		// Check if completed today
		today := time.Now().Format("2006-01-02")
		var count int
		s.db.QueryRow(ctx,
			"SELECT COUNT(*) FROM habit_logs WHERE habit_id = $1 AND logged_date = $2",
			h.ID, today,
		).Scan(&count) //nolint:errcheck
		h.CompletedToday = count > 0

		habits = append(habits, h)
	}

	if habits == nil {
		habits = []HabitDTO{}
	}
	return habits, nil
}

func (s *HabitService) Create(ctx context.Context, userID, name, desc, icon, color, frequency, timeOfDay, checkType string, targetDays []int32, timerMinutes *int, deadlineTime *string, metricConfig []MetricField) (*HabitDTO, error) {
	if timeOfDay == "" {
		timeOfDay = "anytime"
	}
	if checkType == "" {
		checkType = "simple"
	}
	if deadlineTime != nil && strings.TrimSpace(*deadlineTime) == "" {
		deadlineTime = nil
	}
	if timerMinutes != nil && *timerMinutes <= 0 {
		timerMinutes = nil
	}

	var metricConfigJSON interface{}
	if len(metricConfig) > 0 {
		b, err := json.Marshal(metricConfig)
		if err != nil {
			return nil, fmt.Errorf("marshaling metric_config: %w", err)
		}
		metricConfigJSON = b
	}

	row := s.db.QueryRow(ctx,
		`INSERT INTO habits (user_id, name, description, icon, color, frequency, target_days, time_of_day,
		   check_type, timer_minutes, deadline_time, metric_config)
		 VALUES ($1, $2, $3, $4, $5, $6::habit_frequency, $7, $8,
		   $9::habit_check_type, $10, $11::time, $12)
		 RETURNING `+habitSelectColumns,
		userID, name, desc, icon, color, frequency, targetDays, timeOfDay,
		checkType, timerMinutes, deadlineTime, metricConfigJSON,
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
		return nil, ErrNotFound
	}
	return &h, nil
}

func (s *HabitService) Update(ctx context.Context, habitID, userID, name, desc, icon, color, frequency string, targetDays []int32) (*HabitDTO, error) {
	row := s.db.QueryRow(ctx,
		`UPDATE habits SET name=$3, description=$4, icon=$5, color=$6, frequency=$7::habit_frequency, target_days=$8
		 WHERE id=$1 AND user_id=$2
		 RETURNING `+habitSelectColumns,
		habitID, userID, name, desc, icon, color, frequency, targetDays,
	)
	h, err := scanHabit(row.Scan)
	if err != nil {
		return nil, ErrNotFound
	}
	return &h, nil
}

func (s *HabitService) Delete(ctx context.Context, habitID, userID string) error {
	_, err := s.db.Exec(ctx,
		"UPDATE habits SET is_active = false WHERE id = $1 AND user_id = $2",
		habitID, userID,
	)
	return err
}

// CheckIn performs a check-in with validation based on the habit's check_type.
func (s *HabitService) CheckIn(ctx context.Context, habitID, userID string, input CheckInInput) (*HabitLogDTO, error) {
	// Get the habit to validate check-in rules
	habit, err := s.GetByID(ctx, habitID, userID)
	if err != nil {
		return nil, ErrNotFound
	}

	date := input.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
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
			now := time.Now()
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

	var log HabitLogDTO
	var metricsOut []byte

	err = s.db.QueryRow(ctx,
		`INSERT INTO habit_logs (habit_id, user_id, logged_date, notes, timer_seconds, started_at, completed_at, metrics, is_manual)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (habit_id, logged_date)
		 DO UPDATE SET notes = EXCLUDED.notes, timer_seconds = EXCLUDED.timer_seconds,
		   started_at = EXCLUDED.started_at, completed_at = EXCLUDED.completed_at,
		   metrics = EXCLUDED.metrics, is_manual = EXCLUDED.is_manual
		 RETURNING id::text, habit_id::text, logged_date::text, COALESCE(notes, ''), created_at,
		   timer_seconds, started_at, completed_at, metrics, is_manual`,
		habitID, userID, date, input.Notes, input.TimerSeconds, input.StartedAt, input.CompletedAt, metricsJSON, input.IsManual,
	).Scan(&log.ID, &log.HabitID, &log.LoggedDate, &log.Notes, &log.CreatedAt,
		&log.TimerSeconds, &log.StartedAt, &log.CompletedAt, &metricsOut, &log.IsManual)
	if err != nil {
		return nil, fmt.Errorf("check-in: %w", err)
	}

	if len(metricsOut) > 0 {
		json.Unmarshal(metricsOut, &log.Metrics) //nolint:errcheck
	}

	return &log, nil
}

func (s *HabitService) UndoCheckIn(ctx context.Context, habitID, userID, date string) error {
	_, err := s.db.Exec(ctx,
		"DELETE FROM habit_logs WHERE habit_id = $1 AND user_id = $2 AND logged_date = $3",
		habitID, userID, date,
	)
	return err
}

func (s *HabitService) GetLogs(ctx context.Context, habitID, userID, from, to string) ([]HabitLogDTO, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id::text, habit_id::text, logged_date::text, COALESCE(notes, ''), created_at,
		   timer_seconds, started_at, completed_at, metrics, is_manual
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
		var l HabitLogDTO
		var metricsOut []byte
		if err := rows.Scan(&l.ID, &l.HabitID, &l.LoggedDate, &l.Notes, &l.CreatedAt,
			&l.TimerSeconds, &l.StartedAt, &l.CompletedAt, &metricsOut, &l.IsManual); err != nil {
			return nil, err
		}
		if len(metricsOut) > 0 {
			json.Unmarshal(metricsOut, &l.Metrics) //nolint:errcheck
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []HabitLogDTO{}
	}
	return logs, nil
}

func (s *HabitService) GetStats(ctx context.Context, userID string) (*HabitStatsDTO, error) {
	stats := &HabitStatsDTO{}

	// Total habits
	s.db.QueryRow(ctx, "SELECT COUNT(*) FROM habits WHERE user_id = $1 AND is_active = true", userID).
		Scan(&stats.TotalHabits) //nolint:errcheck

	// Completed today
	today := time.Now().Format("2006-01-02")
	s.db.QueryRow(ctx,
		"SELECT COUNT(DISTINCT habit_id) FROM habit_logs WHERE user_id = $1 AND logged_date = $2",
		userID, today,
	).Scan(&stats.CompletedToday) //nolint:errcheck

	// Total check-ins
	s.db.QueryRow(ctx, "SELECT COUNT(*) FROM habit_logs WHERE user_id = $1", userID).
		Scan(&stats.TotalCheckIns) //nolint:errcheck

	return stats, nil
}

func (s *HabitService) GetHeatmap(ctx context.Context, userID string) ([]HeatmapEntry, error) {
	rows, err := s.db.Query(ctx,
		`SELECT logged_date::text, COUNT(*) as habits_completed
		 FROM habit_logs WHERE user_id = $1
		   AND logged_date >= CURRENT_DATE - INTERVAL '365 days'
		 GROUP BY logged_date ORDER BY logged_date ASC`,
		userID,
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
