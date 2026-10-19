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

  // Extrai info do task_details
  const details = task.task_details as Record<string, unknown> | undefined
  const subject = details?.subject as string | undefined
  const project = details?.project as string | undefined
  const workoutType = details?.workout_type as string | undefined
  const subtitle = subject ?? project ?? workoutType

  const kind = task.category === 'exam' ? styles.exam : task.notify && task.start_time ? styles.notify : ''
  const advanceIcon = {
    planned: <path d="M8 5.5v13l11-6.5z" fill="currentColor" stroke="none" />,
    in_progress: <path d="M5 12.5l4.5 4.5L19 7.5" />,
    done: <><path d="M4 12a8 8 0 1 0 2.4-5.7" /><path d="M4 4v4h4" /></>,
    reviewed: null,
  }[task.status]

  return (
    <motion.div
      className={`${styles.block} ${kind} ${isDone ? styles.blockDone : ''}`}
      layout
      initial={{ opacity: 0, y: 4 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.18 }}
    >
      <div className={styles.timeCol}>
        {task.start_time ? (
          <>
            <time className={styles.time}>{task.start_time.slice(0, 5)}</time>
            {task.duration_minutes && <span className={styles.duration}>{task.duration_minutes} min</span>}
          </>
        ) : (
          <span className={styles.timePlaceholder}>—</span>
        )}
      </div>

      <button type="button" className={styles.content} onClick={onEdit} aria-label={`Editar ${task.title}`}>
        <span className={`${styles.taskTitle} ${isDone ? styles.taskTitleDone : ''}`}>{task.title}</span>
        <span className={styles.subtitle}>
          {[meta.label, subtitle].filter(Boolean).join(' · ')}
          {task.notify && task.start_time && !isDone ? ' · aviso ligado' : ''}
        </span>
        {task.notes && <span className={styles.notes}>{task.notes}</span>}
      </button>

      <div className={styles.right}>
        <span className={`${styles.statusBadge} ${styles[`status_${task.status}`]}`}>{statusMeta.label}</span>
        {statusMeta.next && advanceIcon && (
          <button
            type="button"
            className={styles.advanceBtn}
            onClick={onAdvance}
            disabled={isPending}
            aria-label={`Marcar como ${STATUS_META[statusMeta.next].label}`}
            title={`Marcar como ${STATUS_META[statusMeta.next].label}`}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{advanceIcon}</svg>
          </button>
        )}
        <button type="button" className={styles.deleteBtn} onClick={onDelete} aria-label={`Remover ${task.title}`} title="Remover tarefa">
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18" /></svg>
        </button>
      </div>
    </motion.div>
  )
}
