-- Weekly goals: simple "done or not" goals for one week (Monday to Sunday),
-- shown on the Dashboard. Independent of monthly goals.
CREATE TABLE IF NOT EXISTS weekly_goals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL CHECK (char_length(btrim(title)) > 0),
    week        DATE NOT NULL CHECK (EXTRACT(ISODOW FROM week) = 1), -- that week's Monday
    done        BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_weekly_goals_user_week ON weekly_goals (user_id, week);

-- Same lock as every other table (migration 014).
ALTER TABLE weekly_goals ENABLE ROW LEVEL SECURITY;
