package service

import (
	"context"
	"fmt"
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

type HabitDTO struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Icon         string    `json:"icon"`
	Color        string    `json:"color"`
	Frequency    string    `json:"frequency"`
	TargetDays   []int32   `json:"target_days"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	// Computed fields
	CurrentStreak int  `json:"current_streak"`
	CompletedToday bool `json:"completed_today"`
}

type HabitLogDTO struct {
	ID         string    `json:"id"`
	HabitID    string    `json:"habit_id"`
	LoggedDate string    `json:"logged_date"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
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

// ─── Methods ─────────────────────────────────────────────────────────────────

func (s *HabitService) ListByUser(ctx context.Context, userID string) ([]HabitDTO, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, name, COALESCE(description, ''), icon, color, frequency::text, target_days, is_active, created_at
		 FROM habits WHERE user_id = $1 AND is_active = true ORDER BY sort_order ASC, created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing habits: %w", err)
	}
	defer rows.Close()

	var habits []HabitDTO
	for rows.Next() {
		var h HabitDTO
		if err := rows.Scan(&h.ID, &h.Name, &h.Description, &h.Icon, &h.Color,
			&h.Frequency, &h.TargetDays, &h.IsActive, &h.CreatedAt); err != nil {
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

func (s *HabitService) Create(ctx context.Context, userID, name, desc, icon, color, frequency string, targetDays []int32) (*HabitDTO, error) {
	var h HabitDTO
	err := s.db.QueryRow(ctx,
		`INSERT INTO habits (user_id, name, description, icon, color, frequency, target_days)
		 VALUES ($1, $2, $3, $4, $5, $6::habit_frequency, $7)
		 RETURNING id, name, COALESCE(description, ''), icon, color, frequency::text, target_days, is_active, created_at`,
		userID, name, desc, icon, color, frequency, targetDays,
	).Scan(&h.ID, &h.Name, &h.Description, &h.Icon, &h.Color, &h.Frequency, &h.TargetDays, &h.IsActive, &h.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating habit: %w", err)
	}
	return &h, nil
}

func (s *HabitService) GetByID(ctx context.Context, habitID, userID string) (*HabitDTO, error) {
	var h HabitDTO
	err := s.db.QueryRow(ctx,
		`SELECT id, name, COALESCE(description, ''), icon, color, frequency::text, target_days, is_active, created_at
		 FROM habits WHERE id = $1 AND user_id = $2 AND is_active = true`,
		habitID, userID,
	).Scan(&h.ID, &h.Name, &h.Description, &h.Icon, &h.Color, &h.Frequency, &h.TargetDays, &h.IsActive, &h.CreatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &h, nil
}

func (s *HabitService) Update(ctx context.Context, habitID, userID, name, desc, icon, color, frequency string, targetDays []int32) (*HabitDTO, error) {
	var h HabitDTO
	err := s.db.QueryRow(ctx,
		`UPDATE habits SET name=$3, description=$4, icon=$5, color=$6, frequency=$7::habit_frequency, target_days=$8
		 WHERE id=$1 AND user_id=$2
		 RETURNING id, name, COALESCE(description, ''), icon, color, frequency::text, target_days, is_active, created_at`,
		habitID, userID, name, desc, icon, color, frequency, targetDays,
	).Scan(&h.ID, &h.Name, &h.Description, &h.Icon, &h.Color, &h.Frequency, &h.TargetDays, &h.IsActive, &h.CreatedAt)
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

func (s *HabitService) CheckIn(ctx context.Context, habitID, userID, date, notes string) (*HabitLogDTO, error) {
	// Verify habit belongs to user
	if _, err := s.GetByID(ctx, habitID, userID); err != nil {
		return nil, ErrNotFound
	}

	var log HabitLogDTO
	err := s.db.QueryRow(ctx,
		`INSERT INTO habit_logs (habit_id, user_id, logged_date, notes)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (habit_id, logged_date)
		 DO UPDATE SET notes = EXCLUDED.notes
		 RETURNING id::text, habit_id::text, logged_date::text, COALESCE(notes, ''), created_at`,
		habitID, userID, date, notes,
	).Scan(&log.ID, &log.HabitID, &log.LoggedDate, &log.Notes, &log.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("check-in: %w", err)
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
		`SELECT id::text, habit_id::text, logged_date::text, COALESCE(notes, ''), created_at
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
		if err := rows.Scan(&l.ID, &l.HabitID, &l.LoggedDate, &l.Notes, &l.CreatedAt); err != nil {
			return nil, err
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
