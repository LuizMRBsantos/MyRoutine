import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useMonthlyGoals, useCreateGoal, useUpdateGoalStatus, useDeleteGoal } from '@/hooks/useTasks'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import type { MonthlyGoal } from '@/types/task'
import styles from './MonthlyGoalsPanel.module.css'

const GOAL_COLORS = ['#0071E3', '#34C759', '#FF9F0A', '#AF52DE', '#FF2D55', '#5AC8FA']

interface MonthlyGoalsPanelProps {
  viewDate: Date
}

function pad(n: number) {
  return n < 10 ? `0${n}` : `${n}`
}

export function MonthlyGoalsPanel({ viewDate }: MonthlyGoalsPanelProps) {
  const month = `${viewDate.getFullYear()}-${pad(viewDate.getMonth() + 1)}-01`
  const { data: goals = [], isLoading, isError } = useMonthlyGoals(month)
  const createGoal = useCreateGoal()
  const updateStatus = useUpdateGoalStatus()
  const deleteGoal = useDeleteGoal()

  const [newTitle, setNewTitle] = useState('')
  const [newColor, setNewColor] = useState(GOAL_COLORS[0])
  const [deleting, setDeleting] = useState<MonthlyGoal | null>(null)

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault()
    const title = newTitle.trim()
    if (!title) return
    createGoal.mutate(
      { title, month, color: newColor },
      { onSuccess: () => setNewTitle('') }
    )
  }

  const active = goals.filter((g) => g.status === 'active')
  const settled = goals.filter((g) => g.status !== 'active')

  return (
    <div className={`glass-card ${styles.panel}`}>
      <div className={styles.headerRow}>
        <h3 className={styles.title}>Metas do mês</h3>
        <span className={styles.counter}>
          {goals.filter((g) => g.status === 'done').length}/{goals.length || 0} concluídas
        </span>
      </div>

      {/* Quick add */}
      <form onSubmit={handleCreate} className={styles.addRow}>
        <div className={styles.colorDots}>
          {GOAL_COLORS.map((c) => (
            <button
              key={c}
              type="button"
              className={`${styles.colorDot} ${newColor === c ? styles.colorDotSelected : ''}`}
              style={{ background: c }}
              onClick={() => setNewColor(c)}
            />
          ))}
        </div>
        <div className={styles.addInputRow}>
          <input
            className="input"
            placeholder="Nova meta para este mês..."
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
          />
          <button type="submit" className="btn btn-primary" disabled={createGoal.isPending || !newTitle.trim()}>
            +
          </button>
        </div>
      </form>

      {/* Goals */}
      {isLoading ? (
        <p className={styles.hint}>Carregando...</p>
      ) : isError ? (
        <p className={styles.hint}>Não foi possível carregar as metas.</p>
      ) : goals.length === 0 ? (
        <p className={styles.hint}>Nenhuma meta definida para este mês. Sem pressão — defina só o que importa.</p>
      ) : (
        <ul className={styles.list}>
          <AnimatePresence>
            {[...active, ...settled].map((goal) => (
              <motion.li
                key={goal.id}
                className={`${styles.item} ${goal.status !== 'active' ? styles.itemSettled : ''}`}
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.97 }}
              >
                <button
                  className={styles.checkBtn}
                  style={{ borderColor: goal.color ?? 'var(--color-border)', color: goal.color ?? undefined }}
                  title={goal.status === 'done' ? 'Reabrir meta' : 'Concluir meta'}
                  onClick={() =>
                    updateStatus.mutate({
                      id: goal.id,
                      status: goal.status === 'done' ? 'active' : 'done',
                    })
                  }
                >
                  {goal.status === 'done' ? '✓' : ''}
                </button>

                <div className={styles.itemBody}>
                  <span className={styles.itemTitle}>{goal.title}</span>
                  {goal.status === 'abandoned' && (
                    <span className={styles.abandonedTag}>descartada</span>
                  )}
                </div>

                {goal.status === 'active' && (
                  <button
                    className={styles.softBtn}
                    title="Descartar conscientemente"
                    onClick={() => updateStatus.mutate({ id: goal.id, status: 'abandoned' })}
                  >
                    ↷
                  </button>
                )}
                <button
                  className={styles.softBtn}
                  title="Remover"
                  onClick={() => setDeleting(goal)}
                >
                  ×
                </button>
              </motion.li>
            ))}
          </AnimatePresence>
        </ul>
      )}

      <ConfirmDialog
        open={!!deleting}
        title="Remover meta?"
        message={`"${deleting?.title}" será removida deste mês.`}
        confirmLabel="Remover"
        onConfirm={() => {
          if (deleting) deleteGoal.mutate(deleting.id)
          setDeleting(null)
        }}
        onCancel={() => setDeleting(null)}
      />
    </div>
  )
}
