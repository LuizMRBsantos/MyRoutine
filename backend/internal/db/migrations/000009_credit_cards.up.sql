-- ============================================================
-- Migration 009 — Credit cards and installments
--
-- occurred_on keeps meaning "when the money leaves" — for a card
-- purchase that is the due date of the bill it landed in, so the
-- monthly summary stays a cash-flow view. purchased_on records
-- when the purchase actually happened.
--
-- Each installment is its own transaction row: they are distinct
-- cash events sharing an origin, tied together by
-- installment_group_id. This keeps every existing query (summary,
-- budgets, category chart) working untouched and makes future
-- commitments visible.
-- ============================================================

CREATE TABLE IF NOT EXISTS credit_cards (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    closing_day SMALLINT NOT NULL CHECK (closing_day BETWEEN 1 AND 31),
    due_day     SMALLINT NOT NULL CHECK (due_day BETWEEN 1 AND 31),
    color       VARCHAR(7),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN credit_cards.closing_day IS 'Dia do fechamento da fatura (1-31, ajustado para meses curtos)';
COMMENT ON COLUMN credit_cards.due_day     IS 'Dia do vencimento da fatura (1-31, ajustado para meses curtos)';

CREATE INDEX IF NOT EXISTS idx_credit_cards_user ON credit_cards (user_id, is_active);

DROP TRIGGER IF EXISTS set_credit_cards_updated_at ON credit_cards;
CREATE TRIGGER set_credit_cards_updated_at
    BEFORE UPDATE ON credit_cards
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Todas nullable: lançamentos em pix/débito seguem funcionando sem mudança.
ALTER TABLE transactions
  ADD COLUMN IF NOT EXISTS credit_card_id       UUID REFERENCES credit_cards(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS purchased_on         DATE,
  ADD COLUMN IF NOT EXISTS installment_group_id UUID,
  ADD COLUMN IF NOT EXISTS installment_number   SMALLINT,
  ADD COLUMN IF NOT EXISTS installment_total    SMALLINT;

COMMENT ON COLUMN transactions.purchased_on IS 'Data da compra. occurred_on guarda a data de cobrança (vencimento da fatura)';
COMMENT ON COLUMN transactions.installment_group_id IS 'Agrupa as parcelas de uma mesma compra';

CREATE INDEX IF NOT EXISTS idx_transactions_group
  ON transactions (installment_group_id) WHERE installment_group_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_transactions_card
  ON transactions (user_id, credit_card_id) WHERE credit_card_id IS NOT NULL;
