import { AnimatePresence, motion } from 'framer-motion'
import { useHabits, useCheckIn } from '@/hooks/useHabits'
import { useCheckInDock } from '@/store/checkInDock'
import { toast } from '@/lib/toast'
import { HabitTimer } from './HabitTimer'
import { MetricCheckIn } from './MetricCheckIn'
import styles from './CheckInDock.module.css'

// Janelinha no canto inferior direito para hábitos que não fecham com um
// toque: com timer (iniciar, pausar, concluir) ou com medidas (km, páginas).
// Fica no layout, então continua aberta ao trocar de tela.
export function CheckInDock() {
  const habitId = useCheckInDock(s => s.habitId)
  // Fechada, não busca nada: só abre a conexão de dados quando há hábito.
  return (
    <AnimatePresence>
      {habitId && <DockPanel key={habitId} habitId={habitId} />}
    </AnimatePresence>
  )
}

function DockPanel({ habitId }: { habitId: string }) {
  const close = useCheckInDock(s => s.close)
  const { data: habits = [] } = useHabits()
  const checkIn = useCheckIn()
  const habit = habits.find(h => h.id === habitId)

  const save = (input: Parameters<typeof checkIn.mutate>[0]) =>
    checkIn.mutate(input, {
      onSuccess: () => {
        toast.success(`${habit?.name ?? 'Hábito'} registrado`)
        close()
      },
      // Erro: o aviso automático (queryClient) já aparece; a janelinha fica aberta.
    })

  const visible = habit && !habit.completed_today &&
    ((habit.check_type === 'timed' && habit.timer_minutes) || (habit.check_type === 'metric' && habit.metric_config))

  if (!visible) return null
  return (
    <motion.aside
      className={`glass-card ${styles.dock}`}
      aria-label={`Check-in: ${habit.name}`}
      initial={{ opacity: 0, y: 24 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 24 }}
      transition={{ duration: 0.25 }}
    >
      <div className={styles.header}>
        <span className={styles.icon} style={{ background: habit.color + '18', color: habit.color }}>
          {habit.icon}
        </span>
        <span className={styles.name}>{habit.name}</span>
        <button type="button" className={styles.close} aria-label="Fechar" onClick={close}>×</button>
      </div>

      {habit.check_type === 'timed' && habit.timer_minutes && (
        <HabitTimer
          habitId={habit.id}
          color={habit.color}
          targetMinutes={habit.timer_minutes}
          onComplete={(timerSeconds, startedAt, completedAt, isManual, notes) =>
            save({
              id: habit.id, timer_seconds: timerSeconds, started_at: startedAt || undefined,
              completed_at: completedAt || undefined, is_manual: isManual, notes,
            })}
          onCancel={close}
        />
      )}

      {habit.check_type === 'metric' && habit.metric_config && (
        <MetricCheckIn
          habitId={habit.id}
          color={habit.color}
          metricConfig={habit.metric_config}
          onComplete={(metrics, notes) => save({ id: habit.id, metrics, notes })}
          onCancel={close}
        />
      )}
    </motion.aside>
  )
}
