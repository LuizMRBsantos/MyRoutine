// ─── Category ─────────────────────────────────────────────
export type TaskCategory =
  | 'wake_up'
  | 'exercise'
  | 'study'
  | 'work'
  | 'exam'
  | 'appointment'
  | 'other'

// ─── Status ───────────────────────────────────────────────
// Fluxo linear: planned → in_progress → done → reviewed
export type TaskStatus = 'planned' | 'in_progress' | 'done' | 'reviewed'

export type TaskPriority = 'high' | 'medium' | 'low'

// ─── Task Details — payload por categoria ─────────────────
export interface ExerciseDetails {
  workout_type?: string  // ex: "corrida", "musculação"
  plan?: string          // ex: "5km no parque"
  km?: number
  time_min?: number
  calories?: number
}

export interface StudyDetails {
  subject?: string       // ex: "OAC", "BD", "PLP"
  topic?: string         // ex: "Pipeline de instrução"
  exercises_count?: number
}

export interface WorkDetails {
  project?: string       // ex: "MyRoutine", "Nexx"
  deliverables?: string[]
}

export interface ExamDetails {
  subject?: string
  location?: string
  score?: number | null
}

export type TaskDetails = ExerciseDetails | StudyDetails | WorkDetails | ExamDetails | Record<string, unknown>

// ─── Task ─────────────────────────────────────────────────
export interface Task {
  id: string
  user_id: string
  title: string
  date: string             // "YYYY-MM-DD"
  start_time?: string      // "HH:MM" ou undefined
  duration_minutes?: number
  category: TaskCategory
  status: TaskStatus
  priority: TaskPriority
  notes?: string
  task_details?: TaskDetails
  linked_habit_id?: string
  color?: string
  notify: boolean          // "Me avisar antes" (lembrete pelo horário)
  created_at: string
  updated_at: string
}

// ─── Create Input ─────────────────────────────────────────
export interface CreateTaskInput {
  title: string
  date: string
  start_time?: string
  duration_minutes?: number
  category?: TaskCategory
  priority?: TaskPriority
  notes?: string
  task_details?: TaskDetails
  linked_habit_id?: string
  color?: string
  notify?: boolean         // ausente = padrão da categoria
}

// Compromissos, provas e trabalho (reuniões) avisam antes por padrão; aulas,
// exercício, acordar e "outro" só preenchem o calendário. Igual ao backend
// (service.NotifyByDefault).
export function notifyByDefault(category: TaskCategory): boolean {
  return category === 'appointment' || category === 'exam' || category === 'work'
}

// ─── Update Input ─────────────────────────────────────────
export interface UpdateTaskInput {
  title?: string
  start_time?: string
  duration_minutes?: number
  status?: TaskStatus
  priority?: TaskPriority
  notes?: string
  task_details?: TaskDetails
  linked_habit_id?: string
  color?: string
  notify?: boolean
}

// ─── Monthly Goal ─────────────────────────────────────────
export type GoalStatus = 'active' | 'done' | 'abandoned'

export interface MonthlyGoal {
  id: string
  user_id: string
  title: string
  month: string     // "YYYY-MM-DD" (sempre dia 1)
  status: GoalStatus
  notes?: string
  color?: string
  created_at: string
}

export interface CreateGoalInput {
  title: string
  month: string
  notes?: string
  color?: string
}

// ─── UI Helpers ───────────────────────────────────────────

export const CATEGORY_META: Record<TaskCategory, { label: string; icon: string; color: string }> = {
  wake_up:     { label: 'Acordar',      icon: '🌅', color: '#FF9F0A' },
  exercise:    { label: 'Exercício',    icon: '🏃', color: '#34C759' },
  study:       { label: 'Estudo',       icon: '📚', color: '#0071E3' },
  work:        { label: 'Trabalho',     icon: '💻', color: '#5856D6' },
  exam:        { label: 'Prova',        icon: '📝', color: '#FF3B30' },
  appointment: { label: 'Compromisso', icon: '📅', color: '#5AC8FA' },
  other:       { label: 'Outro',        icon: '✦',  color: '#8E8E93' },
}

export const STATUS_META: Record<TaskStatus, { label: string; next: TaskStatus | null }> = {
  planned:     { label: 'Planejado',   next: 'in_progress' },
  in_progress: { label: 'Fazendo',     next: 'done' },
  done:        { label: 'Terminado',   next: 'reviewed' },
  reviewed:    { label: 'Revisado',    next: null },
}
