import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useHabits, useHabitStats, useCheckIn, useUndoCheckIn, useCreateHabit, useDeleteHabit } from '@/hooks/useHabits'
import type { CreateHabitInput } from '@/types/habit'
import styles from './HabitsPage.module.css'

const ICONS = ['⭐', '🔥', '💪', '📚', '🧘', '🏃', '💧', '🌱', '🎯', '✍️', '🙏', '💤', '🎵', '🥗', '🧠']
const COLORS = ['#0071E3', '#34C759', '#FF9F0A', '#FF3B30', '#AF52DE', '#FF2D55', '#5AC8FA', '#5856D6']

export function HabitsPage() {
  const { data: habits = [], isLoading } = useHabits()
  const { data: stats } = useHabitStats()
  const checkIn = useCheckIn()
  const undoCheckIn = useUndoCheckIn()
  const createHabit = useCreateHabit()
  const deleteHabit = useDeleteHabit()
  const [showCreate, setShowCreate] = useState(false)
  const [newHabit, setNewHabit] = useState<CreateHabitInput>({
    name: '', icon: '⭐', color: '#0071E3', frequency: 'daily',
    target_days: [1, 2, 3, 4, 5, 6, 7],
  })

  const today = new Date().toLocaleDateString('pt-BR', {
    weekday: 'long', day: 'numeric', month: 'long',
  })

  const handleCheckIn = (habitId: string, completedToday: boolean) => {
    if (completedToday) {
      undoCheckIn.mutate({ id: habitId })
    } else {
      checkIn.mutate({ id: habitId })
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newHabit.name.trim()) return
    await createHabit.mutateAsync(newHabit)
    setShowCreate(false)
    setNewHabit({ name: '', icon: '⭐', color: '#0071E3', frequency: 'daily', target_days: [1,2,3,4,5,6,7] })
  }

  return (
    <div className={styles.page}>
      {/* ─── Header ──────────────────────────────── */}
      <div className={styles.header}>
        <div>
          <p className={styles.dateLabel}>{today}</p>
          <h1 className={styles.pageTitle}>Meus Hábitos</h1>
        </div>
        <button
          className="btn btn-primary"
          onClick={() => setShowCreate(true)}
          id="create-habit-btn"
        >
          + Novo Hábito
        </button>
      </div>

      {/* ─── Stats Cards ─────────────────────────── */}
      {stats && (
        <div className={styles.statsGrid}>
          <StatCard
            label="Hoje"
            value={`${stats.completed_today}/${stats.total_habits}`}
            icon="✦"
            color="var(--color-accent)"
          />
          <StatCard
            label="Total de check-ins"
            value={stats.total_check_ins}
            icon="◈"
            color="var(--color-success)"
          />
          <StatCard
            label="Conclusão (7d)"
            value={`${Math.round(stats.completion_rate_7d * 100)}%`}
            icon="◎"
            color="var(--color-warning)"
          />
        </div>
      )}

      {/* ─── Habits List ─────────────────────────── */}
      {isLoading ? (
        <div className={styles.loadingGrid}>
          {[1,2,3].map(i => <div key={i} className={styles.skeleton} />)}
        </div>
      ) : habits.length === 0 ? (
        <EmptyState onCreateClick={() => setShowCreate(true)} />
      ) : (
        <div className={styles.habitsGrid}>
          <AnimatePresence>
            {habits.map((habit, i) => (
              <motion.div
                key={habit.id}
                initial={{ opacity: 0, y: 16 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.95 }}
                transition={{ delay: i * 0.05, duration: 0.3 }}
              >
                <HabitCard
                  habit={habit}
                  onCheckIn={() => handleCheckIn(habit.id, habit.completed_today)}
                  onDelete={() => deleteHabit.mutate(habit.id)}
                />
              </motion.div>
            ))}
          </AnimatePresence>
        </div>
      )}

      {/* ─── Create Modal ─────────────────────────── */}
      <AnimatePresence>
        {showCreate && (
          <>
            <motion.div
              className={styles.overlay}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              onClick={() => setShowCreate(false)}
            />
            <motion.div
              className={styles.modal}
              initial={{ opacity: 0, y: 32, scale: 0.96 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: 16, scale: 0.97 }}
              transition={{ type: 'spring', damping: 25, stiffness: 300 }}
            >
              <h2 className={styles.modalTitle}>Novo Hábito</h2>

              <form onSubmit={handleCreate} className={styles.createForm}>
                <div className={styles.previewRow}>
                  <div
                    className={styles.iconPreview}
                    style={{ background: newHabit.color + '20', color: newHabit.color }}
                  >
                    {newHabit.icon}
                  </div>
                  <input
                    className="input"
                    placeholder="Nome do hábito..."
                    value={newHabit.name}
                    onChange={e => setNewHabit(h => ({ ...h, name: e.target.value }))}
                    required
                    autoFocus
                    style={{ flex: 1 }}
                  />
                </div>

                {/* Icon picker */}
                <div>
                  <p className={styles.pickerLabel}>Ícone</p>
                  <div className={styles.iconPicker}>
                    {ICONS.map(icon => (
                      <button
                        key={icon}
                        type="button"
                        className={`${styles.iconOption} ${newHabit.icon === icon ? styles.iconSelected : ''}`}
                        onClick={() => setNewHabit(h => ({ ...h, icon }))}
                      >
                        {icon}
                      </button>
                    ))}
                  </div>
                </div>

                {/* Color picker */}
                <div>
                  <p className={styles.pickerLabel}>Cor</p>
                  <div className={styles.colorPicker}>
                    {COLORS.map(color => (
                      <button
                        key={color}
                        type="button"
                        className={`${styles.colorOption} ${newHabit.color === color ? styles.colorSelected : ''}`}
                        style={{ background: color }}
                        onClick={() => setNewHabit(h => ({ ...h, color }))}
                      />
                    ))}
                  </div>
                </div>

                <div className={styles.modalActions}>
                  <button type="button" className="btn btn-ghost" onClick={() => setShowCreate(false)}>
                    Cancelar
                  </button>
                  <button
                    type="submit"
                    className="btn btn-primary"
                    disabled={createHabit.isPending}
                  >
                    {createHabit.isPending ? 'Criando...' : 'Criar Hábito'}
                  </button>
                </div>
              </form>
            </motion.div>
          </>
        )}
      </AnimatePresence>
    </div>
  )
}

// ─── Sub-components ───────────────────────────────────────────

function StatCard({ label, value, icon, color }: {
  label: string; value: number | string; icon: string; color: string
}) {
  return (
    <div className={`glass-card ${styles.statCard}`}>
      <div className={styles.statIcon} style={{ color, background: color + '15' }}>{icon}</div>
      <div>
        <p className={styles.statValue}>{value}</p>
        <p className={styles.statLabel}>{label}</p>
      </div>
    </div>
  )
}

function HabitCard({ habit, onCheckIn, onDelete }: {
  habit: any; onCheckIn: () => void; onDelete: () => void
}) {
  return (
    <div className={`glass-card ${styles.habitCard}`}>
      {/* Left accent bar */}
      <div className={styles.habitAccent} style={{ background: habit.color }} />

      {/* Icon */}
      <div
        className={styles.habitIcon}
        style={{ background: habit.color + '18', color: habit.color }}
      >
        {habit.icon}
      </div>

      {/* Info */}
      <div className={styles.habitInfo}>
        <h3 className={styles.habitName}>{habit.name}</h3>
        {habit.current_streak > 0 && (
          <p className={styles.habitStreak}>🔥 {habit.current_streak} dias seguidos</p>
        )}
      </div>

      {/* Check-in button */}
      <motion.button
        id={`checkin-${habit.id}`}
        className={`${styles.checkBtn} ${habit.completed_today ? styles.checkBtnDone : ''}`}
        style={habit.completed_today ? { background: habit.color, borderColor: habit.color } : {}}
        onClick={onCheckIn}
        whileTap={{ scale: 0.9 }}
        whileHover={{ scale: 1.05 }}
      >
        {habit.completed_today ? '✓' : '○'}
      </motion.button>
    </div>
  )
}

function EmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  return (
    <motion.div
      className={styles.emptyState}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
    >
      <div className={styles.emptyIcon}>✦</div>
      <h3 className={styles.emptyTitle}>Nenhum hábito ainda</h3>
      <p className={styles.emptyText}>Crie seu primeiro hábito e comece a construir sua rotina ideal.</p>
      <button className="btn btn-primary" onClick={onCreateClick}>
        + Criar primeiro hábito
      </button>
    </motion.div>
  )
}
