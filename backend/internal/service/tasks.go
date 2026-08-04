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
)

// ─── DTOs ────────────────────────────────────────────────────────────────────

type TaskDTO struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	Title           string          `json:"title"`
	Date            string          `json:"date"`            // "YYYY-MM-DD"
	StartTime       *string         `json:"start_time"`      // "HH:MM" or nil
	DurationMinutes *int            `json:"duration_minutes"` // nil = sem duração definida
	Category        string          `json:"category"`
	Status          string          `json:"status"`
	Priority        string          `json:"priority"`
	Notes           *string         `json:"notes"`
	TaskDetails     json.RawMessage `json:"task_details"` // JSONB — payload por categoria
	LinkedHabitID   *string         `json:"linked_habit_id"`
	Color           *string         `json:"color"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type CreateTaskInput struct {
	Title           string          `json:"title"`
	Date            string          `json:"date"`
	StartTime       *string         `json:"start_time"`
	DurationMinutes *int            `json:"duration_minutes"`
	Category        string          `json:"category"`
	Priority        string          `json:"priority"`
	Notes           *string         `json:"notes"`
	TaskDetails     json.RawMessage `json:"task_details"`
	LinkedHabitID   *string         `json:"linked_habit_id"`
	Color           *string         `json:"color"`
}

type MonthlyGoalDTO struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Month     string    `json:"month"` // "YYYY-MM-DD" (sempre dia 1)
	Status    string    `json:"status"`
	Notes     *string   `json:"notes"`
	Color     *string   `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── Service ─────────────────────────────────────────────────────────────────

type TaskService struct {
	db *pgxpool.Pool
}

func NewTaskService(db *pgxpool.Pool) *TaskService {
	return &TaskService{db: db}
}

const taskReturningColumns = `id, user_id, title, date::text, start_time::text, duration_minutes,
	category, status, priority, notes, task_details, linked_habit_id, color,
	created_at, updated_at`

// ─── Tasks CRUD ───────────────────────────────────────────────────────────────

// ListByDate retorna todas as tarefas de um usuário em uma data específica,
// ordenadas pelo horário de início (tarefas sem horário vêm por último).
func (s *TaskService) ListByDate(ctx context.Context, userID, date string) ([]TaskDTO, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+taskReturningColumns+`
		FROM tasks
		WHERE user_id = $1 AND date = $2::date
		ORDER BY start_time ASC NULLS LAST, created_at ASC`,
		userID, date,
	)
	if err != nil {
		return nil, fmt.Errorf("listing tasks by date: %w", err)
	}
	defer rows.Close()

	return scanTasks(rows)
}

// ListByWeek retorna todas as tarefas de uma semana (date_start até date_end).
func (s *TaskService) ListByWeek(ctx context.Context, userID, dateStart, dateEnd string) ([]TaskDTO, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+taskReturningColumns+`
		FROM tasks
		WHERE user_id = $1 AND date BETWEEN $2::date AND $3::date
		ORDER BY date ASC, start_time ASC NULLS LAST`,
		userID, dateStart, dateEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("listing tasks by week: %w", err)
	}
	defer rows.Close()

	return scanTasks(rows)
}

// Create cria uma nova tarefa.
func (s *TaskService) Create(ctx context.Context, userID string, input CreateTaskInput) (*TaskDTO, error) {
	if input.Category == "" {
		input.Category = "other"
	}
	if input.Priority == "" {
		input.Priority = "medium"
	}

	// Converte start_time "HH:MM" → aceito pelo PostgreSQL como TIME
	var startTime interface{} = nil
	if input.StartTime != nil && *input.StartTime != "" {
		startTime = normalizeTimeString(*input.StartTime)
	}

	var details interface{} = nil
	if len(input.TaskDetails) > 0 && string(input.TaskDetails) != "null" {
		details = []byte(input.TaskDetails)
	}

	row := s.db.QueryRow(ctx, `
		INSERT INTO tasks (user_id, title, date, start_time, duration_minutes,
		                   category, priority, notes, task_details, linked_habit_id, color)
		VALUES ($1, $2, $3::date, $4::time, $5, $6, $7, $8, $9, $10, $11)
		RETURNING `+taskReturningColumns,
		userID, input.Title, input.Date, startTime, input.DurationMinutes,
		input.Category, input.Priority, input.Notes, details, input.LinkedHabitID, input.Color,
	)

	return scanTask(row.Scan)
}

// taskUpdatableColumns maps JSON field names to their SQL column and cast.
// Presence in the request body decides what gets updated, so explicit null
// clears a nullable column (unlike COALESCE-based updates).
var taskUpdatableColumns = map[string]string{
	"title":            "title",
	"date":             "date",
	"start_time":       "start_time",
	"duration_minutes": "duration_minutes",
	"category":         "category",
	"status":           "status",
	"priority":         "priority",
	"notes":            "notes",
	"task_details":     "task_details",
	"linked_habit_id":  "linked_habit_id",
	"color":            "color",
}

var taskColumnCasts = map[string]string{
	"date":       "::date",
	"start_time": "::time",
}

// Update atualiza os campos presentes no body (PATCH semântico real):
// campos ausentes são mantidos, campos com null explícito são limpos.
func (s *TaskService) Update(ctx context.Context, taskID, userID string, fields map[string]json.RawMessage) (*TaskDTO, error) {
	setClauses := []string{"updated_at = now()"}
	args := []any{taskID, userID}

	for field, raw := range fields {
		col, ok := taskUpdatableColumns[field]
		if !ok {
			continue
		}

		var value any
		if string(raw) == "null" {
			value = nil
		} else {
			switch field {
			case "duration_minutes":
				var v int
				if err := json.Unmarshal(raw, &v); err != nil {
					return nil, fmt.Errorf("invalid %s: %w", field, err)
				}
				value = v
			case "task_details":
				value = []byte(raw)
			default:
				var v string
				if err := json.Unmarshal(raw, &v); err != nil {
					return nil, fmt.Errorf("invalid %s: %w", field, err)
				}
				if field == "start_time" {
					if v == "" {
						value = nil
						break
					}
					v = normalizeTimeString(v)
				}
				value = v
			}
		}

		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d%s", col, len(args), taskColumnCasts[field]))
	}

	row := s.db.QueryRow(ctx, `
		UPDATE tasks SET `+strings.Join(setClauses, ", ")+`
		WHERE id = $1 AND user_id = $2
		RETURNING `+taskReturningColumns,
		args...,
	)

	return scanTask(row.Scan)
}

// AdvanceStatus avança o status da tarefa no fluxo:
// planned → in_progress → done → reviewed.
// Ao atingir 'done', uma tarefa com linked_habit_id gera o check-in do hábito
// referenciando a tarefa como origem (source_type='task') — nunca sobrescreve
// um check-in manual já existente (fonte única de verdade).
func (s *TaskService) AdvanceStatus(ctx context.Context, taskID, userID string) (*TaskDTO, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	row := tx.QueryRow(ctx, `
		UPDATE tasks SET
		  status = CASE status
		    WHEN 'planned'     THEN 'in_progress'
		    WHEN 'in_progress' THEN 'done'
		    WHEN 'done'        THEN 'reviewed'
		    ELSE status
		  END,
		  updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING `+taskReturningColumns,
		taskID, userID,
	)

	t, err := scanTask(row.Scan)
	if err != nil {
		return nil, err
	}

	if t.Status == "done" && t.LinkedHabitID != nil {
		_, err = tx.Exec(ctx, `
			INSERT INTO habit_logs (habit_id, user_id, logged_date, source_type, source_id)
			SELECT id, user_id, $3::date, 'task', $4
			FROM habits WHERE id = $1 AND user_id = $2 AND is_active = true
			ON CONFLICT (habit_id, logged_date) DO NOTHING`,
			*t.LinkedHabitID, userID, t.Date, t.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("linked habit check-in: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing tx: %w", err)
	}
	return t, nil
}

// Delete remove uma tarefa.
func (s *TaskService) Delete(ctx context.Context, taskID, userID string) error {
	tag, err := s.db.Exec(ctx,
		"DELETE FROM tasks WHERE id = $1 AND user_id = $2",
		taskID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Monthly Goals CRUD ────────────────────────────────────────────────────────

// ListGoalsByMonth retorna as metas de um mês.
func (s *TaskService) ListGoalsByMonth(ctx context.Context, userID, month string) ([]MonthlyGoalDTO, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, title, month::text, status, notes, color, created_at
		FROM monthly_goals
		WHERE user_id = $1 AND month = date_trunc('month', $2::date)
		ORDER BY created_at ASC`,
		userID, month,
	)
	if err != nil {
		return nil, fmt.Errorf("listing monthly goals: %w", err)
	}
	defer rows.Close()

	var goals []MonthlyGoalDTO
	for rows.Next() {
		var g MonthlyGoalDTO
		if err := rows.Scan(&g.ID, &g.UserID, &g.Title, &g.Month,
			&g.Status, &g.Notes, &g.Color, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	if goals == nil {
		goals = []MonthlyGoalDTO{}
	}
	return goals, nil
}

// CreateGoal cria uma meta mensal.
func (s *TaskService) CreateGoal(ctx context.Context, userID, title, month string, notes, color *string) (*MonthlyGoalDTO, error) {
	var g MonthlyGoalDTO
	err := s.db.QueryRow(ctx, `
		INSERT INTO monthly_goals (user_id, title, month, notes, color)
		VALUES ($1, $2, date_trunc('month', $3::date), $4, $5)
		RETURNING id, user_id, title, month::text, status, notes, color, created_at`,
		userID, title, month, notes, color,
	).Scan(&g.ID, &g.UserID, &g.Title, &g.Month, &g.Status, &g.Notes, &g.Color, &g.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating monthly goal: %w", err)
	}
	return &g, nil
}

// UpdateGoalStatus atualiza o status de uma meta.
func (s *TaskService) UpdateGoalStatus(ctx context.Context, goalID, userID, status string) (*MonthlyGoalDTO, error) {
	var g MonthlyGoalDTO
	err := s.db.QueryRow(ctx, `
		UPDATE monthly_goals SET status = $3
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, title, month::text, status, notes, color, created_at`,
		goalID, userID, status,
	).Scan(&g.ID, &g.UserID, &g.Title, &g.Month, &g.Status, &g.Notes, &g.Color, &g.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("updating goal status: %w", err)
	}
	return &g, nil
}

// DeleteGoal remove uma meta mensal.
func (s *TaskService) DeleteGoal(ctx context.Context, goalID, userID string) error {
	tag, err := s.db.Exec(ctx,
		"DELETE FROM monthly_goals WHERE id = $1 AND user_id = $2",
		goalID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Scan helpers ─────────────────────────────────────────────────────────────

// normalizeTimeString accepts "HH:MM" or "HH:MM:SS" and returns "HH:MM:SS".
func normalizeTimeString(t string) string {
	if len(t) == 5 {
		return t + ":00"
	}
	return t
}

func scanTask(scan func(...any) error) (*TaskDTO, error) {
	var t TaskDTO
	var details []byte
	if err := scan(
		&t.ID, &t.UserID, &t.Title, &t.Date,
		&t.StartTime, &t.DurationMinutes,
		&t.Category, &t.Status, &t.Priority,
		&t.Notes, &details, &t.LinkedHabitID, &t.Color,
		&t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scanning task: %w", err)
	}
	if len(details) > 0 {
		t.TaskDetails = json.RawMessage(details)
	}
	if t.StartTime != nil && len(*t.StartTime) >= 5 {
		trimmed := (*t.StartTime)[:5]
		t.StartTime = &trimmed
	}
	return &t, nil
}

func scanTasks(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]TaskDTO, error) {
	var tasks []TaskDTO
	for rows.Next() {
		var t TaskDTO
		var details []byte
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Title, &t.Date,
			&t.StartTime, &t.DurationMinutes,
			&t.Category, &t.Status, &t.Priority,
			&t.Notes, &details, &t.LinkedHabitID, &t.Color,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if len(details) > 0 {
			t.TaskDetails = json.RawMessage(details)
		}
		if t.StartTime != nil && len(*t.StartTime) >= 5 {
			trimmed := (*t.StartTime)[:5]
			t.StartTime = &trimmed
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if tasks == nil {
		tasks = []TaskDTO{}
	}
	return tasks, nil
}
