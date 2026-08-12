import { useState } from 'react'
import { useCards, useCreateCard, useDeleteCard } from '@/hooks/useFinance'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { toast } from '@/lib/toast'
import type { CreditCard } from '@/types/finance'
import styles from './CardsPanel.module.css'

const DAYS = Array.from({ length: 31 }, (_, i) => i + 1)

export function CardsPanel() {
  const { data: cards = [] } = useCards()
  const createCard = useCreateCard()
  const deleteCard = useDeleteCard()

  const [adding, setAdding] = useState(false)
  const [name, setName] = useState('')
  const [closingDay, setClosingDay] = useState(25)
  const [dueDay, setDueDay] = useState(5)
  const [deleting, setDeleting] = useState<CreditCard | null>(null)

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) {
      toast.error('Dê um nome ao cartão')
      return
    }
    createCard.mutate(
      { name: name.trim(), closing_day: closingDay, due_day: dueDay },
      {
        onSuccess: () => {
          setName('')
          setAdding(false)
          toast.success('Cartão cadastrado')
        },
      }
    )
  }

  return (
    <div className={styles.panel}>
      {cards.length === 0 && !adding && (
        <p className={styles.hint}>
          Cadastre um cartão para lançar compras parceladas — o sistema calcula
          em qual fatura cada parcela cai.
        </p>
      )}

      {cards.length > 0 && (
        <ul className={styles.list}>
          {cards.map((c) => (
            <li key={c.id} className={styles.item}>
              <span className={styles.itemIcon}>💳</span>
              <div className={styles.itemBody}>
                <span className={styles.itemName}>{c.name}</span>
                <span className={styles.itemMeta}>
                  fecha dia {c.closing_day} · vence dia {c.due_day}
                </span>
              </div>
              <button
                className={styles.removeBtn}
                title="Remover cartão"
                onClick={() => setDeleting(c)}
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}

      {adding ? (
        <form onSubmit={handleCreate} className={styles.form}>
          <input
            className="input"
            placeholder="Nome do cartão (ex: Nubank)"
            value={name}
            onChange={(e) => setName(e.target.value)}
            autoFocus
          />
          <div className={styles.dayRow}>
            <label className={styles.dayField}>
              <span className={styles.dayLabel}>Fecha dia</span>
              <select
                aria-label="Dia do fechamento"
                className="input"
                value={closingDay}
                onChange={(e) => setClosingDay(Number(e.target.value))}
              >
                {DAYS.map((d) => <option key={d} value={d}>{d}</option>)}
              </select>
            </label>
            <label className={styles.dayField}>
              <span className={styles.dayLabel}>Vence dia</span>
              <select
                aria-label="Dia do vencimento"
                className="input"
                value={dueDay}
                onChange={(e) => setDueDay(Number(e.target.value))}
              >
                {DAYS.map((d) => <option key={d} value={d}>{d}</option>)}
              </select>
            </label>
          </div>
          <div className={styles.formActions}>
            <button type="button" className="btn btn-ghost" onClick={() => setAdding(false)}>
              Cancelar
            </button>
            <button type="submit" className="btn btn-primary" disabled={createCard.isPending}>
              Salvar
            </button>
          </div>
        </form>
      ) : (
        <button className={styles.addBtn} onClick={() => setAdding(true)}>
          + Cadastrar cartão
        </button>
      )}

      <ConfirmDialog
        open={!!deleting}
        title="Remover cartão?"
        message={`"${deleting?.name}" sai da lista, mas as compras já lançadas continuam no histórico.`}
        confirmLabel="Remover"
        onConfirm={() => {
          if (deleting) deleteCard.mutate(deleting.id)
          setDeleting(null)
        }}
        onCancel={() => setDeleting(null)}
      />
    </div>
  )
}
