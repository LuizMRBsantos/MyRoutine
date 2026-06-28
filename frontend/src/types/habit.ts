export interface Habit {
  id: string
  name: string
  description: string
  icon: string
  color: string
  frequency: 'daily' | 'weekly' | 'custom'
  target_days: number[]
  is_active: boolean
  created_at: string
  current_streak: number
  completed_today: boolean
}

export interface HabitLog {
  id: string
  habit_id: string
  logged_date: string
  notes: string
  created_at: string
}

export interface HeatmapEntry {
  date: string
  habits_completed: number
}

export interface HabitStats {
  total_habits: number
  completed_today: number
  total_check_ins: number
  best_streak: number
  current_streak: number
  completion_rate_7d: number
}

export interface CreateHabitInput {
  name: string
  description?: string
  icon: string
  color: string
  frequency: string
  target_days: number[]
}
