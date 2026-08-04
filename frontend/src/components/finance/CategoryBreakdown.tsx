import type { CategorySummary } from '@/types/finance'
import { categoryIcon, categoryLabel, formatCents } from '@/types/finance'
import styles from './CategoryBreakdown.module.css'

interface CategoryBreakdownProps {
  categories: CategorySummary[]
}

// Barra horizontal por categoria — série única em um só matiz (acento),
// com o orçamento como contexto textual neutro, nunca alarme visual.
export function CategoryBreakdown({ categories }: CategoryBreakdownProps) {
  const spent = categories.filter((c) => c.spent_cents > 0 || c.budget_cents)
  if (spent.length === 0) {
    return (
      <p className={styles.empty}>
        Nenhum gasto registrado neste mês.
      </p>
    )
  }

  const max = Math.max(...spent.map((c) => Math.max(c.spent_cents, c.budget_cents ?? 0)), 1)

  return (
    <ul className={styles.list}>
      {spent.map((c) => {
        const spentPct = Math.min((c.spent_cents / max) * 100, 100)
        const budgetPct = c.budget_cents ? Math.min((c.budget_cents / max) * 100, 100) : null
        const usage = c.budget_cents ? Math.round((c.spent_cents / c.budget_cents) * 100) : null
        return (
          <li
            key={c.category}
            className={styles.row}
            title={
              c.budget_cents
                ? `${categoryLabel(c.category)}: ${formatCents(c.spent_cents)} de ${formatCents(c.budget_cents)} (${usage}%)`
                : `${categoryLabel(c.category)}: ${formatCents(c.spent_cents)}`
            }
          >
            <span className={styles.label}>
              <span className={styles.icon}>{categoryIcon(c.category)}</span>
              {categoryLabel(c.category)}
            </span>
            <div className={styles.track}>
              {budgetPct !== null && (
                <div className={styles.budgetMark} style={{ left: `${budgetPct}%` }} />
              )}
              <div className={styles.bar} style={{ width: `${spentPct}%` }} />
            </div>
            <span className={styles.value}>
              {formatCents(c.spent_cents)}
              {c.budget_cents != null && (
                <span className={styles.budgetNote}> de {formatCents(c.budget_cents)}</span>
              )}
            </span>
          </li>
        )
      })}
    </ul>
  )
}
