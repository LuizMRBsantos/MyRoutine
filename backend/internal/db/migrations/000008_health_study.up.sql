-- ============================================================
-- Migration 008 — Health and Study modules
--
-- Health does NOT get its own events table: activities are read
-- from habit_logs of habits with category='health' (single source
-- of truth). body_metrics is health's own data — measurements are
-- not habit events.
--
-- Study sessions ARE their own source of truth; a session with a
-- linked habit creates a habit_log referencing it (source_type=
-- 'study_session'), never a copy.
-- ============================================================

CREATE TABLE IF NOT EXISTS body_metrics (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    measured_on DATE NOT NULL,
    weight_kg   NUMERIC(5,2),
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, measured_on)
);

CREATE INDEX IF NOT EXISTS idx_body_metrics_user_date ON body_metrics (user_id, measured_on DESC);

CREATE TABLE IF NOT EXISTS study_sessions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject          VARCHAR(100) NOT NULL,
    topic            VARCHAR(255),
    studied_on       DATE NOT NULL,
    duration_minutes INT NOT NULL CHECK (duration_minutes > 0),
    notes            TEXT,
    task_id          UUID REFERENCES tasks(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_study_sessions_user_date ON study_sessions (user_id, studied_on DESC);
