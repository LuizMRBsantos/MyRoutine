-- "Me avisar antes" per task. On by default for appointments, exams and work
-- (meetings); off for classes, exercise, waking up and "other", which mostly
-- fill the calendar. The app sets it on create; this backfills existing tasks
-- with the same rule.
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS notify BOOLEAN NOT NULL DEFAULT true;
UPDATE tasks SET notify = false WHERE category NOT IN ('appointment', 'exam', 'work');
