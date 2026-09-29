-- ============================================================
-- Migration 013 — Track Day journal (web)
--
-- The free-text daily journal ("Bullet Journal" style), one entry per
-- user per day. The entry is the SOURCE of what it mentions: an expense
-- or a workout recognized in a line becomes a transaction / habit_log
-- carrying source_type='track_day' and a source_id derived from
-- (entry id, line text) — the event lives in its own module and points
-- back here; nothing is copied into this table.
-- ============================================================

CREATE TABLE IF NOT EXISTS journal_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_date  DATE NOT NULL,
    content     TEXT NOT NULL DEFAULT '' CHECK (char_length(content) <= 20000),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, entry_date)
);
