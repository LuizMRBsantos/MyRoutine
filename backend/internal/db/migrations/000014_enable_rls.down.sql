-- Privileges revoked from Supabase's API roles are not re-granted: the app
-- never used them. Only RLS is switched back off.
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
        EXECUTE format('ALTER TABLE public.%I DISABLE ROW LEVEL SECURITY', t.relname);
    END LOOP;
END $$;
