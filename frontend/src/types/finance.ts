export type TransactionKind = 'expense' | 'income'

export interface Transaction {
  id: string
  amount_cents: number
  kind: TransactionKind
  category: string
  description: string
  method: string | null
  occurred_on: string
  source_type: 'manual' | 'track_day' | 'import'
  source_id?: string
  created_at: string
}

export interface CreateTransactionInput {
  amount_cents: number
  kind?: TransactionKind
  category: string
  description: string
  method?: string
  occurred_on: string
}

export interface Budget {
  id: string
  category: string
  month: string
  amount_cents: number
}

export interface CategorySummary {
  category: string
  spent_cents: number
  budget_cents?: number
}

export interface FinanceSummary {
  month: string
  income_cents: number
  expense_cents: number
  by_category: CategorySummary[]
}

// ─── Category metadata ────────────────────────────────────
export const FINANCE_CATEGORIES: Record<string, { label: string; icon: string }> = {
  alimentacao: { label: 'Alimentação', icon: '🍽' },
  transporte: { label: 'Transporte', icon: '🚗' },
  moradia: { label: 'Moradia', icon: '🏠' },
  saude: { label: 'Saúde', icon: '💊' },
  lazer: { label: 'Lazer', icon: '🎬' },
  educacao: { label: 'Educação', icon: '📚' },
  esporte: { label: 'Esporte', icon: '🏃' },
  salario: { label: 'Salário', icon: '💼' },
  other: { label: 'Outros', icon: '📦' },
}

export function categoryLabel(key: string): string {
  return FINANCE_CATEGORIES[key]?.label ?? key
}

export function categoryIcon(key: string): string {
  return FINANCE_CATEGORIES[key]?.icon ?? '📦'
}

// ─── Money helpers ────────────────────────────────────────
const brl = new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' })

export function formatCents(cents: number): string {
  return brl.format(cents / 100)
}

// "12,34" | "12.34" | "12" → 1234
export function parseAmountToCents(value: string): number | null {
  const normalized = value.trim().replace(/\./g, '').replace(',', '.')
  const parsed = Number(normalized)
  if (!Number.isFinite(parsed) || parsed <= 0) return null
  return Math.round(parsed * 100)
}
