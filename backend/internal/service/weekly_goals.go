package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Metas da semana: simples (feita ou não), de segunda a domingo, no
// Dashboard. Sem contagem, placar ou cobrança (product-constitution).

// ErrInvalidWeeklyGoal: empty title or a date that is not YYYY-MM-DD.
var ErrInvalidWeeklyGoal = errors.New("invalid weekly goal")

type WeeklyGoalDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Week      string    `json:"week"` // segunda-feira da semana, "YYYY-MM-DD"
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

type WeeklyGoalService struct {
	db *pgxpool.Pool
}

func NewWeeklyGoalService(db *pgxpool.Pool) *WeeklyGoalService {
	return &WeeklyGoalService{db: db}
}

const weeklyGoalColumns = "id, title, week::text, done, created_at"

// WeekStart returns the Monday of the week containing day ("YYYY-MM-DD").
func WeekStart(day string) (string, error) {
	d, err := time.Parse(dateLayout, day)
	if err != nil {
		return "", ErrInvalidWeeklyGoal
	}
	offset := (int(d.Weekday()) + 6) % 7 // Monday = 0 … Sunday = 6
	return d.AddDate(0, 0, -offset).Format(dateLayout), nil
}

// List returns the goals of the week containing day, oldest first.
func (s *WeeklyGoalService) List(ctx context.Context, userID, day string) ([]WeeklyGoalDTO, error) {
	week, err := WeekStart(day)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT `+weeklyGoalColumns+` FROM weekly_goals
		 WHERE user_id = $1 AND week = $2::date ORDER BY created_at, id`,
		userID, week)
	if err != nil {
		return nil, fmt.Errorf("listing weekly goals: %w", err)
	}
	defer rows.Close()

	goals := []WeeklyGoalDTO{}
	for rows.Next() {
		var g WeeklyGoalDTO
		if err := rows.Scan(&g.ID, &g.Title, &g.Week, &g.Done, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning weekly goal: %w", err)
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

// Create adds a goal to the week containing day.
func (s *WeeklyGoalService) Create(ctx context.Context, userID, title, day string) (*WeeklyGoalDTO, error) {
	title = strings.TrimSpace(title)
	if title == "" || len(title) > 255 {
		return nil, ErrInvalidWeeklyGoal
	}
	week, err := WeekStart(day)
	if err != nil {
		return nil, err
	}
	return s.scanOne(s.db.QueryRow(ctx,
		`INSERT INTO weekly_goals (user_id, title, week) VALUES ($1, $2, $3::date)
		 RETURNING `+weeklyGoalColumns,
		userID, title, week))
}

// SetDone marks a goal as done, or undoes it.
func (s *WeeklyGoalService) SetDone(ctx context.Context, goalID, userID string, done bool) (*WeeklyGoalDTO, error) {
	return s.scanOne(s.db.QueryRow(ctx,
		`UPDATE weekly_goals SET done = $3, updated_at = NOW()
		 WHERE id = $1 AND user_id = $2
		 RETURNING `+weeklyGoalColumns,
		goalID, userID, done))
}

// Delete removes one of the user's goals.
func (s *WeeklyGoalService) Delete(ctx context.Context, goalID, userID string) error {
	tag, err := s.db.Exec(ctx, "DELETE FROM weekly_goals WHERE id = $1 AND user_id = $2", goalID, userID)
	if err != nil {
		return fmt.Errorf("deleting weekly goal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *WeeklyGoalService) scanOne(row pgx.Row) (*WeeklyGoalDTO, error) {
	var g WeeklyGoalDTO
	if err := row.Scan(&g.ID, &g.Title, &g.Week, &g.Done, &g.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("saving weekly goal: %w", err)
	}
	return &g, nil
}
