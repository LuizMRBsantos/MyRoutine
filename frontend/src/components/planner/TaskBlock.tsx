import { motion } from 'framer-motion'
import type { Task } from '@/types/task'
import { CATEGORY_META, STATUS_META } from '@/types/task'
import styles from './TaskBlock.module.css'

interface TaskBlockProps {
  task: Task
  onAdvance: () => void
  onEdit: () => void
  onDelete: () => void
  isPending?: boolean
}

export function TaskBlock({ task, onAdvance, onEdit, onDelete, isPending }: TaskBlockProps) {
  const meta = CATEGORY_META[task.category]
  const statusMeta = STATUS_META[task.status]
  const isDone = task.status === 'done' || task.status === 'reviewed'
  const color = task.color ?? meta.color

  // Extrai info do task_details
  const details = task.task_details as Record<string, unknown> | undefined
  const subject = details?.subject as string | undefined
  const project = details?.project as string | undefined
  const workoutType = details?.workout_type as string | undefined
  const subtitle = subject ?? project ?? workoutType

  return (
    <motion.div
      className={`${styles.block} ${isDone ? styles.blockDone : ''}`}
      style={{ borderLeftColor: color }}
      layout
      initial={{ opacity: 0, x: -8 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 8 }}
    >
      {/* Lado esquerdo: horário */}
      <div className={styles.timeCol}>
        {task.start_time ? (
          <>
            <span className={styles.time}>{task.start_time.slice(0, 5)}</span>
            {task.duration_minutes && (
              <span className={styles.duration}>{task.duration_minutes}min</span>
            )}
          </>
        ) : (
          <span className={styles.timePlaceholder}>—</span>
        )}
      </div>

      {/* Ícone da categoria */}
      <div className={styles.iconCol} style={{ background: color + '18', color }}>
        {meta.icon}
      </div>

      {/* Conteúdo */}
      <div className={styles.content}>
        <p className={`${styles.taskTitle} ${isDone ? styles.taskTitleDone : ''}`}>
          {task.title}
        </p>
        {subtitle && (
          <p className={styles.subtitle}>{subtitle}</p>
        )}
        {task.notes && (
          <p className={styles.notes}>{task.notes}</p>
        )}
      </div>

      {/* Status badge */}
      <div className={styles.right}>
        <span
          className={`${styles.statusBadge} ${styles[`status_${task.status}`]}`}
          style={task.status === 'in_progress' ? { background: color + '22', color } : {}}
        >
          {statusMeta.label}
        </span>

        {/* Botão avançar status */}
        {statusMeta.next && (
          <motion.button
            className={styles.advanceBtn}
            style={{ borderColor: color, color }}
            onClick={onAdvance}
            disabled={isPending}
            whileTap={{ scale: 0.9 }}
            whileHover={{ scale: 1.05 }}
            title={`Marcar como ${STATUS_META[statusMeta.next].label}`}
          >
            {task.status === 'planned'     && '▶'}
            {task.status === 'in_progress' && '✓'}
            {task.status === 'done'        && '✦'}
          </motion.button>
        )}

        {/* Botão editar */}
        <button className={styles.editBtn} onClick={onEdit} title="Editar tarefa">
          ✏
        </button>

        {/* Botão remover */}
        <button className={styles.deleteBtn} onClick={onDelete} title="Remover tarefa">
          ×
        </button>
      </div>
    </motion.div>
  )
}
