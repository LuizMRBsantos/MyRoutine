import { Link } from 'react-router-dom'
import type { Habit } from '@/types/habit'
import { useCheckIn, useUndoCheckIn } from '@/hooks/useHabits'
import { useCheckInDock } from '@/store/checkInDock'
import styles from './TodayHabits.module.css'

interface TodayHabitsProps {
  habits: Habit[]
  isLoading: boolean
  // Falha ao carregar (ex.: sem internet) não é "lista vazia": dizer
  // "nenhum hábito" faria parecer que os hábitos sumiram.
  isError?: boolean
}

const PERIODS: [Habit['time_of_day'], string][] = [
  ['morning', 'Manhã'], ['afternoon', 'Tarde'], ['evening', 'Noite'], ['anytime', 'A qualquer hora'],
]

export function TodayHabits({ habits: allHabits, isLoading, isError = false }: TodayHabitsProps) {
  const checkIn = useCheckIn()
  const undoCheckIn = useUndoCheckIn()
  const openDock = useCheckInDock(s => s.open)

  // Só os hábitos de hoje (ex.: "correr" de terça e quinta não aparece na
  // quarta). Um feito fora do dia continua visível, para poder desfazer.
  const habits = allHabits.filter(h => h.scheduled_today || h.completed_today)

  const handleToggle = (habit: Habit) => {
    if (habit.completed_today) {
      undoCheckIn.mutate({ id: habit.id })
      return
    }
    // Timer e medidas precisam de mais que um toque: abrem a janelinha.
    if (habit.check_type === 'timed' || habit.check_type === 'metric') {
      openDock(habit.id)
      return
    }
    checkIn.mutate({ id: habit.id })
  }

  if (isLoading) {
    return (
      <div className={styles.skeletonList}>
        {[1, 2, 3].map(i => <div key={i} className={styles.skeleton} />)}
      </div>
    )
  }

  if (isError && habits.length === 0) {
    return (
      <p className={styles.empty} role="status">
        Não deu para carregar seus hábitos agora. Eles aparecem assim que a conexão voltar.
      </p>
    )
  }

  if (habits.length === 0) {
    return allHabits.length > 0
      ? <p className={styles.empty}>Nenhum hábito programado para hoje.</p>
      : <p className={styles.empty}>Nenhum hábito para hoje. <Link to="/habits">Criar hábito →</Link></p>
  }

  // Agrupados pelo período do dia; sem contagem nem barra (product-constitution).
  const groups = PERIODS
    .map(([key, label]) => ({ label, items: habits.filter(h => (h.time_of_day ?? 'anytime') === key) }))
    .filter(g => g.items.length > 0)

  return (
    <div className={styles.wrapper}>
      {groups.map(group => (
        <div key={group.label} className={styles.group}>
          <p className={styles.groupLabel}>{group.label}</p>
          <ul className={styles.list}>
            {group.items.map(habit => {
              const timed = habit.check_type === 'timed' && !habit.completed_today
              return (
                <li key={habit.id} className={styles.row}>
                  <span className={styles.habitIcon} aria-hidden="true">{habit.icon}</span>
                  <span className={`${styles.habitName} ${habit.completed_today ? styles.done : ''}`}>
                    {habit.name}
                  </span>
                  {timed && habit.timer_minutes && (
                    <span className={styles.meta}>{habit.timer_minutes} min</span>
                  )}
                  <button
                    type="button"
                    className={styles.checkBtn}
                    onClick={() => handleToggle(habit)}
                    id={`dashboard-checkin-${habit.id}`}
                    aria-label={
                      habit.completed_today ? `Desfazer ${habit.name}`
                        : timed ? `Abrir timer de ${habit.name}`
                        : `Marcar ${habit.name}`
                    }
                  >
                    <span className={`${styles.circle} ${habit.completed_today ? styles.circleDone : ''}`}>
                      {habit.completed_today && (
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                          <path d="M5 12.5l4.5 4.5L19 7.5" />
                        </svg>
                      )}
                      {timed && (
                        <svg width="11" height="11" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                          <path d="M8 5.5v13l11-6.5z" />
                        </svg>
                      )}
                    </span>
                  </button>
                </li>
              )
            })}
          </ul>
        </div>
      ))}
    </div>
  )
}
