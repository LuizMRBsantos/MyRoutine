import { useState, useMemo } from 'react'
import { motion } from 'framer-motion'
import type { MetricField } from '@/types/habit'
import styles from './MetricCheckIn.module.css'

interface MetricCheckInProps {
  habitId: string
  color: string
  metricConfig: MetricField[]
  onComplete: (metrics: Record<string, number>, notes?: string) => void
  onCancel: () => void
}

export function MetricCheckIn({ color, metricConfig, onComplete, onCancel }: MetricCheckInProps) {
  // Campos de META chegam pré-preenchidos com target_value.
  // Campos de RESULTADO começam vazios — o usuário preenche no check-in.
  const initialValues = useMemo(() => {
    const initial: Record<string, string> = {}
    for (const field of metricConfig) {
      if (field.is_target && field.target_value != null) {
        initial[field.key] = String(field.target_value)
      }
    }
    return initial
  }, [metricConfig])

  const [values, setValues] = useState<Record<string, string>>(initialValues)
  const [notes, setNotes] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    // Campos vazios são ignorados — métricas são opcionais (non-binary-habit-modeling)
    const parsedMetrics: Record<string, number> = {}
    for (const field of metricConfig) {
      const val = parseFloat(values[field.key])
      if (!isNaN(val)) {
        parsedMetrics[field.key] = val
      }
    }

    onComplete(parsedMetrics, notes.trim() || undefined)
  }

  return (
    <motion.div
      className={styles.wrapper}
      initial={{ opacity: 0, height: 0 }}
      animate={{ opacity: 1, height: 'auto' }}
      exit={{ opacity: 0, height: 0 }}
      transition={{ duration: 0.3 }}
      onClick={(e) => e.stopPropagation()}
    >
      <form onSubmit={handleSubmit} className={styles.form}>
        <div className={styles.metricsGrid}>
          {metricConfig.map(field => (
            <div key={field.key} className={`${styles.inputGroup} ${field.is_target ? styles.inputGroupTarget : ''}`}>
              <label>
                {field.label}
                <span className={styles.unitTag}>{field.unit}</span>
                {field.is_target
                  ? <span className={styles.badge} style={{ background: color + '22', color }}>meta</span>
                  : <span className={styles.badgeOptional}>opcional</span>
                }
              </label>
              <input
                type="number"
                step="any"
                value={values[field.key] || ''}
                onChange={e => setValues(prev => ({ ...prev, [field.key]: e.target.value }))}
                className={styles.input}
                placeholder={field.is_target && field.target_value != null
                  ? String(field.target_value)
                  : `0 ${field.unit}`
                }
              />
            </div>
          ))}
        </div>

        <div className={styles.inputGroup}>
          <label>Notas <span className={styles.badgeOptional}>opcional</span></label>
          <input
            type="text"
            value={notes}
            onChange={e => setNotes(e.target.value)}
            className={styles.input}
            placeholder="Como foi?"
          />
        </div>

        <div className={styles.actions}>
          <button type="button" className={styles.cancelBtn} onClick={onCancel}>
            Cancelar
          </button>
          <button type="submit" className={styles.submitBtn} style={{ background: color }}>
            Salvar ✓
          </button>
        </div>
      </form>
    </motion.div>
  )
}
