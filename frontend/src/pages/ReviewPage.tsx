import { useMissedDays, useReviewDay, type MissedDay } from '@/hooks/useWeeklyReview'
import { toast } from '@/lib/toast'
import styles from './ReviewPage.module.css'

// Revisão da semana (habit-review-flow): dias sem registro são estado neutro,
// não falha. A pessoa chega aqui por escolha (menu), decide dia a dia entre
// migrar e descartar — ou não decide, e nada vira falha. Sem contadores.

type Status = 'migrated' | 'discarded'

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

export function ReviewPage() {
  const { data: days = [], isLoading, isError } = useMissedDays()
  const review = useReviewDay()

  const decide = (day: MissedDay, status: Status) => {
    if (day.review?.status === status) return
    review.mutate(
      { habitId: day.habit_id, reviewDate: day.date, status },
      { onError: () => toast.error('Não foi possível salvar sua decisão. Tente de novo.') }
    )
  }

  return (
    <div className={styles.page}>
      <div>
        <h1 className={styles.pageTitle}>Revisão da semana</h1>
        <p className={styles.pageSubtitle}>
          Os dias da última semana em que um hábito ficou sem registro. Decida o que fazer com cada
          um — ou deixe como está.
        </p>
      </div>

      <div className={`glass-card ${styles.legend}`}>
        <span><strong>→ Migrar</strong>: não fiz, mas ainda quero — levo a intenção para a próxima semana.</span>
        <span><strong>· Descartar</strong>: esse dia não vai ser recuperado, e tudo bem.</span>
      </div>

      {isLoading && <p className={styles.empty}>Carregando…</p>}
      {isError && <p className={styles.empty}>Não deu para carregar a revisão agora.</p>}
      {!isLoading && !isError && days.length === 0 && (
        <p className={styles.empty}>Nenhum dia sem registro nos últimos 7 dias.</p>
      )}

      {groupByDay(days).map(([date, habits]) => (
        <section key={date} className={`glass-card ${styles.day}`} aria-label={dayTitle(date)}>
          <h2 className={styles.dayTitle}>{dayTitle(date)}</h2>
          {habits.map(day => (
            <div key={day.habit_id} className={styles.row}>
              <div className={styles.habit}>
                <span className={styles.habitIcon} aria-hidden="true">{day.habit_icon}</span>
                <span className={styles.habitName}>{day.habit_name}</span>
              </div>
              <div className={styles.choice} role="group" aria-label={`Decisão para ${day.habit_name}`}>
                <button
                  type="button"
                  className={styles.option}
                  aria-pressed={day.review?.status === 'migrated'}
                  disabled={review.isPending}
                  onClick={() => decide(day, 'migrated')}
                >
                  → Migrar
                </button>
                <button
                  type="button"
                  className={styles.option}
                  aria-pressed={day.review?.status === 'discarded'}
                  disabled={review.isPending}
                  onClick={() => decide(day, 'discarded')}
                >
                  · Descartar
                </button>
              </div>
            </div>
          ))}
        </section>
      ))}

      {days.length > 0 && (
        <p className={styles.note}>Se você não revisar, esses dias continuam neutros — nada vira falha.</p>
      )}
    </div>
  )
}
