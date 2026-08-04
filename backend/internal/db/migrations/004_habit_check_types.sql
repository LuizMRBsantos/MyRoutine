-- ============================================================
-- Migration 004 — Tipos de check-in avançados
-- MyRoutine — Criado em: 2026-07-18
-- ============================================================

-- Tipo de check-in do hábito
CREATE TYPE habit_check_type AS ENUM ('simple', 'timed', 'deadline', 'metric');

-- ────────────────────────────────────────────────────────────
-- HABITS — novos campos de configuração por tipo
-- ────────────────────────────────────────────────────────────
ALTER TABLE habits
  ADD COLUMN IF NOT EXISTS check_type     habit_check_type NOT NULL DEFAULT 'simple',
  ADD COLUMN IF NOT EXISTS timer_minutes  INTEGER,
  ADD COLUMN IF NOT EXISTS deadline_time  TIME,
  ADD COLUMN IF NOT EXISTS metric_config  JSONB;

COMMENT ON COLUMN habits.check_type    IS 'Tipo de check-in: simple (toggle), timed (timer obrigatório), deadline (horário limite), metric (registrar valores)';
COMMENT ON COLUMN habits.timer_minutes IS 'Para check_type=timed: duração mínima em minutos para o check-in ser válido';
COMMENT ON COLUMN habits.deadline_time IS 'Para check_type=deadline: horário limite para dar check (ex: 05:30)';
COMMENT ON COLUMN habits.metric_config IS 'Para check_type=metric: definição dos campos. Ex: [{"key":"km","label":"Quilômetros","unit":"km"},{"key":"calories","label":"Calorias","unit":"kcal"}]';

-- Constraints para garantir dados consistentes
ALTER TABLE habits
  ADD CONSTRAINT habits_timer_minutes_check
    CHECK (check_type != 'timed' OR timer_minutes IS NOT NULL AND timer_minutes > 0),
  ADD CONSTRAINT habits_deadline_time_check
    CHECK (check_type != 'deadline' OR deadline_time IS NOT NULL),
  ADD CONSTRAINT habits_metric_config_check
    CHECK (check_type != 'metric' OR metric_config IS NOT NULL AND jsonb_array_length(metric_config) > 0);

-- ────────────────────────────────────────────────────────────
-- HABIT_LOGS — dados do check-in enriquecido
-- ────────────────────────────────────────────────────────────
ALTER TABLE habit_logs
  ADD COLUMN IF NOT EXISTS timer_seconds  INTEGER,
  ADD COLUMN IF NOT EXISTS started_at     TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS completed_at   TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS metrics        JSONB,
  ADD COLUMN IF NOT EXISTS is_manual      BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN habit_logs.timer_seconds IS 'Tempo total gasto em segundos (para hábitos timed)';
COMMENT ON COLUMN habit_logs.started_at    IS 'Quando o timer foi iniciado';
COMMENT ON COLUMN habit_logs.completed_at  IS 'Quando o timer foi concluído';
COMMENT ON COLUMN habit_logs.metrics       IS 'Valores registrados. Ex: {"km": 5.2, "calories": 320}';
COMMENT ON COLUMN habit_logs.is_manual     IS 'true se o usuário registrou manualmente (esqueceu de ligar o timer)';
