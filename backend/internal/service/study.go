package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/appctx"
)

// StudyService owns study sessions (they are their own source of truth).
// A session created with a habit_id also creates the habit check-in
// referencing the session (source_type='study_session') — the habit log is
// a reference, never a copy, and an existing manual check-in is preserved.
type StudyService struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewStudyService(db *pgxpool.Pool, logger *zap.Logger) *StudyService {
	return &StudyService{db: db, logger: logger}
}

// ─── DTOs ─────────────────────────────────────────────────────────────────────

type StudySessionDTO struct {
	ID              string    `json:"id"`
	Subject         string    `json:"subject"`
	Topic           *string   `json:"topic"`
	StudiedOn       string    `json:"studied_on"`
	DurationMinutes int       `json:"duration_minutes"`
	Notes           *string   `json:"notes"`
	TaskID          *string   `json:"task_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateStudySessionInput struct {
	Subject         string  `json:"subject"`
	Topic           *string `json:"topic"`
	StudiedOn       string  `json:"studied_on"`
	DurationMinutes int     `json:"duration_minutes"`
	Notes           *string `json:"notes"`
	TaskID          *string `json:"task_id"`
	HabitID         *string `json:"habit_id"` // opcional: gera check-in referenciando a sessão
}

type SubjectSummary struct {
	Subject      string `json:"subject"`
	TotalMinutes int    `json:"total_minutes"`
	Sessions     int    `json:"sessions"`
}

type StudySummaryDTO struct {
	TotalMinutes30d int              `json:"total_minutes_30d"`
	Sessions30d     int              `json:"sessions_30d"`
	BySubject       []SubjectSummary `json:"by_subject"`
}

const studySessionColumns = `id::text, subject, topic, studied_on::text, duration_minutes, notes, task_id::text, created_at`

// ─── Methods ─────────────────────────────────────────────────────────────────

func (s *StudyService) ListSessions(ctx context.Context, userID, from, to, subject string) ([]StudySessionDTO, error) {
	query := `SELECT ` + studySessionColumns + `
		FROM study_sessions
		WHERE user_id = $1 AND studied_on BETWEEN $2::date AND $3::date`
	args := []any{userID, from, to}
	if subject != "" {
		args = append(args, subject)
		query += fmt.Sprintf(" AND subject = $%d", len(args))
	}
	query += " ORDER BY studied_on DESC, created_at DESC"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing study sessions: %w", err)
	}
	defer rows.Close()

	var sessions []StudySessionDTO
	for rows.Next() {
		var sess StudySessionDTO
		if err := rows.Scan(&sess.ID, &sess.Subject, &sess.Topic, &sess.StudiedOn,
			&sess.DurationMinutes, &sess.Notes, &sess.TaskID, &sess.CreatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	if sessions == nil {
		sessions = []StudySessionDTO{}
	}
	return sessions, rows.Err()
}

func (s *StudyService) CreateSession(ctx context.Context, userID string, input CreateStudySessionInput) (*StudySessionDTO, error) {
	return s.createSession(ctx, userID, input, appctx.Today(ctx))
}

// createSession defaults studied_on to today, the user's local calendar day.
func (s *StudyService) createSession(ctx context.Context, userID string, input CreateStudySessionInput, today time.Time) (*StudySessionDTO, error) {
	if input.StudiedOn == "" {
		input.StudiedOn = today.Format(dateLayout)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if input.TaskID != nil {
		if err := requireOwned(ctx, tx, ownedTasks, *input.TaskID, userID); err != nil {
			return nil, err
		}
	}

	var sess StudySessionDTO
	err = tx.QueryRow(ctx, `
		INSERT INTO study_sessions (user_id, subject, topic, studied_on, duration_minutes, notes, task_id)
		VALUES ($1, $2, $3, $4::date, $5, $6, $7)
		RETURNING `+studySessionColumns,
		userID, input.Subject, input.Topic, input.StudiedOn, input.DurationMinutes, input.Notes, input.TaskID,
	).Scan(&sess.ID, &sess.Subject, &sess.Topic, &sess.StudiedOn,
		&sess.DurationMinutes, &sess.Notes, &sess.TaskID, &sess.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating study session: %w", err)
	}

	if input.HabitID != nil {
		_, err = tx.Exec(ctx, `
			INSERT INTO habit_logs (habit_id, user_id, logged_date, timer_seconds, source_type, source_id)
			SELECT id, user_id, $3::date, $4, 'study_session', $5
			FROM habits WHERE id = $1 AND user_id = $2 AND is_active = true
			ON CONFLICT (habit_id, logged_date) DO NOTHING`,
			*input.HabitID, userID, sess.StudiedOn, input.DurationMinutes*60, sess.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("linked habit check-in: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing tx: %w", err)
	}
	return &sess, nil
}

func (s *StudyService) DeleteSession(ctx context.Context, sessionID, userID string) error {
	tag, err := s.db.Exec(ctx,
		"DELETE FROM study_sessions WHERE id = $1 AND user_id = $2",
		sessionID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *StudyService) GetSummary(ctx context.Context, userID string) (*StudySummaryDTO, error) {
	return s.getSummary(ctx, userID, appctx.Today(ctx))
}

// getSummary anchors the 30-day window on today, the user's local calendar
// day (not the database server's CURRENT_DATE).
func (s *StudyService) getSummary(ctx context.Context, userID string, today time.Time) (*StudySummaryDTO, error) {
	summary := &StudySummaryDTO{BySubject: []SubjectSummary{}}

	rows, err := s.db.Query(ctx, `
		SELECT subject, COALESCE(SUM(duration_minutes), 0), COUNT(*)
		FROM study_sessions
		WHERE user_id = $1 AND studied_on >= $2::date - INTERVAL '30 days'
		GROUP BY subject
		ORDER BY SUM(duration_minutes) DESC`,
		userID, today.Format(dateLayout),
	)
	if err != nil {
		return nil, fmt.Errorf("summarizing study: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sub SubjectSummary
		if err := rows.Scan(&sub.Subject, &sub.TotalMinutes, &sub.Sessions); err != nil {
			return nil, err
		}
		summary.TotalMinutes30d += sub.TotalMinutes
		summary.Sessions30d += sub.Sessions
		summary.BySubject = append(summary.BySubject, sub)
	}
	return summary, rows.Err()
}
