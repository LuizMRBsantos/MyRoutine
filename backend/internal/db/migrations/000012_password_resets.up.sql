-- ============================================================
-- Migration 012 — Admin-issued password reset links
--
-- The app sends no email, so "forgot my password" works like invites:
-- the user asks the admin, the admin generates a reset link and sends
-- it personally. Only the SHA-256 hash of the code is stored.
--
-- A reset link opens someone's account, so it lives 1 hour, works
-- once, and issuing a new one invalidates the previous (see service).
-- ============================================================

CREATE TABLE IF NOT EXISTS password_resets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(64) NOT NULL UNIQUE,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_resets_user ON password_resets (user_id, created_at DESC);
