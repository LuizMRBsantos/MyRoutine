-- ============================================================
-- Migration 011 — Invite-only registration
--
-- The beta is for a handful of invited people, so an account can only
-- be created from an invite link. The admin generates the link and
-- sends it personally (email/WhatsApp); the app sends no email.
--
-- Like refresh tokens, the invite code is never stored in clear: only
-- its SHA-256 hash, so reading the table does not yield usable links.
-- An invite is bound to one email and works once, until it expires.
-- ============================================================

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS invites (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash  VARCHAR(64) NOT NULL UNIQUE,
    -- Stored lowercased; registration must use this exact email.
    email       VARCHAR(255) NOT NULL,
    invited_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    used_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_invites_created ON invites (created_at DESC);
