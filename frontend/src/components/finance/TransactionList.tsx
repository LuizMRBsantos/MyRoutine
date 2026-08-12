import { useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { useDeleteTransaction, useDeleteInstallmentGroup } from '@/hooks/useFinance'
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
  const deleteGroup = useDeleteInstallmentGroup()
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
                    <span className={styles.itemTitle}>
                      {t.description}
                      {t.installment_total && (
                        <span className={styles.installmentTag}>
                          {t.installment_number}/{t.installment_total}
                        </span>
                      )}
                    </span>
                    <span className={styles.itemMeta}>
                      {categoryLabel(t.category)}
                      {t.credit_card_name ? ` · 💳 ${t.credit_card_name}` : t.method ? ` · ${t.method}` : ''}
                      {t.purchased_on && t.purchased_on !== t.occurred_on
                        ? ` · compra ${new Date(t.purchased_on + 'T12:00').toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' })}`
                        : ''}
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

      {/* Apagar uma parcela sozinha deixaria a compra pela metade, então o
          diálogo remove o grupo inteiro quando a transação é parcelada. */}
      <ConfirmDialog
        open={!!deleting}
        title={deleting?.installment_group_id ? 'Remover a compra parcelada?' : 'Remover transação?'}
        message={
          deleting?.installment_group_id
            ? `"${deleting.description}" foi lançada em ${deleting.installment_total}x. Todas as parcelas serão removidas, inclusive as dos próximos meses.`
            : `"${deleting?.description}" (${deleting ? formatCents(deleting.amount_cents) : ''}) será removida.`
        }
        confirmLabel="Remover"
        onConfirm={() => {
          if (deleting?.installment_group_id) {
            deleteGroup.mutate(deleting.installment_group_id)
          } else if (deleting) {
            deleteTransaction.mutate(deleting.id)
          }
          setDeleting(null)
        }}
        onCancel={() => setDeleting(null)}
      />
    </div>
  )
}
