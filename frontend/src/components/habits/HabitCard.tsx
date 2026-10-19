import { useState, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import type { Habit, CheckInInput } from '@/types/habit'
import { HabitTimer } from './HabitTimer'
import { MetricCheckIn } from './MetricCheckIn'
import { scheduleLabel } from '@/lib/habitSchedule'
import styles from './HabitCard.module.css'

interface HabitCardProps {
  habit: Habit
  onCheckIn: (input: CheckInInput) => void
  onUndo: () => void
  onDelete: () => void
  onClick: () => void
  isPending?: boolean
  // Últimos 7 dias, um caractere por dia (códigos de /habits/consistency).
  recent?: string
}

function typeLabel(h: Habit): string | null {
  if (h.check_type === 'timed' && h.timer_minutes) return `timer de ${h.timer_minutes} min`
  if (h.check_type === 'deadline' && h.deadline_time) return `até ${h.deadline_time.slice(0, 5)}`
  if (h.check_type === 'metric' && h.metric_config?.length) {
    return `registra ${h.metric_config.map(m => m.unit || m.label).join(', ')}`
  }
  return null
}

const DOT: Record<string, string> = { x: 'dotDone', o: 'dotOpen', '.': 'dotToday', '-': 'dotOff', ' ': 'dotOff' }

export function HabitCard({ habit, onCheckIn, onUndo, onDelete, onClick, isPending, recent }: HabitCardProps) {
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

  const done7 = recent ? [...recent].filter(c => c === 'x').length : 0
  const due7 = recent ? [...recent].filter(c => c === 'x' || c === 'o').length : 0
  const details = [
    scheduleLabel(habit.target_days),
    typeLabel(habit),
    recent && due7 > 0 ? `${done7} de ${due7} nos últimos 7 dias` : null,
  ].filter(Boolean).join(' · ')

  const actionLabel = habit.completed_today ? `Desfazer ${habit.name}`
    : pastDeadline ? 'Janela encerrada para hoje'
    : habit.check_type === 'timed' ? `Abrir timer de ${habit.name}`
    : habit.check_type === 'metric' ? `Registrar medidas de ${habit.name}`
    : `Marcar ${habit.name}`

  return (
    <div className={`${styles.card} ${activeMode !== 'none' ? styles.cardExpanded : ''}`}>
      <div className={styles.mainRow} onClick={activeMode === 'none' ? onClick : undefined}>
        <span className={styles.icon} aria-hidden="true">{habit.icon}</span>

        <div className={styles.info}>
          <h3 className={`${styles.name} ${habit.completed_today ? styles.nameDone : ''}`}>{habit.name}</h3>
          <p className={styles.details}>
            {pastDeadline ? 'Janela encerrada hoje' : details}
          </p>
        </div>

        {recent && (
          <span className={styles.dots} aria-hidden="true">
            {[...recent].map((c, i) => <span key={i} className={`${styles.dot} ${styles[DOT[c] ?? 'dotOff']}`} />)}
          </span>
        )}

        <button
          type="button"
          className={`${styles.ghostBtn} ${styles.deleteBtn}`}
          onClick={(e) => { e.stopPropagation(); onDelete() }}
          aria-label={`Remover ${habit.name}`}
          title="Remover hábito"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18" /></svg>
        </button>

        {!habit.completed_today && !pastDeadline && ['simple', 'deadline'].includes(habit.check_type) && (
          <button
            type="button"
            className={styles.ghostBtn}
            onClick={(e) => { e.stopPropagation(); setActiveMode(v => v === 'note' ? 'none' : 'note') }}
            aria-label={`Registrar ${habit.name} com nota`}
            title="Registrar com nota"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M4 20h4l10-10-4-4L4 16z" /><path d="M13 7l4 4" /></svg>
          </button>
        )}

        <button
          type="button"
          className={styles.checkBtn}
          onClick={handleActionClick}
          disabled={isPending || pastDeadline}
          aria-label={actionLabel}
          title={pastDeadline ? 'Janela encerrada para hoje' : undefined}
        >
          <span className={`${styles.circle} ${habit.completed_today ? styles.circleDone : ''} ${pastDeadline ? styles.circleMuted : ''}`}>
            {habit.completed_today ? (
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="M5 12.5l4.5 4.5L19 7.5" /></svg>
            ) : habit.check_type === 'timed' ? (
              <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M8 5.5v13l11-6.5z" /></svg>
            ) : habit.check_type === 'metric' ? (
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" aria-hidden="true"><path d="M12 6v12M6 12h12" /></svg>
            ) : null}
          </span>
        </button>
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
              <button type="submit" className={styles.noteSubmitBtn}>
                Registrar
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
