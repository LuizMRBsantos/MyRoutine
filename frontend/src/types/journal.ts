export interface JournalItem {
  source_id: string
  kind: 'transaction' | 'workout'
  label: string
  amount_cents?: number
  habit_id?: string
  metrics?: Record<string, number>
}

export interface Journal {
  date: string
  content: string
  updated_at: string | null
  // linha normalizada (ver lineKey) → etiqueta do que foi registrado a partir dela
  line_ids: Record<string, string>
  items: JournalItem[]
}

export type RegisterJournalItem =
  | {
      line: string
      kind: 'transaction'
      expense: { amount_cents: number; category: string; description: string }
    }
  | {
      line: string
      kind: 'workout'
      workout: { habit_id: string; metrics: Record<string, number>; time_minutes?: number }
    }
