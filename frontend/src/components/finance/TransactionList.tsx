import { useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useDeleteTransaction } from '@/hooks/useFinance'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { categoryIcon, categoryLabel, formatCents } from '@/types/finance'
import type { Transaction } from '@/types/finance'
import styles from './TransactionList.module.css'

interface TransactionListProps {
  transactions: Transaction[]
  isLoading: boolean
  isError: boolean
}

export function TransactionList({ transactions, isLoading, isError }: TransactionListProps) {
  const deleteTransaction = useDeleteTransaction()
  const [deleting, setDeleting] = useState<Transaction | null>(null)

  if (isLoading) return <p className={styles.hint}>Carregando...</p>
  if (isError) return <p className={styles.hint}>Não foi possível carregar as transações.</p>
  if (transactions.length === 0) {
    return <p className={styles.hint}>Nenhuma transação neste período.</p>
  }

  // Agrupa por dia
  const byDay = transactions.reduce<Record<string, Transaction[]>>((acc, t) => {
    ;(acc[t.occurred_on] ??= []).push(t)
    return acc
  }, {})
  const days = Object.keys(byDay).sort((a, b) => b.localeCompare(a))

  return (
    <div className={styles.wrapper}>
      {days.map((day) => (
        <div key={day} className={styles.dayGroup}>
          <p className={styles.dayLabel}>
            {new Date(day + 'T12:00').toLocaleDateString('pt-BR', {
              weekday: 'long', day: 'numeric', month: 'long',
            })}
          </p>
          <ul className={styles.list}>
            <AnimatePresence>
              {byDay[day].map((t) => (
                <motion.li
                  key={t.id}
                  className={styles.item}
                  initial={{ opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, scale: 0.97 }}
                >
                  <span className={styles.itemIcon}>{categoryIcon(t.category)}</span>
                  <div className={styles.itemBody}>
                    <span className={styles.itemTitle}>{t.description}</span>
                    <span className={styles.itemMeta}>
                      {categoryLabel(t.category)}
                      {t.method ? ` · ${t.method}` : ''}
                      {t.source_type === 'track_day' ? ' · via diário' : ''}
                    </span>
                  </div>
                  <span className={`${styles.amount} ${t.kind === 'income' ? styles.income : ''}`}>
                    {t.kind === 'income' ? '+' : '−'}{formatCents(t.amount_cents)}
                  </span>
                  <button
                    className={styles.deleteBtn}
                    title="Remover"
                    onClick={() => setDeleting(t)}
                  >
                    ×
                  </button>
                </motion.li>
              ))}
            </AnimatePresence>
          </ul>
        </div>
      ))}

      <ConfirmDialog
        open={!!deleting}
        title="Remover transação?"
        message={`"${deleting?.description}" (${deleting ? formatCents(deleting.amount_cents) : ''}) será removida.`}
        confirmLabel="Remover"
        onConfirm={() => {
          if (deleting) deleteTransaction.mutate(deleting.id)
          setDeleting(null)
        }}
        onCancel={() => setDeleting(null)}
      />
    </div>
  )
}
