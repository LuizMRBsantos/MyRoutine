-- ============================================================
-- Migration 002 — Adicionar time_of_day nos hábitos
-- MyRoutine — Criado em: 2026-07-03
-- ============================================================

ALTER TABLE habits
ADD COLUMN IF NOT EXISTS time_of_day VARCHAR(10) NOT NULL DEFAULT 'anytime'
  CONSTRAINT habits_time_of_day_check
  CHECK (time_of_day IN ('morning', 'afternoon', 'evening', 'anytime'));

COMMENT ON COLUMN habits.time_of_day IS 'Período do dia para agrupar hábitos no dashboard (morning/afternoon/evening/anytime)';
