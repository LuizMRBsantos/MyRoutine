-- ============================================================
-- Migration 016 — Auth rate limits shared by every instance
--
-- On Vercel several copies of the API run at once, so an in-memory
-- counter only limits per copy. The counter lives here instead: one row
-- per client key with the attempts in the current minute, bumped and
-- read back in a single statement (no race between copies).
--
-- key_hash is SHA-256 of the client IP — the IP itself (personal data)
-- is never stored. Rows older than an hour are swept by the app.
-- ============================================================

CREATE TABLE IF NOT EXISTS auth_rate_limits (
    key_hash      VARCHAR(64) PRIMARY KEY,
    window_start  TIMESTAMPTZ NOT NULL,
    attempts      INTEGER     NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_rate_limits_window ON auth_rate_limits (window_start);

-- Same lock as every other table (migration 014).
ALTER TABLE auth_rate_limits ENABLE ROW LEVEL SECURITY;
