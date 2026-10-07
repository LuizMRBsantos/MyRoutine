-- ============================================================
-- Migration 014 — Lock every table against Supabase's auto API
--
-- Supabase publishes a REST API (PostgREST) for the public schema,
-- reachable with the public "publishable/anon" key that ships in any
-- Supabase client. MyRoutine never uses it: all access goes through
-- the Go API, which connects as the tables' owner.
--
-- Defense in depth (the Data API is also switched off in the
-- dashboard):
--   1. RLS ON in every table, with NO policies: the API's roles see
--      nothing. The owner (our backend) is not subject to RLS, so the
--      app is unaffected. (Not FORCE — that would bind the owner too.)
--   2. Revoke table privileges from Supabase's API roles, when they
--      exist (they don't in local Docker/test databases).
--
-- Every NEW table must enable RLS in its own migration;
-- TestEveryTableHasRowLevelSecurity enforces it.
-- ============================================================

DO $$
DECLARE
    t record;
BEGIN
    FOR t IN
        SELECT c.relname
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p')
    LOOP
        EXECUTE format('ALTER TABLE public.%I ENABLE ROW LEVEL SECURITY', t.relname);
    END LOOP;
END $$;

DO $$
DECLARE
    r text;
BEGIN
    FOREACH r IN ARRAY ARRAY['anon', 'authenticated'] LOOP
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = r) THEN
            EXECUTE format('REVOKE ALL ON ALL TABLES IN SCHEMA public FROM %I', r);
            EXECUTE format('REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM %I', r);
            EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON TABLES FROM %I', r);
            EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE ALL ON SEQUENCES FROM %I', r);
        END IF;
    END LOOP;
END $$;
