import { motion } from 'framer-motion'
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

  const completedCount = habits.filter(h => h.completed_today).length

  return (
    <div className={styles.wrapper}>
      <div className={styles.progressHeader}>
        <span className={styles.progressText}>
          {completedCount} de {habits.length} concluídos
        </span>
        <span className={styles.progressPct}>
          {Math.round((completedCount / habits.length) * 100)}%
        </span>
      </div>

      {/* Progress bar */}
      <div className={styles.progressBar}>
        <motion.div
          className={styles.progressFill}
          initial={{ width: 0 }}
          animate={{ width: `${(completedCount / habits.length) * 100}%` }}
          transition={{ duration: 0.6, ease: [0.4, 0, 0.2, 1] }}
        />
      </div>

      {/* Habit rows */}
      <div className={styles.list}>
        {habits.map((habit) => (
          <div key={habit.id} className={styles.row}>
            <div
              className={styles.habitIcon}
              style={{ background: habit.color + '18', color: habit.color }}
            >
              {habit.icon}
            </div>
            <span className={`${styles.habitName} ${habit.completed_today ? styles.done : ''}`}>
              {habit.name}
            </span>
            {habit.current_streak > 0 && (
              <span className={styles.streakBadge}>🔥{habit.current_streak}</span>
            )}
            <motion.button
              className={`${styles.checkBtn} ${habit.completed_today ? styles.checkBtnDone : ''}`}
              style={habit.completed_today ? { background: habit.color, borderColor: habit.color } : {}}
              onClick={() => handleToggle(habit)}
              whileTap={{ scale: 0.85 }}
              id={`dashboard-checkin-${habit.id}`}
            >
              {habit.completed_today ? '✓' : '○'}
            </motion.button>
          </div>
        ))}
      </div>
    </div>
  )
}
