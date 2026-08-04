import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useHabitTimer } from '@/hooks/useHabitTimer'
import styles from './HabitTimer.module.css'

interface HabitTimerProps {
  habitId: string
  color: string
  targetMinutes: number
  onComplete: (timerSeconds: number, startedAt: string | null, completedAt: string | null, isManual?: boolean, notes?: string) => void
  onCancel: () => void
}

export function HabitTimer({ habitId, color, targetMinutes, onComplete, onCancel }: HabitTimerProps) {
  const timer = useHabitTimer(habitId, targetMinutes)
  const [showManual, setShowManual] = useState(false)
  const [manualNotes, setManualNotes] = useState('')
  const [manualMinutes, setManualMinutes] = useState(targetMinutes)

  const handleManualSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onComplete(manualMinutes * 60, null, new Date().toISOString(), true, manualNotes.trim() || undefined)
    timer.clear()
  }

  const handleComplete = () => {
    onComplete(timer.elapsedSeconds, timer.startedAt, new Date().toISOString(), false)
    timer.clear()
  }

  return (
    <motion.div
      className={styles.timerWrapper}
      initial={{ opacity: 0, height: 0 }}
      animate={{ opacity: 1, height: 'auto' }}
      exit={{ opacity: 0, height: 0 }}
      transition={{ duration: 0.3 }}
      onClick={(e) => e.stopPropagation()} // Prevent clicking the card behind it
    >
      <div className={styles.timerContent}>
        {/* Progress Ring & Display */}
        <div className={styles.ringContainer}>
          <svg viewBox="0 0 120 120" className={styles.ring}>
            <circle cx="60" cy="60" r="54" className={styles.ringTrack} />
            <circle
              cx="60" cy="60" r="54"
              className={styles.ringFill}
              style={{ stroke: color }}
              strokeDasharray={`${2 * Math.PI * 54}`}
              strokeDashoffset={`${2 * Math.PI * 54 * (1 - timer.progressPercent / 100)}`}
            />
          </svg>
          <div className={styles.timeDisplay}>
            {timer.formattedTime}
            <span className={styles.timeTarget}>/ {targetMinutes}m</span>
          </div>
        </div>

        {/* Controls */}
        <div className={styles.controls}>
          {!timer.isRunning && !timer.isCompleted && (
            <button className={styles.controlBtn} style={{ background: color, color: '#fff' }} onClick={timer.start}>
              ▶ Iniciar
            </button>
          )}
          
          {timer.isRunning && (
            <button className={styles.controlBtn} style={{ background: '#FF9F0A', color: '#fff' }} onClick={timer.pause}>
              ⏸ Pausar
            </button>
          )}

          {timer.elapsedSeconds > 0 && !timer.isRunning && !timer.isCompleted && (
            <button className={styles.controlBtn} onClick={timer.stopAndReset}>
              ⏹ Resetar
            </button>
          )}

          {timer.isCompleted && (
            <button className={styles.controlBtn} style={{ background: '#34C759', color: '#fff' }} onClick={handleComplete}>
              ✓ Concluir Hábito
            </button>
          )}
        </div>

        {/* Cancel & Manual Entry Links */}
        <div className={styles.footerLinks}>
          <button className={styles.linkBtn} onClick={onCancel}>Fechar timer</button>
          <span className={styles.dot}>•</span>
          <button className={styles.linkBtn} onClick={() => setShowManual(v => !v)}>
            Registrar manualmente (esqueci)
          </button>
        </div>

        {/* Manual Entry Form */}
        <AnimatePresence>
          {showManual && (
            <motion.form
              className={styles.manualForm}
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: 'auto' }}
              exit={{ opacity: 0, height: 0 }}
              onSubmit={handleManualSubmit}
            >
              <div className={styles.inputGroup}>
                <label>Minutos realizados</label>
                <input 
                  type="number" 
                  min="1" 
                  value={manualMinutes} 
                  onChange={e => setManualMinutes(parseInt(e.target.value) || 0)} 
                  className={styles.input}
                />
              </div>
              <div className={styles.inputGroup}>
                <label>Notas (opcional)</label>
                <input 
                  type="text" 
                  value={manualNotes} 
                  onChange={e => setManualNotes(e.target.value)} 
                  className={styles.input}
                  placeholder="Esqueci de ligar o timer..."
                />
              </div>
              <button type="submit" className={styles.submitManualBtn} style={{ background: color }}>
                Salvar Manual
              </button>
            </motion.form>
          )}
        </AnimatePresence>
      </div>
    </motion.div>
  )
}
