// ─── Check Types ──────────────────────────────────────────
export type HabitCheckType = 'simple' | 'timed' | 'deadline' | 'metric'

// ─── Metric Config ────────────────────────────────────────
export interface MetricField {
  key: string
  label: string
  unit: string
  // Se is_target=true, o valor é uma META definida na criação do hábito
  // e vem pré-preenchida no check-in (mas editável).
  is_target?: boolean
  target_value?: number
}

// ─── Habit ────────────────────────────────────────────────
export type HabitCategory = 'general' | 'health' | 'study'

export interface Habit {
  id: string
  name: string
  description: string
  icon: string
  color: string
  frequency: 'daily' | 'weekly' | 'custom'
  target_days: number[]
  time_of_day: 'morning' | 'afternoon' | 'evening' | 'anytime'
  category: HabitCategory
  is_active: boolean
  created_at: string
  current_streak: number
  completed_today: boolean
  // Hoje (no fuso da pessoa) é um dos target_days.
  scheduled_today: boolean
  // Advanced check type fields
  check_type: HabitCheckType
  timer_minutes?: number
  deadline_time?: string       // "HH:MM" format
  metric_config?: MetricField[]
}

// ─── Habit Log ────────────────────────────────────────────
export interface HabitLog {
  id: string
  habit_id: string
  logged_date: string
  notes: string
  created_at: string
  // Advanced check-in fields
  timer_seconds?: number
  started_at?: string
  completed_at?: string
  metrics?: Record<string, number>
  is_manual: boolean
  source_type: 'manual' | 'task' | 'track_day' | 'study_session' | 'import'
}

// ─── Heatmap & Stats ──────────────────────────────────────
export interface HeatmapEntry {
  date: string
  habits_completed: number
}

export interface HabitStat {
  habit_id: string
  habit_name: string
  icon: string
  color: string
  current_streak: number
  completion_rate: number
}

export interface HabitStats {
  total_habits: number
  completed_today: number
  total_check_ins: number
  best_streak: number
  current_streak: number
  completion_rate_7d: number
  habit_stats: HabitStat[]
}

// ─── Create Input ─────────────────────────────────────────
export interface CreateHabitInput {
  name: string
  description?: string
  icon: string
  color: string
  frequency: string
  target_days: number[]
  time_of_day: 'morning' | 'afternoon' | 'evening' | 'anytime'
  category?: HabitCategory
  // Advanced check type fields
  check_type?: HabitCheckType
  timer_minutes?: number
  deadline_time?: string
  metric_config?: MetricField[]
}

// ─── Check-in Input ───────────────────────────────────────
export interface CheckInInput {
  notes?: string
  timer_seconds?: number
  started_at?: string
  completed_at?: string
  metrics?: Record<string, number>
  is_manual?: boolean
}
