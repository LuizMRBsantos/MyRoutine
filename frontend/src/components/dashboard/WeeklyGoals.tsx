import { useState } from 'react'
import {
  useWeeklyGoals, useCreateWeeklyGoal, useToggleWeeklyGoal, useDeleteWeeklyGoal,
} from '@/hooks/useWeeklyGoals'
import styles from './WeeklyGoals.module.css'

// "Semana de 12 a 18 de out." a partir da segunda-feira ("YYYY-MM-DD").
function weekLabel(monday: string): string {
  const start = new Date(`${monday}T12:00:00`)
  const end = new Date(start)
  end.setDate(start.getDate() + 6)
  const month = (d: Date) => d.toLocaleDateString('pt-BR', { month: 'short' })
  return start.getMonth() === end.getMonth()
    ? `Semana de ${start.getDate()} a ${end.getDate()} de ${month(end)}`
    : `Semana de ${start.getDate()} de ${month(start)} a ${end.getDate()} de ${month(end)}`
}

// Metas da semana: feita ou não. Sem placar nem porcentagem
// (product-constitution) — só a lista, para lembrar o que importa.
export function WeeklyGoals() {
  const { data: goals = [], isLoading, isError } = useWeeklyGoals()
  const create = useCreateWeeklyGoal()
  const toggle = useToggleWeeklyGoal()
  const remove = useDeleteWeeklyGoal()
  const [title, setTitle] = useState('')

  const handleAdd = (e: React.FormEvent) => {
    e.preventDefault()
    const t = title.trim()
    if (!t) return
    create.mutate(t, { onSuccess: () => setTitle('') })
  }

  return (
    <section className={`glass-card ${styles.card}`} aria-labelledby="weekly-goals-title">
      <div className={styles.header}>
        <h2 id="weekly-goals-title" className={styles.title}>Metas da semana</h2>
        {goals[0] && <span className={styles.week}>{weekLabel(goals[0].week)}</span>}
      </div>

      {isLoading && <p className={styles.hint}>Carregando…</p>}
      {isError && <p className={styles.hint} role="status">Não deu para carregar suas metas agora.</p>}

      {goals.length > 0 && (
        <ul className={styles.list}>
          {goals.map(goal => (
            <li key={goal.id} className={styles.item}>
              <label className={`${styles.goal} ${goal.done ? styles.done : ''}`}>
                <input
                  type="checkbox"
                  checked={goal.done}
                  onChange={e => toggle.mutate({ id: goal.id, done: e.target.checked })}
                />
                <span>{goal.title}</span>
              </label>
              <button
                type="button"
                className={styles.remove}
                aria-label={`Apagar meta: ${goal.title}`}
                onClick={() => remove.mutate(goal.id)}
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}

      {!isLoading && !isError && goals.length === 0 && (
        <p className={styles.hint}>O que você quer fazer esta semana?</p>
      )}

      <form onSubmit={handleAdd} className={styles.addRow}>
        <input
          className="input"
          placeholder="Nova meta da semana"
          aria-label="Nova meta da semana"
          maxLength={255}
          value={title}
          onChange={e => setTitle(e.target.value)}
        />
        <button type="submit" className="btn btn-primary" disabled={!title.trim() || create.isPending}>
          Adicionar
        </button>
      </form>
    </section>
  )
}
