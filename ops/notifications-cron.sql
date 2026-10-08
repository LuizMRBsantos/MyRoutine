-- One-off setup, run in the Supabase SQL Editor (NOT a migration: pg_cron,
-- pg_net and Vault are Supabase services, and the local/CI Postgres has none).
--
-- Every 5 minutes Supabase calls the notification dispatcher on Vercel. The
-- Vercel Hobby cron only runs once a day, too slow for "15 min before".
-- Replace <CRON_SECRET> with the value of CRON_SECRET on Vercel. It lives
-- in Vault (encrypted), so it never appears in cron.job.
--
-- Safe to run again: it updates the secret and replaces the job.

CREATE EXTENSION IF NOT EXISTS pg_cron;
CREATE EXTENSION IF NOT EXISTS pg_net;

DO $$
DECLARE
  existing uuid;
BEGIN
  SELECT id INTO existing FROM vault.secrets WHERE name = 'myroutine_cron_secret';
  IF existing IS NULL THEN
    PERFORM vault.create_secret('<CRON_SECRET>', 'myroutine_cron_secret', 'MyRoutine notification dispatch');
  ELSE
    PERFORM vault.update_secret(existing, '<CRON_SECRET>');
  END IF;
END $$;

SELECT cron.schedule(
  'myroutine-notifications',
  '*/5 * * * *',
  $job$
  SELECT net.http_post(
    url := 'https://myroutine-eight.vercel.app/api/v1/internal/notifications/dispatch',
    headers := jsonb_build_object(
      'Content-Type', 'application/json',
      'Authorization', 'Bearer ' || (SELECT decrypted_secret FROM vault.decrypted_secrets WHERE name = 'myroutine_cron_secret')
    ),
    body := '{}'::jsonb,
    timeout_milliseconds := 30000
  );
  $job$
);

-- Check: the job exists, and (after ~10 min) the last calls answered 200.
SELECT jobname, schedule, active FROM cron.job WHERE jobname = 'myroutine-notifications';
-- SELECT status_code, left(content, 120), created FROM net._http_response ORDER BY created DESC LIMIT 5;

-- To stop: SELECT cron.unschedule('myroutine-notifications');
