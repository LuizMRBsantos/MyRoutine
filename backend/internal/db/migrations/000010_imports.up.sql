-- ============================================================
-- Migration 010 — Statement import and reconciliation
--
-- A statement line you have not reviewed is NOT a fact about your
-- money: it is a proposal. So imported rows land in a staging area
-- and never touch `transactions` until approved — otherwise they
-- would already skew the summary, budgets and charts.
--
-- On approval the entry becomes a transaction carrying
-- source_type='import' and source_id = the entry id, reusing the
-- unique partial index that already guards against duplicates.
-- ============================================================

CREATE TABLE IF NOT EXISTS import_batches (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename       VARCHAR(255) NOT NULL,
    -- Extrato de cartão fica ligado ao cartão; extrato de conta usa o rótulo.
    credit_card_id UUID REFERENCES credit_cards(id) ON DELETE SET NULL,
    account_label  VARCHAR(100),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_import_batches_user ON import_batches (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS import_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id     UUID NOT NULL REFERENCES import_batches(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    occurred_on     DATE NOT NULL,
    amount_cents    BIGINT NOT NULL CHECK (amount_cents > 0),
    kind            VARCHAR(10) NOT NULL DEFAULT 'expense',  -- 'expense' | 'income'
    raw_description VARCHAR(500) NOT NULL,

    -- Preenchidos pela conciliação, revisáveis pelo usuário
    suggested_category     VARCHAR(50),
    matched_transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,

    -- pending: esperando decisão | imported: virou transação
    -- ignored: descartada conscientemente | matched: era duplicata de algo que já existia
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
      CHECK (status IN ('pending', 'imported', 'ignored', 'matched')),

    -- Impede que reimportar o mesmo arquivo traga as linhas de novo.
    fingerprint TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_import_entries_batch ON import_entries (batch_id);
CREATE INDEX IF NOT EXISTS idx_import_entries_pending
  ON import_entries (user_id, status) WHERE status = 'pending';
CREATE UNIQUE INDEX IF NOT EXISTS idx_import_entries_fingerprint
  ON import_entries (user_id, fingerprint);

-- Aprender com o que você categoriza: na próxima importação a sugestão vem pronta.
CREATE TABLE IF NOT EXISTS category_rules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pattern    VARCHAR(200) NOT NULL,   -- trecho procurado na descrição, sem acento e em caixa alta
    category   VARCHAR(50) NOT NULL,
    hit_count  INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, pattern)
);

CREATE INDEX IF NOT EXISTS idx_category_rules_user ON category_rules (user_id);
