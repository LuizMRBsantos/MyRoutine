import { useState } from 'react'
import { useTransactions, useFinanceSummary } from '@/hooks/useFinance'
import { QuickAddTransaction } from '@/components/finance/QuickAddTransaction'
import { TransactionList } from '@/components/finance/TransactionList'
import { CategoryBreakdown } from '@/components/finance/CategoryBreakdown'
import { BudgetsPanel } from '@/components/finance/BudgetsPanel'
import { CardsPanel } from '@/components/finance/CardsPanel'
import { formatCents } from '@/types/finance'
import styles from './FinancePage.module.css'

function pad(n: number) {
  return n < 10 ? `0${n}` : `${n}`
}

function monthBounds(d: Date) {
  const y = d.getFullYear()
  const m = d.getMonth()
  const lastDay = new Date(y, m + 1, 0).getDate()
  return {
    month: `${y}-${pad(m + 1)}-01`,
    from: `${y}-${pad(m + 1)}-01`,
    to: `${y}-${pad(m + 1)}-${pad(lastDay)}`,
  }
}

export function FinancePage() {
  const [viewDate, setViewDate] = useState(new Date())
  const { month, from, to } = monthBounds(viewDate)

  const { data: summary } = useFinanceSummary(month)
  const { data: transactions = [], isLoading, isError } = useTransactions(from, to)

  const balance = (summary?.income_cents ?? 0) - (summary?.expense_cents ?? 0)

  return (
    <div className={styles.page}>
      {/* ── Header ── */}
      <div className={styles.header}>
        <div>
          <h1 className={styles.pageTitle}>Finanças</h1>
          <p className={styles.pageSubtitle}>Transações e orçamentos do mês</p>
        </div>
        <div className={styles.monthNav}>
          <button
            className={styles.navBtn}
            onClick={() => setViewDate((d) => new Date(d.getFullYear(), d.getMonth() - 1, 1))}
          >
            ‹
          </button>
          <span className={styles.monthLabel}>
            {viewDate.toLocaleDateString('pt-BR', { month: 'long', year: 'numeric' })}
          </span>
          <button
            className={styles.navBtn}
            onClick={() => setViewDate((d) => new Date(d.getFullYear(), d.getMonth() + 1, 1))}
          >
            ›
          </button>
        </div>
      </div>

      {/* ── Summary tiles ── */}
      {summary && (
        <div className={styles.statsGrid}>
          <div className={`glass-card ${styles.statCard}`}>
            <p className={styles.statLabel}>Receitas</p>
            <p className={styles.statValue} style={{ color: 'var(--color-success)' }}>
              {formatCents(summary.income_cents)}
            </p>
          </div>
          <div className={`glass-card ${styles.statCard}`}>
            <p className={styles.statLabel}>Despesas</p>
            <p className={styles.statValue}>{formatCents(summary.expense_cents)}</p>
          </div>
          <div className={`glass-card ${styles.statCard}`}>
            <p className={styles.statLabel}>Saldo do mês</p>
            <p className={styles.statValue}>{formatCents(balance)}</p>
          </div>
        </div>
      )}

      {/* ── Quick add ── */}
      <QuickAddTransaction />

      <div className={styles.columns}>
        {/* ── Transactions ── */}
        <section className={`glass-card ${styles.section}`}>
          <h2 className={styles.sectionTitle}>Transações</h2>
          <TransactionList transactions={transactions} isLoading={isLoading} isError={isError} />
        </section>

        <div className={styles.sideColumn}>
          {/* ── Category breakdown ── */}
          <section className={`glass-card ${styles.section}`}>
            <h2 className={styles.sectionTitle}>Gastos por categoria</h2>
            <CategoryBreakdown categories={summary?.by_category ?? []} />
          </section>

          {/* ── Budgets ── */}
          <section className={`glass-card ${styles.section}`}>
            <h2 className={styles.sectionTitle}>Orçamentos do mês</h2>
            <BudgetsPanel month={month} />
          </section>

          {/* ── Credit cards ── */}
          <section className={`glass-card ${styles.section}`}>
            <h2 className={styles.sectionTitle}>Cartões</h2>
            <CardsPanel />
          </section>
        </div>
      </div>
    </div>
  )
}
