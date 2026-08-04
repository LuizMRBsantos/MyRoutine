import { useState } from 'react'
import { useCreateTransaction } from '@/hooks/useFinance'
import { FINANCE_CATEGORIES, parseAmountToCents } from '@/types/finance'
import type { TransactionKind } from '@/types/finance'
import { toast } from '@/lib/toast'
import styles from './QuickAddTransaction.module.css'

function todayStr() {
  const d = new Date()
  const pad = (n: number) => (n < 10 ? `0${n}` : `${n}`)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function QuickAddTransaction() {
  const createTransaction = useCreateTransaction()
  const [kind, setKind] = useState<TransactionKind>('expense')
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')
  const [category, setCategory] = useState('alimentacao')
  const [date, setDate] = useState(todayStr())

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const cents = parseAmountToCents(amount)
    if (!cents) {
      toast.error('Informe um valor válido (ex: 25,90)')
      return
    }
    if (!description.trim()) {
      toast.error('Descreva a transação')
      return
    }
    createTransaction.mutate(
      {
        amount_cents: cents,
        kind,
        category: kind === 'income' ? 'salario' : category,
        description: description.trim(),
        occurred_on: date,
      },
      {
        onSuccess: () => {
          setAmount('')
          setDescription('')
          toast.success(kind === 'expense' ? 'Despesa registrada' : 'Receita registrada')
        },
      }
    )
  }

  return (
    <form onSubmit={handleSubmit} className={`glass-card ${styles.card}`}>
      <div className={styles.kindToggle}>
        <button
          type="button"
          className={`${styles.kindBtn} ${kind === 'expense' ? styles.kindActive : ''}`}
          onClick={() => setKind('expense')}
        >
          Despesa
        </button>
        <button
          type="button"
          className={`${styles.kindBtn} ${kind === 'income' ? styles.kindActive : ''}`}
          onClick={() => setKind('income')}
        >
          Receita
        </button>
      </div>

      <div className={styles.fields}>
        <input
          className={`input ${styles.amount}`}
          placeholder="R$ 0,00"
          inputMode="decimal"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
        <input
          className={`input ${styles.description}`}
          placeholder={kind === 'expense' ? 'Ex: Almoço' : 'Ex: Bolsa de estudos'}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        {kind === 'expense' && (
          <select
            className={`input ${styles.category}`}
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          >
            {Object.entries(FINANCE_CATEGORIES)
              .filter(([key]) => key !== 'salario')
              .map(([key, meta]) => (
                <option key={key} value={key}>{meta.icon} {meta.label}</option>
              ))}
          </select>
        )}
        <input
          type="date"
          className={`input ${styles.date}`}
          value={date}
          onChange={(e) => setDate(e.target.value)}
        />
        <button type="submit" className="btn btn-primary" disabled={createTransaction.isPending}>
          Adicionar
        </button>
      </div>
    </form>
  )
}
