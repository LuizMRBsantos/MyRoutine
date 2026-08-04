import { useState } from 'react'
import { useBudgets, useUpsertBudget, useDeleteBudget } from '@/hooks/useFinance'
import { FINANCE_CATEGORIES, categoryIcon, categoryLabel, formatCents, parseAmountToCents } from '@/types/finance'
import { toast } from '@/lib/toast'
import styles from './BudgetsPanel.module.css'

interface BudgetsPanelProps {
  month: string
}

export function BudgetsPanel({ month }: BudgetsPanelProps) {
  const { data: budgets = [] } = useBudgets(month)
  const upsertBudget = useUpsertBudget()
  const deleteBudget = useDeleteBudget()

  const [category, setCategory] = useState('alimentacao')
  const [amount, setAmount] = useState('')

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault()
    const cents = parseAmountToCents(amount)
    if (!cents) {
      toast.error('Informe um valor válido para o orçamento')
      return
    }
    upsertBudget.mutate(
      { category, month, amount_cents: cents },
      { onSuccess: () => { setAmount(''); toast.success('Orçamento salvo') } }
    )
  }

  return (
    <div className={styles.panel}>
      <form onSubmit={handleSave} className={styles.addRow}>
        <select
          className={`input ${styles.select}`}
          value={category}
          onChange={(e) => setCategory(e.target.value)}
        >
          {Object.entries(FINANCE_CATEGORIES)
            .filter(([key]) => key !== 'salario')
            .map(([key, meta]) => (
              <option key={key} value={key}>{meta.icon} {meta.label}</option>
            ))}
        </select>
        <input
          className={`input ${styles.amount}`}
          placeholder="R$ 0,00"
          inputMode="decimal"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
        <button type="submit" className="btn btn-ghost" disabled={upsertBudget.isPending}>
          Definir
        </button>
      </form>

      {budgets.length > 0 && (
        <ul className={styles.list}>
          {budgets.map((b) => (
            <li key={b.id} className={styles.item}>
              <span className={styles.itemLabel}>
                {categoryIcon(b.category)} {categoryLabel(b.category)}
              </span>
              <span className={styles.itemValue}>{formatCents(b.amount_cents)}</span>
              <button
                className={styles.removeBtn}
                title="Remover orçamento"
                onClick={() => deleteBudget.mutate(b.id)}
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
