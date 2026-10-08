-- ============================================================
-- Migration 015 — Built-in UUID defaults
--
-- The tables from migration 001 defaulted their ids to
-- uuid_generate_v4() (extension uuid-ossp). On Supabase that function
-- lives in the "extensions" schema, so a backup of schema public could
-- not be restored into a plain Postgres: those tables (and everything
-- referencing them) failed to recreate. gen_random_uuid() is built into
-- Postgres 13+ — the other tables already use it.
--
-- Only the DEFAULT changes: existing ids are untouched. The extension is
-- left in place (dropping it is a separate, explicit decision).
-- ============================================================

ALTER TABLE users          ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE refresh_tokens ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE habits         ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE habit_logs     ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE audit_logs     ALTER COLUMN id SET DEFAULT gen_random_uuid();
