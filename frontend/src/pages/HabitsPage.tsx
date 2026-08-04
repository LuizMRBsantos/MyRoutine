import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  useHabits, useHabitStats, useCheckIn, useUndoCheckIn,
  useCreateHabit, useUpdateHabit, useDeleteHabit,
} from '@/hooks/useHabits'
import { HabitCard } from '@/components/habits/HabitCard'
import { HabitDrawer } from '@/components/habits/HabitDrawer'
import { HabitFormModal } from '@/components/habits/HabitFormModal'
import { WeeklyReview } from '@/components/habits/WeeklyReview'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { toast } from '@/lib/toast'
import type { Habit, CheckInInput } from '@/types/habit'
import styles from './HabitsPage.module.css'

const TIME_GROUPS = [
  { key: 'morning', label: '☀️ Manhã' },
  { key: 'afternoon', label: '🌤 Tarde' },
  { key: 'evening', label: '🌙 Noite' },
  { key: 'anytime', label: '✦ Qualquer hora' },
] as const

export function HabitsPage() {
  const { data: habits = [], isLoading, isError } = useHabits()
  const { data: stats } = useHabitStats()
  const checkIn = useCheckIn()
  const undoCheckIn = useUndoCheckIn()
  const createHabit = useCreateHabit()
  const updateHabit = useUpdateHabit()
  const deleteHabit = useDeleteHabit()

  const [showCreate, setShowCreate] = useState(false)
  const [editingHabit, setEditingHabit] = useState<Habit | null>(null)
  const [selectedHabit, setSelectedHabit] = useState<Habit | null>(null)
  const [deletingHabit, setDeletingHabit] = useState<Habit | null>(null)

  const today = new Date().toLocaleDateString('pt-BR', {
    weekday: 'long', day: 'numeric', month: 'long',
  })

  const handleCheckIn = (habit: Habit, input: CheckInInput) => {
    if (habit.completed_today) {
      undoCheckIn.mutate({ id: habit.id })
    } else {
      checkIn.mutate({ id: habit.id, ...input })
    }
  }

  const groupedHabits = TIME_GROUPS.map(group => ({
    ...group,
    habits: habits.filter(h => h.time_of_day === group.key),
  })).filter(g => g.habits.length > 0)

  return (
    <div className={styles.page}>
      {/* ─── Header ─────────────────────────────── */}
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

      {/* ─── Stats ──────────────────────────────── */}
      {stats && (
        <div className={styles.statsGrid}>
          <div className={`glass-card ${styles.statCard}`}>
            <div className={styles.statIcon} style={{ color: 'var(--color-accent)', background: 'var(--color-accent-light)' }}>✦</div>
            <div>
              <p className={styles.statValue}>{stats.completed_today}/{stats.total_habits}</p>
              <p className={styles.statLabel}>Hoje</p>
            </div>
          </div>
          <div className={`glass-card ${styles.statCard}`}>
            <div className={styles.statIcon} style={{ color: 'var(--color-success)', background: 'var(--color-success-light)' }}>◈</div>
            <div>
              <p className={styles.statValue}>{stats.total_check_ins}</p>
              <p className={styles.statLabel}>Total check-ins</p>
            </div>
          </div>
          <div className={`glass-card ${styles.statCard}`}>
            <div className={styles.statIcon} style={{ color: 'var(--color-warning)', background: 'var(--color-warning-light)' }}>◎</div>
            <div>
              <p className={styles.statValue}>{Math.round((stats.completion_rate_7d ?? 0) * 100)}%</p>
              <p className={styles.statLabel}>Conclusão (7d)</p>
            </div>
          </div>
        </div>
      )}

      {/* ─── Weekly Review ─────────────────────────────────── */}
      <WeeklyReview />

      {/* ─── Habits by group ─────────────────────── */}
      {isLoading ? (
        <div className={styles.skeletonGrid}>
          {[1,2,3].map(i => <div key={i} className={styles.skeleton} />)}
        </div>
      ) : isError ? (
        <div className={styles.emptyState}>
          <div className={styles.emptyIcon}>◌</div>
          <h3 className={styles.emptyTitle}>Não foi possível carregar seus hábitos</h3>
          <p className={styles.emptyText}>Verifique sua conexão e tente novamente.</p>
        </div>
      ) : habits.length === 0 ? (
        <motion.div className={styles.emptyState} initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
          <div className={styles.emptyIcon}>✦</div>
          <h3 className={styles.emptyTitle}>Nenhum hábito ainda</h3>
          <p className={styles.emptyText}>Crie seu primeiro hábito e comece a construir sua rotina ideal.</p>
          <button className="btn btn-primary" onClick={() => setShowCreate(true)}>
            + Criar primeiro hábito
          </button>
        </motion.div>
      ) : (
        <div className={styles.groups}>
          {groupedHabits.map((group) => (
            <div key={group.key} className={styles.group}>
              <h2 className={styles.groupLabel}>{group.label}</h2>
              <div className={styles.habitsList}>
                <AnimatePresence>
                  {group.habits.map((habit, i) => (
                    <motion.div
                      key={habit.id}
                      initial={{ opacity: 0, y: 12 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={{ opacity: 0, scale: 0.95 }}
                      transition={{ delay: i * 0.04, duration: 0.25 }}
                    >
                      <HabitCard
                        habit={habit}
                        onCheckIn={(input) => handleCheckIn(habit, input)}
                        onUndo={() => undoCheckIn.mutate({ id: habit.id })}
                        onDelete={() => setDeletingHabit(habit)}
                        onClick={() => setSelectedHabit(habit)}
                        isPending={checkIn.isPending || undoCheckIn.isPending || deleteHabit.isPending}
                      />
                    </motion.div>
                  ))}
                </AnimatePresence>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* ─── Create / Edit Modal ─────────────────────────── */}
      <AnimatePresence>
        {(showCreate || editingHabit) && (
          <HabitFormModal
            key={editingHabit?.id ?? 'create'}
            habit={editingHabit ?? undefined}
            isPending={createHabit.isPending || updateHabit.isPending}
            onSave={(input) => {
              if (editingHabit) {
                updateHabit.mutate(
                  { id: editingHabit.id, input },
                  {
                    onSuccess: (updated) => {
                      setEditingHabit(null)
                      setSelectedHabit((s) => (s?.id === updated.id ? updated : s))
                      toast.success('Hábito atualizado')
                    },
                  }
                )
              } else {
                createHabit.mutate(input, {
                  onSuccess: () => {
                    setShowCreate(false)
                    toast.success('Hábito criado')
                  },
                })
              }
            }}
            onClose={() => { setShowCreate(false); setEditingHabit(null) }}
          />
        )}
      </AnimatePresence>

      {/* ─── Delete confirmation ─────────────────────────── */}
      <ConfirmDialog
        open={!!deletingHabit}
        title="Excluir hábito?"
        message={`"${deletingHabit?.name}" será arquivado. Seu histórico de check-ins é preservado.`}
        confirmLabel="Excluir"
        onConfirm={() => {
          if (deletingHabit) {
            deleteHabit.mutate(deletingHabit.id, {
              onSuccess: () => toast.success('Hábito excluído'),
            })
          }
          setDeletingHabit(null)
        }}
        onCancel={() => setDeletingHabit(null)}
      />

      {/* ─── Habit Drawer ─────────────────────────── */}
      <HabitDrawer
        habit={selectedHabit}
        onClose={() => setSelectedHabit(null)}
        onEdit={(habit) => { setSelectedHabit(null); setEditingHabit(habit) }}
      />
    </div>
  )
}
