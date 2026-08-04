-- Migration 005: Tasks and Monthly Goals
-- Módulo de Planejamento Diário, Semanal e Mensal

-- ─── Tasks ──────────────────────────────────────────────────────────────────
-- Tarefas/eventos planejados com horário, duração e tipo específico.
-- task_details (JSONB) carrega payload por categoria sem precisar de novas colunas.

CREATE TABLE tasks (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  title            VARCHAR(255) NOT NULL,
  date             DATE NOT NULL,          -- dia a que a tarefa pertence
  start_time       TIME,                   -- ex: "05:30" (opcional)
  duration_minutes INT,                    -- ex: 90 minutos (opcional)

  category         VARCHAR(50) NOT NULL DEFAULT 'other',
  -- valores: 'wake_up' | 'exercise' | 'study' | 'work' | 'exam' | 'appointment' | 'other'

  status           VARCHAR(20) NOT NULL DEFAULT 'planned',
  -- valores: 'planned' | 'in_progress' | 'done' | 'reviewed'

  priority         VARCHAR(10) NOT NULL DEFAULT 'medium',
  -- valores: 'high' | 'medium' | 'low'

  notes            TEXT,

  -- Payload específico por categoria (nullable).
  -- exercise : { "workout_type": "corrida", "plan": "...", "km": 5, "time_min": 42, "calories": 380 }
  -- study    : { "subject": "OAC", "topic": "...", "exercises_count": 10 }
  -- work     : { "project": "MyRoutine", "deliverables": ["..."] }
  -- exam     : { "subject": "BD", "location": "Sala 201", "score": null }
  task_details     JSONB,

  -- Vínculo opcional com um hábito (check-in automático ao concluir)
  linked_habit_id  UUID REFERENCES habits(id) ON DELETE SET NULL,

  color            VARCHAR(7),
  created_at       TIMESTAMP NOT NULL DEFAULT now(),
  updated_at       TIMESTAMP NOT NULL DEFAULT now()
);

-- Índice principal: busca por usuário + data (diária/semanal)
CREATE INDEX idx_tasks_user_date ON tasks (user_id, date);
-- Índice para busca por status
CREATE INDEX idx_tasks_user_status ON tasks (user_id, status);

-- ─── Monthly Goals ──────────────────────────────────────────────────────────
-- Metas mensais — sem horário específico, ciclo de revisão mensal.

CREATE TABLE monthly_goals (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  title      VARCHAR(255) NOT NULL,
  month      DATE NOT NULL,   -- sempre o 1º dia do mês (ex: 2026-07-01)

  status     VARCHAR(20) NOT NULL DEFAULT 'active',
  -- valores: 'active' | 'done' | 'abandoned'

  notes      TEXT,
  color      VARCHAR(7),
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_monthly_goals_user_month ON monthly_goals (user_id, month);

-- ─── Down migration (referência) ─────────────────────────────────────────────
-- DROP TABLE IF EXISTS monthly_goals;
-- DROP TABLE IF EXISTS tasks;
