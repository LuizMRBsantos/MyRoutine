export interface Activity {
  log_id: string
  habit_id: string
  habit_name: string
  icon: string
  color: string
  date: string
  timer_seconds?: number
  metrics?: Record<string, number>
  notes: string
  source_type: string
}

export interface WeeklyVolume {
  week_start: string
  sessions: number
  total_km: number
  total_minutes: number
  avg_rpe: number
}

export interface HealthSummary {
  weeks: WeeklyVolume[]
}

export interface BodyMetric {
  id: string
  measured_on: string
  weight_kg: number | null
  notes: string | null
}
