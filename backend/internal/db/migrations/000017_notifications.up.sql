-- ============================================================
-- Migration 017 — Push notifications (Web Push)
--
-- push_subscriptions: one row per device where the person turned
--   notifications on (the browser's PushSubscription). The endpoint is
--   unique: when someone else logs in on the same device, the row
--   moves to them.
-- notification_settings: what to send and when (local time), with the
--   product defaults — appointment reminders 15 min before, morning
--   digest at 07:00 ON; evening digest at 21:00 OFF.
-- notification_deliveries: what was already sent, so a dispatch that
--   runs again never sends the same notification twice.
-- ============================================================

CREATE TABLE IF NOT EXISTS push_subscriptions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint      TEXT NOT NULL UNIQUE CHECK (endpoint LIKE 'https://%' AND char_length(endpoint) <= 1024),
    p256dh        TEXT NOT NULL CHECK (char_length(p256dh) <= 256),
    auth          TEXT NOT NULL CHECK (char_length(auth) <= 128),
    user_agent    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user ON push_subscriptions (user_id);

CREATE TABLE IF NOT EXISTS notification_settings (
    user_id            UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    task_reminders     BOOLEAN NOT NULL DEFAULT true,
    task_lead_minutes  INTEGER NOT NULL DEFAULT 15 CHECK (task_lead_minutes BETWEEN 5 AND 240),
    morning_digest     BOOLEAN NOT NULL DEFAULT true,
    morning_time       TIME    NOT NULL DEFAULT '07:00',
    evening_digest     BOOLEAN NOT NULL DEFAULT false,
    evening_time       TIME    NOT NULL DEFAULT '21:00',
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notification_deliveries (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind      VARCHAR(30) NOT NULL, -- task_reminder | morning_digest | evening_digest
    ref       VARCHAR(100) NOT NULL, -- task id + date, or the local date of a digest
    sent_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, kind, ref)
);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_sent ON notification_deliveries (sent_at);

-- Same lock as every other table (migration 014).
ALTER TABLE push_subscriptions      ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_settings   ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_deliveries ENABLE ROW LEVEL SECURITY;
