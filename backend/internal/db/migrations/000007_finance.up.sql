-- ============================================================
-- Migration 007 — Finance module: transactions and budgets
-- Money is stored as integer cents (never float). Rows pushed
-- from other modules carry source_type/source_id; the partial
-- unique index makes those pushes idempotent.
-- ============================================================

CREATE TABLE IF NOT EXISTS transactions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    kind         VARCHAR(10) NOT NULL DEFAULT 'expense',  -- 'expense' | 'income'
    category     VARCHAR(50) NOT NULL DEFAULT 'other',
    description  VARCHAR(255) NOT NULL,
    method       VARCHAR(30),                             -- pix | credit | debit | cash | ...
    occurred_on  DATE NOT NULL,
    source_type  VARCHAR(20) NOT NULL DEFAULT 'manual',   -- 'manual' | 'track_day' | 'import'
    source_id    UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_date ON transactions (user_id, occurred_on DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_user_category ON transactions (user_id, category);
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_source
  ON transactions (source_type, source_id) WHERE source_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS budgets (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category     VARCHAR(50) NOT NULL,
    month        DATE NOT NULL,                           -- sempre dia 1
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, category, month)
);
