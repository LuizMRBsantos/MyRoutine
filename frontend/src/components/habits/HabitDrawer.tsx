import { motion, AnimatePresence } from 'framer-motion'
import type { Habit, HabitLog } from '@/types/habit'
import { useHabitLogs } from '@/hooks/useHabits'
import { HabitHeatmap } from './HabitHeatmap'
import styles from './HabitDrawer.module.css'

interface HabitDrawerProps {
  habit: Habit | null
  onClose: () => void
  onEdit: (habit: Habit) => void
}

function toDateStr(d: Date) {
  const pad = (n: number) => (n < 10 ? `0${n}` : `${n}`)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function formatLogDetail(log: HabitLog, habit: Habit): string | null {
  const parts: string[] = []
  if (log.timer_seconds != null) {
    const min = Math.round(log.timer_seconds / 60)
    parts.push(`${min} min`)
  }
  if (log.metrics) {
    for (const [key, value] of Object.entries(log.metrics)) {
      const field = habit.metric_config?.find((m) => m.key === key)
      parts.push(`${value}${field?.unit ? ` ${field.unit}` : ''} ${field?.label ?? key}`.trim())
    }
  }
  if (log.notes) parts.push(log.notes)
  return parts.length ? parts.join(' · ') : null
}

export function HabitDrawer({ habit, onClose, onEdit }: HabitDrawerProps) {
  const to = toDateStr(new Date())
  const from = toDateStr(new Date(Date.now() - 89 * 24 * 60 * 60 * 1000))
  const { data: logs = [] } = useHabitLogs(habit?.id, from, to)

  // Heatmap individual: cada dia com log conta 1
  const heatmapData = logs.map((l) => ({ date: l.logged_date, habits_completed: 1 }))
  const recentLogs = logs.slice(0, 14)

  return (
    <AnimatePresence>
      {habit && (
        <>
          {/* Backdrop */}
          <motion.div
            className={styles.backdrop}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
          />

          {/* Drawer */}
          <motion.aside
            className={styles.drawer}
            initial={{ x: '100%' }}
            animate={{ x: 0 }}
            exit={{ x: '100%' }}
            transition={{ type: 'spring', damping: 28, stiffness: 280 }}
          >
            {/* Header */}
            <div className={styles.header}>
              <div
                className={styles.habitIcon}
                style={{ background: habit.color + '18', color: habit.color }}
              >
                {habit.icon}
              </div>
              <div className={styles.habitMeta}>
                <h2 className={styles.habitName}>{habit.name}</h2>
                {habit.description && (
                  <p className={styles.habitDesc}>{habit.description}</p>
                )}
              </div>
              <button
                className={styles.editBtn}
                onClick={() => onEdit(habit)}
                title="Editar hábito"
                id="edit-habit-btn"
              >
                Editar
              </button>
              <button className={styles.closeBtn} onClick={onClose} id="close-habit-drawer">×</button>
            </div>

            {/* Stats row */}
            <div className={styles.statsRow}>
              <div className={styles.statItem}>
                <span className={styles.statValue}>{habit.current_streak}</span>
                <span className={styles.statLabel}>Sequência atual</span>
              </div>
              <div className={styles.statDivider} />
              <div className={styles.statItem}>
                <span className={styles.statValue}>{logs.length}</span>
                <span className={styles.statLabel}>Check-ins (90d)</span>
              </div>
              <div className={styles.statDivider} />
              <div className={styles.statItem}>
                <span className={styles.statValue} style={{ color: habit.color }}>
                  {habit.completed_today ? '✓' : '—'}
                </span>
                <span className={styles.statLabel}>Hoje</span>
              </div>
            </div>

            {/* Heatmap — apenas deste hábito */}
            <div className={styles.section}>
              <h3 className={styles.sectionTitle}>Consistência (90 dias)</h3>
              <HabitHeatmap data={heatmapData} days={90} />
            </div>

            {/* Histórico recente */}
            {recentLogs.length > 0 && (
              <div className={styles.section}>
                <h3 className={styles.sectionTitle}>Histórico recente</h3>
                <ul className={styles.logList}>
                  {recentLogs.map((log) => {
                    const detail = formatLogDetail(log, habit)
                    const day = new Date(log.logged_date + 'T12:00')
                    return (
                      <li key={log.id} className={styles.logItem}>
                        <span className={styles.logDate}>
                          {day.toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' })}
                        </span>
                        <span className={styles.logDetail}>
                          {detail ?? 'Check-in'}
                        </span>
                        {log.source_type !== 'manual' && (
                          <span className={styles.logSource}>
                            {{ task: 'via tarefa', track_day: 'via diário', study_session: 'via estudo', import: 'importado' }[log.source_type] ?? log.source_type}
                          </span>
                        )}
                      </li>
                    )
                  })}
                </ul>
              </div>
            )}

            {/* Frequency info */}
            <div className={styles.section}>
              <h3 className={styles.sectionTitle}>Configuração</h3>
              <div className={styles.configRow}>
                <span className={styles.configLabel}>Frequência</span>
                <span className={styles.configValue}>
                  {habit.frequency === 'daily' ? 'Diário' : `${habit.target_days.length} dias/semana`}
                </span>
              </div>
              <div className={styles.configRow}>
                <span className={styles.configLabel}>Período do dia</span>
                <span className={styles.configValue}>
                  {{ morning: '☀️ Manhã', afternoon: '🌤 Tarde', evening: '🌙 Noite', anytime: '✦ Qualquer hora' }[habit.time_of_day]}
                </span>
              </div>
              <div className={styles.configRow}>
                <span className={styles.configLabel}>Categoria</span>
                <span className={styles.configValue}>
                  {{ general: 'Geral', health: 'Saúde', study: 'Estudos' }[habit.category] ?? habit.category}
                </span>
              </div>
            </div>
          </motion.aside>
        </>
      )}
    </AnimatePresence>
  )
}
