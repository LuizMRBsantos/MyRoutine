import { useState } from 'react'
import { useMissedDays, useSaveReview, type MissedDay, type ReviewDecision } from '@/hooks/useWeeklyReview'
import { toast } from '@/lib/toast'
import styles from './ReviewPage.module.css'

// Revisão (habit-review-flow, regra atualizada em 19/10/2026): só aparecem os
// dias de ontem e anteontem que ficaram sem registro e sem decisão. A pessoa
// escolhe migrar ou descartar e salva; os dias decididos saem da lista. Dias
// mais antigos sem decisão são descartados pelo app, sem vermelho nem contador.

type Status = ReviewDecision['status']

function dayTitle(iso: string): string {
  return new Date(iso + 'T12:00:00').toLocaleDateString('pt-BR', {
    weekday: 'long', day: 'numeric', month: 'long',
  })
}

function groupByDay(days: MissedDay[]): [string, MissedDay[]][] {
  const groups = new Map<string, MissedDay[]>()
  for (const d of days) groups.set(d.date, [...(groups.get(d.date) ?? []), d])
  return [...groups.entries()]
}

const keyOf = (d: { habit_id: string; date: string }) => `${d.habit_id}|${d.date}`

export function ReviewPage() {
  const { data: days = [], isLoading, isError } = useMissedDays()
  const save = useSaveReview()
  const [choices, setChoices] = useState<Record<string, Status>>({})

  const choose = (day: MissedDay, status: Status) =>
    setChoices(c => {
      const k = keyOf(day)
      if (c[k] === status) {
        const { [k]: _, ...rest } = c
        return rest
      }
      return { ...c, [k]: status }
    })

  const decided = days.filter(d => choices[keyOf(d)])
  const handleSave = () => {
    const decisions = decided.map(d => ({ habitId: d.habit_id, reviewDate: d.date, status: choices[keyOf(d)] }))
    save.mutate(decisions, {
      onSuccess: () => {
        setChoices({})
        toast.success(decisions.length === 1 ? 'Revisão salva' : `${decisions.length} decisões salvas`)
      },
      onError: () => toast.error('Não foi possível salvar a revisão. Tente de novo.'),
    })
  }

  return (
    <div className={styles.page}>
      <div>
        <h1 className={styles.pageTitle}>Revisão</h1>
        <p className={styles.pageSubtitle}>
          Ontem e anteontem, os hábitos que ficaram sem registro. Escolha o que fazer com cada um
          e salve. O que ficar mais de dois dias sem decisão é descartado sozinho — e tudo bem.
        </p>
      </div>

      <div className={`glass-card ${styles.legend}`}>
        <span><strong>→ Migrar</strong>: não fiz, mas ainda quero — levo a intenção para a próxima semana.</span>
        <span><strong>· Descartar</strong>: esse dia não vai ser recuperado, e tudo bem.</span>
      </div>

      {isLoading && <p className={styles.empty}>Carregando…</p>}
      {isError && <p className={styles.empty}>Não deu para carregar a revisão agora.</p>}
      {!isLoading && !isError && days.length === 0 && (
        <p className={styles.empty}>Nada para revisar: nenhum dia sem registro em aberto.</p>
      )}

      {groupByDay(days).map(([date, habits]) => (
        <section key={date} className={`glass-card ${styles.day}`} aria-label={dayTitle(date)}>
          <h2 className={styles.dayTitle}>{dayTitle(date)}</h2>
          {habits.map(day => {
            const chosen = choices[keyOf(day)]
            return (
              <div key={day.habit_id} className={styles.row}>
                <div className={styles.habit}>
                  <span className={styles.habitIcon} aria-hidden="true">{day.habit_icon}</span>
                  <span className={styles.habitName}>{day.habit_name}</span>
                </div>
                <div className={styles.choice} role="group" aria-label={`Decisão para ${day.habit_name}`}>
                  <button type="button" className={styles.option} aria-pressed={chosen === 'migrated'} onClick={() => choose(day, 'migrated')}>
                    → Migrar
                  </button>
                  <button type="button" className={styles.option} aria-pressed={chosen === 'discarded'} onClick={() => choose(day, 'discarded')}>
                    · Descartar
                  </button>
                </div>
              </div>
            )
          })}
        </section>
      ))}

      {days.length > 0 && (
        <div className={styles.saveBar}>
          <span className={styles.note}>
            {decided.length === 0 ? 'Nada vira falha: o que ficar sem decisão continua neutro até o prazo.'
              : decided.length === 1 ? '1 decisão para salvar' : `${decided.length} decisões para salvar`}
          </span>
          <button type="button" className="btn btn-primary" disabled={decided.length === 0 || save.isPending} onClick={handleSave}>
            {save.isPending ? 'Salvando…' : 'Salvar revisão'}
          </button>
        </div>
      )}
    </div>
  )
}
