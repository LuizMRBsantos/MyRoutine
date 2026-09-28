package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/appctx"
)

// HealthService is a CONSUMER module: activities are habit_logs of habits
// with category='health' — it queries by reference and never keeps copies
// (product constitution: single source of truth per event). Only
// body_metrics is health's own data.
type HealthService struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewHealthService(db *pgxpool.Pool, logger *zap.Logger) *HealthService {
	return &HealthService{db: db, logger: logger}
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type ActivityDTO struct {
	LogID        string                 `json:"log_id"`
	HabitID      string                 `json:"habit_id"`
	HabitName    string                 `json:"habit_name"`
	Icon         string                 `json:"icon"`
	Color        string                 `json:"color"`
	Date         string                 `json:"date"`
	TimerSeconds *int                   `json:"timer_seconds,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	Notes        string                 `json:"notes"`
	SourceType   string                 `json:"source_type"`
}

type WeeklyVolume struct {
	WeekStart    string  `json:"week_start"`
	Sessions     int     `json:"sessions"`
	TotalKm      float64 `json:"total_km"`
	TotalMinutes int     `json:"total_minutes"`
	AvgRPE       float64 `json:"avg_rpe"`
}

type HealthSummaryDTO struct {
	Weeks []WeeklyVolume `json:"weeks"`
}

type BodyMetricDTO struct {
	ID         string   `json:"id"`
	MeasuredOn string   `json:"measured_on"`
	WeightKg   *float64 `json:"weight_kg"`
	Notes      *string  `json:"notes"`
}

// ─── Activities (referência a habit_logs) ────────────────────────────────────

func (s *HealthService) ListActivities(ctx context.Context, userID, from, to string) ([]ActivityDTO, error) {
	rows, err := s.db.Query(ctx, `
		SELECT hl.id::text, h.id::text, h.name, h.icon, h.color, hl.logged_date::text,
		       hl.timer_seconds, hl.metrics, COALESCE(hl.notes, ''), hl.source_type
		FROM habit_logs hl
		JOIN habits h ON h.id = hl.habit_id
		WHERE hl.user_id = $1 AND h.category = 'health'
		  AND hl.logged_date BETWEEN $2::date AND $3::date
		ORDER BY hl.logged_date DESC`,
		userID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("listing activities: %w", err)
	}
	defer rows.Close()

	var activities []ActivityDTO
	for rows.Next() {
		var a ActivityDTO
		var metricsJSON []byte
		if err := rows.Scan(&a.LogID, &a.HabitID, &a.HabitName, &a.Icon, &a.Color, &a.Date,
			&a.TimerSeconds, &metricsJSON, &a.Notes, &a.SourceType); err != nil {
			return nil, err
		}
		if len(metricsJSON) > 0 {
			json.Unmarshal(metricsJSON, &a.Metrics) //nolint:errcheck
		}
		activities = append(activities, a)
	}
	if activities == nil {
		activities = []ActivityDTO{}
	}
	return activities, rows.Err()
}

// GetSummary aggregates weekly training volume from habit_logs metrics.
// km comes from metrics->>'km', minutes from timer_seconds or
// metrics->>'time_min', RPE from metrics->>'rpe'.
func (s *HealthService) GetSummary(ctx context.Context, userID string, weeks int) (*HealthSummaryDTO, error) {
	return s.getSummary(ctx, userID, weeks, appctx.Today(ctx))
}

// getSummary anchors the week window on today, the user's local calendar day.
func (s *HealthService) getSummary(ctx context.Context, userID string, weeks int, today time.Time) (*HealthSummaryDTO, error) {
	if weeks <= 0 || weeks > 26 {
		weeks = 4
	}

	rows, err := s.db.Query(ctx, `
		SELECT
		  date_trunc('week', hl.logged_date)::date::text AS week_start,
		  COUNT(*) AS sessions,
		  COALESCE(SUM((hl.metrics->>'km')::numeric) FILTER (WHERE hl.metrics ? 'km'), 0) AS total_km,
		  COALESCE(SUM(hl.timer_seconds) / 60, 0)
		    + COALESCE(SUM((hl.metrics->>'time_min')::numeric) FILTER (WHERE hl.metrics ? 'time_min' AND hl.timer_seconds IS NULL), 0) AS total_minutes,
		  COALESCE(AVG((hl.metrics->>'rpe')::numeric) FILTER (WHERE hl.metrics ? 'rpe'), 0) AS avg_rpe
		FROM habit_logs hl
		JOIN habits h ON h.id = hl.habit_id
		WHERE hl.user_id = $1 AND h.category = 'health'
		  AND hl.logged_date >= date_trunc('week', $3::date) - ($2 - 1) * INTERVAL '1 week'
		GROUP BY week_start
		ORDER BY week_start DESC`,
		userID, weeks, today.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("summarizing health: %w", err)
	}
	defer rows.Close()

	summary := &HealthSummaryDTO{Weeks: []WeeklyVolume{}}
	for rows.Next() {
		var w WeeklyVolume
		var totalMinutes float64
		if err := rows.Scan(&w.WeekStart, &w.Sessions, &w.TotalKm, &totalMinutes, &w.AvgRPE); err != nil {
			return nil, err
		}
		w.TotalMinutes = int(totalMinutes)
		summary.Weeks = append(summary.Weeks, w)
	}
	return summary, rows.Err()
}

// ─── Body metrics ─────────────────────────────────────────────────────────────

func (s *HealthService) ListBodyMetrics(ctx context.Context, userID string, limit int) ([]BodyMetricDTO, error) {
	if limit <= 0 || limit > 365 {
		limit = 90
	}
	rows, err := s.db.Query(ctx, `
		SELECT id::text, measured_on::text, weight_kg, notes
		FROM body_metrics
		WHERE user_id = $1
		ORDER BY measured_on DESC
		LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listing body metrics: %w", err)
	}
	defer rows.Close()

	var metrics []BodyMetricDTO
	for rows.Next() {
		var m BodyMetricDTO
		if err := rows.Scan(&m.ID, &m.MeasuredOn, &m.WeightKg, &m.Notes); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	if metrics == nil {
		metrics = []BodyMetricDTO{}
	}
	return metrics, rows.Err()
}

// UpsertBodyMetric records one measurement per day (re-recording updates it).
func (s *HealthService) UpsertBodyMetric(ctx context.Context, userID, measuredOn string, weightKg *float64, notes *string) (*BodyMetricDTO, error) {
	if measuredOn == "" {
		measuredOn = appctx.Today(ctx).Format("2006-01-02")
	}
	var m BodyMetricDTO
	err := s.db.QueryRow(ctx, `
		INSERT INTO body_metrics (user_id, measured_on, weight_kg, notes)
		VALUES ($1, $2::date, $3, $4)
		ON CONFLICT (user_id, measured_on)
		DO UPDATE SET weight_kg = EXCLUDED.weight_kg, notes = EXCLUDED.notes
		RETURNING id::text, measured_on::text, weight_kg, notes`,
		userID, measuredOn, weightKg, notes,
	).Scan(&m.ID, &m.MeasuredOn, &m.WeightKg, &m.Notes)
	if err != nil {
		return nil, fmt.Errorf("upserting body metric: %w", err)
	}
	return &m, nil
}
