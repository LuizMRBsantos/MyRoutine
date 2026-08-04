-- ============================================================
-- Migration 006 — Cross-module references (single source of truth)
-- Implements .agents/skills/cross-module-data-flow: events created by
-- other modules (tasks, track day, study sessions) reference their
-- origin instead of being duplicated.
-- ============================================================

ALTER TABLE habit_logs
  ADD COLUMN IF NOT EXISTS source_type VARCHAR(20) NOT NULL DEFAULT 'manual',
  ADD COLUMN IF NOT EXISTS source_id   UUID;

COMMENT ON COLUMN habit_logs.source_type IS 'Origin of the check-in: manual | task | track_day | study_session | import';
COMMENT ON COLUMN habit_logs.source_id   IS 'ID of the originating record in the source module (unique per source_type)';

-- Makes pushes from other modules idempotent: the same source event can
-- never create two logs.
CREATE UNIQUE INDEX IF NOT EXISTS idx_habit_logs_source
  ON habit_logs (source_type, source_id) WHERE source_id IS NOT NULL;

-- Consumer modules (Health, Studies) query habits by category instead of
-- keeping their own copies of events.
ALTER TABLE habits
  ADD COLUMN IF NOT EXISTS category VARCHAR(20) NOT NULL DEFAULT 'general';

COMMENT ON COLUMN habits.category IS 'Module grouping: general | health | study';

CREATE INDEX IF NOT EXISTS idx_habits_user_category ON habits (user_id, category);
