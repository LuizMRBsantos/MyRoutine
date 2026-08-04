import { motion, AnimatePresence } from 'framer-motion'
import type { Habit } from '@/types/habit'
import { useHeatmap } from '@/hooks/useHabits'
import { HabitHeatmap } from './HabitHeatmap'
import styles from './HabitDrawer.module.css'

interface HabitDrawerProps {
  habit: Habit | null
  onClose: () => void
}

export function HabitDrawer({ habit, onClose }: HabitDrawerProps) {
  const { data: heatmapData = [] } = useHeatmap()

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
                <span className={styles.statValue} style={{ color: habit.color }}>
                  {habit.completed_today ? '✓' : '—'}
                </span>
                <span className={styles.statLabel}>Hoje</span>
              </div>
            </div>

            {/* Heatmap */}
            <div className={styles.section}>
              <h3 className={styles.sectionTitle}>Consistência (90 dias)</h3>
              <HabitHeatmap data={heatmapData} days={90} />
            </div>

            {/* Frequency info */}
            <div className={styles.section}>
              <h3 className={styles.sectionTitle}>Configuração</h3>
              <div className={styles.configRow}>
                <span className={styles.configLabel}>Frequência</span>
                <span className={styles.configValue}>
                  {habit.frequency === 'daily' ? 'Diário' : habit.frequency}
                </span>
              </div>
              <div className={styles.configRow}>
                <span className={styles.configLabel}>Período do dia</span>
                <span className={styles.configValue}>
                  {{ morning: '☀️ Manhã', afternoon: '🌤 Tarde', evening: '🌙 Noite', anytime: '✦ Qualquer hora' }[habit.time_of_day]}
                </span>
              </div>
            </div>
          </motion.aside>
        </>
      )}
    </AnimatePresence>
  )
}
