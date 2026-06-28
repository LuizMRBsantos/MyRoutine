-- ============================================================
-- SQL Queries para o sqlc gerar código Go tipado
-- Arquivo: backend/internal/db/queries/habits.sql
-- ============================================================

-- name: CreateHabit :one
INSERT INTO habits (user_id, name, description, icon, color, frequency, target_days, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetHabitByID :one
SELECT * FROM habits
WHERE id = $1 AND user_id = $2 AND is_active = true
LIMIT 1;

-- name: ListHabitsByUser :many
SELECT * FROM habits
WHERE user_id = $1 AND is_active = true
ORDER BY sort_order ASC, created_at ASC;

-- name: UpdateHabit :one
UPDATE habits
SET name = $3, description = $4, icon = $5, color = $6,
    frequency = $7, target_days = $8
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: SoftDeleteHabit :exec
UPDATE habits SET is_active = false
WHERE id = $1 AND user_id = $2;

-- name: CreateHabitLog :one
INSERT INTO habit_logs (habit_id, user_id, logged_date, notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (habit_id, logged_date)
DO UPDATE SET notes = EXCLUDED.notes
RETURNING *;

-- name: GetHabitLog :one
SELECT * FROM habit_logs
WHERE habit_id = $1 AND logged_date = $2
LIMIT 1;

-- name: DeleteHabitLog :exec
DELETE FROM habit_logs
WHERE habit_id = $1 AND user_id = $2 AND logged_date = $3;

-- name: GetHabitLogsInRange :many
SELECT * FROM habit_logs
WHERE habit_id = $1 AND user_id = $2
  AND logged_date BETWEEN $3 AND $4
ORDER BY logged_date DESC;

-- name: GetUserLogsForHeatmap :many
SELECT logged_date, COUNT(*) as habits_completed
FROM habit_logs
WHERE user_id = $1
  AND logged_date >= CURRENT_DATE - INTERVAL '365 days'
GROUP BY logged_date
ORDER BY logged_date ASC;

-- name: GetHabitStreak :one
-- Calcula streak atual (dias consecutivos) para um hábito
WITH RECURSIVE streak AS (
    SELECT logged_date, 1 AS streak_count
    FROM habit_logs
    WHERE habit_id = $1
      AND logged_date = CURRENT_DATE
    UNION ALL
    SELECT h.logged_date, s.streak_count + 1
    FROM habit_logs h
    JOIN streak s ON h.logged_date = s.logged_date - INTERVAL '1 day'
    WHERE h.habit_id = $1
)
SELECT COALESCE(MAX(streak_count), 0)::INTEGER AS current_streak
FROM streak;
