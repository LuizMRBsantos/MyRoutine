-- ============================================================
-- Migration 003 — Revisão consciente de dias perdidos (BuJo-style)
-- MyRoutine — Criado em: 2026-07-03
-- ============================================================

CREATE TABLE IF NOT EXISTS habit_day_reviews (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id     UUID        NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    review_date  DATE        NOT NULL,
    status       VARCHAR(20) NOT NULL,
    reviewed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT habit_day_reviews_status_check
        CHECK (status IN ('migrated', 'discarded')),

    -- Um review por hábito por dia
    UNIQUE(habit_id, review_date)
);

CREATE INDEX IF NOT EXISTS idx_habit_day_reviews_user ON habit_day_reviews(user_id, review_date DESC);
CREATE INDEX IF NOT EXISTS idx_habit_day_reviews_habit ON habit_day_reviews(habit_id, review_date);

COMMENT ON TABLE habit_day_reviews IS 'Registro de decisões conscientes sobre dias sem check-in (espírito Bullet Journal)';
COMMENT ON COLUMN habit_day_reviews.status IS 'migrated = intenção de fazer na próxima semana | discarded = descartado conscientemente';
