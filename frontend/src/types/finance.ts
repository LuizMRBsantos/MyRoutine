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

// "12,34" | "12.34" | "1.234,56" | "12" → centavos inteiros.
//
// O ponto é ambíguo em pt-BR: separa milhar em "1.234,56" mas é decimal em
// "25.90" (como muita gente digita). Desambiguamos assim: havendo vírgula,
// ela é o decimal e os pontos são milhar; sem vírgula, um único ponto
// seguido de 1 ou 2 dígitos é decimal — o resto é milhar.
export function parseAmountToCents(value: string): number | null {
  const trimmed = value.trim()
  if (!trimmed) return null

  let normalized: string
  if (trimmed.includes(',')) {
    normalized = trimmed.replace(/\./g, '').replace(',', '.')
  } else if (/^\d+\.\d{1,2}$/.test(trimmed)) {
    normalized = trimmed
  } else {
    normalized = trimmed.replace(/\./g, '')
  }

  const parsed = Number(normalized)
  if (!Number.isFinite(parsed) || parsed <= 0) return null
  return Math.round(parsed * 100)
}
