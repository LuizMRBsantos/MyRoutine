import { useState, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import type { Habit, CheckInInput } from '@/types/habit'
import { HabitTimer } from './HabitTimer'
import { MetricCheckIn } from './MetricCheckIn'
import styles from './HabitCard.module.css'

interface HabitCardProps {
  habit: Habit
  onCheckIn: (input: CheckInInput) => void
  onUndo: () => void
  onDelete: () => void
  onClick: () => void
  isPending?: boolean
}

export function HabitCard({ habit, onCheckIn, onUndo, onDelete, onClick, isPending }: HabitCardProps) {
  const [activeMode, setActiveMode] = useState<'none' | 'note' | 'timer' | 'metric'>('none')
  const [note, setNote] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (activeMode === 'note' && textareaRef.current) {
      textareaRef.current.focus()
    }
  }, [activeMode])

  // Reset mode when habit is completed
  useEffect(() => {
    if (habit.completed_today) {
      setActiveMode('none')
    }
  }, [habit.completed_today])

  // Deadline check
  const isPastDeadline = () => {
    if (habit.check_type !== 'deadline' || !habit.deadline_time || habit.completed_today) return false
    const now = new Date()
    const [hours, minutes] = habit.deadline_time.split(':').map(Number)
    const deadline = new Date()
    deadline.setHours(hours, minutes, 0, 0)
    return now > deadline
  }

  const pastDeadline = isPastDeadline()

  const handleActionClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (habit.completed_today) {
      onUndo()
      return
    }

    if (pastDeadline) return

    switch (habit.check_type) {
      case 'timed':
        setActiveMode(v => v === 'timer' ? 'none' : 'timer')
        break
      case 'metric':
        setActiveMode(v => v === 'metric' ? 'none' : 'metric')
        break
      case 'simple':
      case 'deadline':
      default:
        if (activeMode === 'note') {
          setActiveMode('none')
        } else {
          onCheckIn({})
        }
        break
    }
  }

  const renderActionIcon = () => {
    if (habit.completed_today) return '✓'
    if (pastDeadline) return '⏰'
    switch (habit.check_type) {
      case 'timed': return '▶'
      case 'metric': return '📊'
      default: return '○'
    }
  }

  return (
    <div className={`glass-card ${styles.card} ${activeMode !== 'none' ? styles.cardExpanded : ''}`}>
      {/* Accent bar — mantém a cor do hábito sempre, deadline expirado não muda cor */}
      <div className={styles.accent} style={{ background: habit.color }} />

      {/* Main row */}
      <div className={styles.mainRow} onClick={activeMode === 'none' ? onClick : undefined}>
        {/* Icon */}
        <div
          className={styles.icon}
          style={{ background: habit.color + '18', color: habit.color }}
        >
          {habit.icon}
        </div>

        {/* Info */}
        <div className={styles.info}>
          <h3 className={styles.name}>
            {habit.name}
            {habit.check_type === 'deadline' && habit.deadline_time && (
              <span className={styles.deadlineInfo}>
                {' '}• até {habit.deadline_time}
              </span>
            )}
          </h3>
          {habit.current_streak > 0 && (
            <p className={styles.streak}>🔥 {habit.current_streak} dias seguidos</p>
          )}
          {pastDeadline && !habit.completed_today && (
            <p className={styles.missedDeadline}>
              Janela encerrada
            </p>
          )}
        </div>

        {/* Delete */}
        <button
          className={styles.deleteBtn}
          onClick={(e) => { e.stopPropagation(); onDelete() }}
          title="Remover hábito"
        >
          ×
        </button>

        {/* Note button (only for simple/deadline when not completed) */}
        {!habit.completed_today && !pastDeadline && ['simple', 'deadline'].includes(habit.check_type) && (
          <button
            className={styles.noteBtn}
            onClick={(e) => { e.stopPropagation(); setActiveMode(v => v === 'note' ? 'none' : 'note') }}
            title="Registrar com nota"
          >
            ✎
          </button>
        )}

        {/* Action button */}
        <motion.button
          className={`${styles.checkBtn} ${habit.completed_today ? styles.checkBtnDone : ''} ${pastDeadline ? styles.checkBtnMuted : ''}`}
          style={habit.completed_today ? { background: habit.color, borderColor: habit.color } : {}}
          onClick={handleActionClick}
          whileTap={!pastDeadline ? { scale: 0.88 } : {}}
          whileHover={!pastDeadline ? { scale: 1.08 } : {}}
          disabled={isPending || pastDeadline}
          title={pastDeadline ? 'Janela encerrada para hoje' : ''}
        >
          {renderActionIcon()}
        </motion.button>
      </div>

      {/* Expanded Modes */}
      <AnimatePresence>
        {activeMode === 'note' && (
          <motion.form
            className={styles.noteForm}
            onSubmit={(e) => { e.preventDefault(); onCheckIn({ notes: note.trim() || undefined }); setNote(''); setActiveMode('none'); }}
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            onClick={(e) => e.stopPropagation()}
          >
            <textarea
              ref={textareaRef}
              className={styles.noteInput}
              placeholder="Adicione contexto (opcional)..."
              value={note}
              onChange={e => setNote(e.target.value)}
              rows={2}
              onKeyDown={e => {
                if (e.key === 'Enter' && !e.shiftKey) { 
                  e.preventDefault(); 
                  onCheckIn({ notes: note.trim() || undefined })
                  setNote('')
                  setActiveMode('none')
                }
                if (e.key === 'Escape') setActiveMode('none')
              }}
            />
            <div className={styles.noteActions}>
              <button type="button" className={styles.noteCancelBtn} onClick={() => setActiveMode('none')}>
                Cancelar
              </button>
              <button type="submit" className={styles.noteSubmitBtn} style={{ background: habit.color }}>
                Confirmar ✓
              </button>
            </div>
          </motion.form>
        )}

        {activeMode === 'timer' && habit.check_type === 'timed' && habit.timer_minutes && (
          <HabitTimer
            habitId={habit.id}
            color={habit.color}
            targetMinutes={habit.timer_minutes}
            onComplete={(timerSeconds, startedAt, completedAt, isManual, notes) => {
              onCheckIn({ timer_seconds: timerSeconds, started_at: startedAt || undefined, completed_at: completedAt || undefined, is_manual: isManual, notes })
              setActiveMode('none')
            }}
            onCancel={() => setActiveMode('none')}
          />
        )}

        {activeMode === 'metric' && habit.check_type === 'metric' && habit.metric_config && (
          <MetricCheckIn
            habitId={habit.id}
            color={habit.color}
            metricConfig={habit.metric_config}
            onComplete={(metrics, notes) => {
              onCheckIn({ metrics, notes })
              setActiveMode('none')
            }}
            onCancel={() => setActiveMode('none')}
          />
        )}
      </AnimatePresence>
    </div>
  )
}
