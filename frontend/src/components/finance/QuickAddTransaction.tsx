import { useState } from 'react'
import { useCreateTransaction, useCards } from '@/hooks/useFinance'
import { FINANCE_CATEGORIES, formatCents, parseAmountToCents } from '@/types/finance'
import type { TransactionKind } from '@/types/finance'
import { toast } from '@/lib/toast'
import styles from './QuickAddTransaction.module.css'

function todayStr() {
  const d = new Date()
  const pad = (n: number) => (n < 10 ? `0${n}` : `${n}`)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/**
 * Espelha a regra do backend (chargeDate) só para pré-visualizar quando a
 * primeira parcela cai. O valor gravado é sempre o que o servidor calcula.
 */
function previewFirstCharge(purchase: string, closingDay: number, dueDay: number): string | null {
  const [y, m, d] = purchase.split('-').map(Number)
  if (!y || !m || !d) return null

  const lastDayOf = (year: number, month: number) => new Date(year, month, 0).getDate()
  const clamp = (year: number, month: number, day: number) => {
    const capped = Math.min(day, lastDayOf(year, month))
    return new Date(year, month - 1, capped)
  }

  let closingMonth = m
  let closingYear = y
  if (d > Math.min(closingDay, lastDayOf(y, m))) {
    closingMonth += 1
    if (closingMonth > 12) { closingMonth = 1; closingYear += 1 }
  }

  let dueMonth = closingMonth
  let dueYear = closingYear
  if (dueDay <= closingDay) {
    dueMonth += 1
    if (dueMonth > 12) { dueMonth = 1; dueYear += 1 }
  }

  return clamp(dueYear, dueMonth, dueDay).toLocaleDateString('pt-BR', {
    day: '2-digit', month: 'short', year: 'numeric',
  })
}

export function QuickAddTransaction() {
  const createTransaction = useCreateTransaction()
  const { data: cards = [] } = useCards()

  const [kind, setKind] = useState<TransactionKind>('expense')
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')
  const [category, setCategory] = useState('alimentacao')
  const [date, setDate] = useState(todayStr())
  const [cardId, setCardId] = useState('')
  const [installments, setInstallments] = useState(1)

  const selectedCard = cards.find((c) => c.id === cardId)
  const cents = parseAmountToCents(amount)
  const isCardPurchase = kind === 'expense' && !!selectedCard

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
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
        ...(isCardPurchase ? { credit_card_id: cardId, installments } : {}),
      },
      {
        onSuccess: () => {
          setAmount('')
          setDescription('')
          setInstallments(1)
          toast.success(
            isCardPurchase && installments > 1
              ? `Compra lançada em ${installments}x`
              : kind === 'expense' ? 'Despesa registrada' : 'Receita registrada'
          )
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
          onClick={() => { setKind('income'); setCardId('') }}
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
            aria-label="Categoria"
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

      {/* Cartão e parcelamento — só faz sentido em despesa */}
      {kind === 'expense' && cards.length > 0 && (
        <div className={styles.cardRow}>
          <select
            aria-label="Forma de pagamento"
            className={`input ${styles.cardSelect}`}
            value={cardId}
            onChange={(e) => { setCardId(e.target.value); if (!e.target.value) setInstallments(1) }}
          >
            <option value="">À vista (pix, débito, dinheiro)</option>
            {cards.map((c) => (
              <option key={c.id} value={c.id}>💳 {c.name}</option>
            ))}
          </select>

          {isCardPurchase && (
            <select
              aria-label="Parcelas"
              className={`input ${styles.installmentSelect}`}
              value={installments}
              onChange={(e) => setInstallments(Number(e.target.value))}
            >
              {Array.from({ length: 24 }, (_, i) => i + 1).map((n) => (
                <option key={n} value={n}>{n === 1 ? 'À vista no cartão' : `${n}x`}</option>
              ))}
            </select>
          )}
        </div>
      )}

      {/* Projeção: onde essa compra vai cair de verdade */}
      {isCardPurchase && cents && selectedCard && (
        <p className={styles.preview}>
          {installments > 1
            ? `${installments}x de ${formatCents(Math.floor(cents / installments))} `
            : `${formatCents(cents)} `}
          — primeira cobrança em{' '}
          <strong>{previewFirstCharge(date, selectedCard.closing_day, selectedCard.due_day)}</strong>
          {installments > 1 && `, última ${installments - 1} ${installments - 1 === 1 ? 'mês' : 'meses'} depois`}
        </p>
      )}
    </form>
  )
}
