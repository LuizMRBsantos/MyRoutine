export interface StudySession {
  id: string
  subject: string
  topic: string | null
  studied_on: string
  duration_minutes: number
  notes: string | null
  task_id?: string
  created_at: string
}

export interface CreateStudySessionInput {
  subject: string
  topic?: string
  studied_on?: string
  duration_minutes: number
  notes?: string
  habit_id?: string
}

export interface SubjectSummary {
  subject: string
  total_minutes: number
  sessions: number
}

export interface StudySummary {
  total_minutes_30d: number
  sessions_30d: number
  by_subject: SubjectSummary[]
}
