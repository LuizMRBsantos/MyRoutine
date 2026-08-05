import { useState } from 'react'
import { motion } from 'framer-motion'
import type { Habit, CreateHabitInput, MetricField } from '@/types/habit'
import { normalizeHabitPayload, validateHabit } from '@/lib/habitSchema'
import styles from './HabitFormModal.module.css'

const ICONS = ['⭐', '🔥', '💪', '📚', '🧘', '🏃', '💧', '🌱', '🎯', '✍️', '🙏', '💤', '🎵', '🥗', '🧠']
const COLORS = ['#0071E3', '#34C759', '#FF9F0A', '#FF3B30', '#AF52DE', '#FF2D55', '#5AC8FA', '#5856D6']

const TIME_OPTIONS = [
  { key: 'morning', label: '☀️ Manhã' },
  { key: 'afternoon', label: '🌤 Tarde' },
  { key: 'evening', label: '🌙 Noite' },
  { key: 'anytime', label: '✦ Qualquer hora' },
] as const

const CATEGORY_OPTIONS = [
  { key: 'general', label: 'Geral' },
  { key: 'health', label: 'Saúde' },
  { key: 'study', label: 'Estudos' },
] as const

const CHECK_TYPE_OPTIONS = [
  { key: 'simple', label: 'Simples (Check)' },
  { key: 'timed', label: 'Timer' },
  { key: 'deadline', label: 'Horário Limite' },
  { key: 'metric', label: 'Métricas (Km, Kg, etc)' },
] as const

const WEEKDAYS = [
  { day: 1, label: 'S' }, { day: 2, label: 'T' }, { day: 3, label: 'Q' },
  { day: 4, label: 'Q' }, { day: 5, label: 'S' }, { day: 6, label: 'S' }, { day: 7, label: 'D' },
]

const METRIC_PRESETS: { label: string; fields: MetricField[] }[] = [
  {
    label: '🏃 Corrida',
    fields: [
      { key: 'km', label: 'Distância', unit: 'km', is_target: true },
      { key: 'time_min', label: 'Tempo', unit: 'min' },
      { key: 'calories', label: 'Calorias', unit: 'kcal' },
    ],
  },
  {
    label: '🏋 Academia',
    fields: [
      { key: 'time_min', label: 'Tempo', unit: 'min' },
      { key: 'calories', label: 'Calorias', unit: 'kcal' },
    ],
  },
]

function initialForm(habit?: Habit): CreateHabitInput {
  if (habit) {
    return {
      name: habit.name,
      description: habit.description || undefined,
      icon: habit.icon,
      color: habit.color,
      frequency: habit.frequency,
      target_days: [...habit.target_days],
      time_of_day: habit.time_of_day,
      category: habit.category,
      check_type: habit.check_type,
      timer_minutes: habit.timer_minutes,
      deadline_time: habit.deadline_time,
      metric_config: habit.metric_config?.map((m) => ({ ...m })),
    }
  }
  return {
    name: '', icon: '⭐', color: '#0071E3',
    frequency: 'daily', target_days: [1, 2, 3, 4, 5, 6, 7],
    time_of_day: 'anytime', category: 'general', check_type: 'simple',
  }
}

interface HabitFormModalProps {
  habit?: Habit          // presente = modo edição
  isPending: boolean
  onSave: (input: CreateHabitInput) => void
  onClose: () => void
}

export function HabitFormModal({ habit, isPending, onSave, onClose }: HabitFormModalProps) {
  const [form, setForm] = useState<CreateHabitInput>(() => initialForm(habit))
  const [error, setError] = useState<string | null>(null)
  const isEdit = !!habit

  const toggleDay = (day: number) => {
    setForm((f) => {
      const days = f.target_days.includes(day)
        ? f.target_days.filter((d) => d !== day)
        : [...f.target_days, day].sort((a, b) => a - b)
      return { ...f, target_days: days, frequency: days.length === 7 ? 'daily' : 'custom' }
    })
  }

  const updateMetricField = (index: number, field: Partial<MetricField>) => {
    setForm((f) => {
      const config = [...(f.metric_config || [])]
      config[index] = { ...config[index], ...field }
      return { ...f, metric_config: config }
    })
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const payload = normalizeHabitPayload(form)

    const validationError = validateHabit(payload)
    if (validationError) {
      setError(validationError)
      return
    }
    setError(null)
    onSave(payload)
  }

  return (
    <>
      <motion.div
        className={styles.overlay}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
      />
      <div className={styles.modalWrapper}>
        <motion.div
          className={styles.modal}
          initial={{ opacity: 0, y: 32, scale: 0.96 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: 16, scale: 0.97 }}
          transition={{ type: 'spring', damping: 25, stiffness: 300 }}
        >
          <h2 className={styles.modalTitle}>{isEdit ? 'Editar Hábito' : 'Novo Hábito'}</h2>

          <form onSubmit={handleSubmit} className={styles.createForm}>
            {/* Preview + name */}
            <div className={styles.previewRow}>
              <div
                className={styles.iconPreview}
                style={{ background: form.color + '20', color: form.color }}
              >
                {form.icon}
              </div>
              <input
                className="input"
                placeholder="Nome do hábito..."
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                required
                autoFocus
                style={{ flex: 1 }}
              />
            </div>

            {/* Description */}
            <input
              className="input"
              placeholder="Descrição (opcional)"
              value={form.description ?? ''}
              onChange={(e) => setForm((f) => ({ ...f, description: e.target.value || undefined }))}
            />

            {/* Icon picker */}
            <div>
              <p className={styles.pickerLabel}>Ícone</p>
              <div className={styles.iconPicker}>
                {ICONS.map((icon) => (
                  <button
                    key={icon}
                    type="button"
                    className={`${styles.iconOption} ${form.icon === icon ? styles.selected : ''}`}
                    onClick={() => setForm((f) => ({ ...f, icon }))}
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
                {COLORS.map((color) => (
                  <button
                    key={color}
                    type="button"
                    className={`${styles.colorOption} ${form.color === color ? styles.selected : ''}`}
                    style={{ background: color }}
                    onClick={() => setForm((f) => ({ ...f, color }))}
                  />
                ))}
              </div>
            </div>

            {/* Weekdays */}
            <div>
              <p className={styles.pickerLabel}>Dias da semana</p>
              <div className={styles.weekdayPicker}>
                {WEEKDAYS.map((w) => (
                  <button
                    key={w.day}
                    type="button"
                    className={`${styles.weekdayOption} ${form.target_days.includes(w.day) ? styles.selected : ''}`}
                    style={form.target_days.includes(w.day) ? { borderColor: form.color, color: form.color } : {}}
                    onClick={() => toggleDay(w.day)}
                  >
                    {w.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Time of day */}
            <div>
              <p className={styles.pickerLabel}>Período do dia</p>
              <div className={styles.timeGrid}>
                {TIME_OPTIONS.map((t) => (
                  <button
                    key={t.key}
                    type="button"
                    className={`${styles.timeOption} ${form.time_of_day === t.key ? styles.selected : ''}`}
                    style={form.time_of_day === t.key ? { borderColor: form.color, color: form.color } : {}}
                    onClick={() => setForm((f) => ({ ...f, time_of_day: t.key }))}
                  >
                    {t.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Category (módulo consumidor: Saúde/Estudos leem por categoria) */}
            <div>
              <p className={styles.pickerLabel}>Categoria</p>
              <div className={styles.timeGrid}>
                {CATEGORY_OPTIONS.map((c) => (
                  <button
                    key={c.key}
                    type="button"
                    className={`${styles.timeOption} ${form.category === c.key ? styles.selected : ''}`}
                    style={form.category === c.key ? { borderColor: form.color, color: form.color } : {}}
                    onClick={() => setForm((f) => ({ ...f, category: c.key }))}
                  >
                    {c.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Check Type */}
            <div>
              <p className={styles.pickerLabel}>Tipo de Hábito</p>
              <div className={styles.timeGrid}>
                {CHECK_TYPE_OPTIONS.map((t) => (
                  <button
                    key={t.key}
                    type="button"
                    className={`${styles.timeOption} ${form.check_type === t.key ? styles.selected : ''}`}
                    style={form.check_type === t.key ? { borderColor: form.color, color: form.color } : {}}
                    onClick={() => setForm((f) => ({ ...f, check_type: t.key }))}
                  >
                    {t.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Conditional fields */}
            {form.check_type === 'timed' && (
              <div>
                <p className={styles.pickerLabel}>Duração do Timer (minutos)</p>
                <input
                  type="number"
                  min="1"
                  required
                  className="input"
                  placeholder="Ex: 30"
                  value={form.timer_minutes || ''}
                  onChange={(e) => setForm((f) => ({ ...f, timer_minutes: parseInt(e.target.value) || undefined }))}
                />
              </div>
            )}

            {form.check_type === 'deadline' && (
              <div>
                <p className={styles.pickerLabel}>Horário Limite</p>
                <input
                  type="time"
                  required
                  className="input"
                  value={form.deadline_time || ''}
                  onChange={(e) => setForm((f) => ({ ...f, deadline_time: e.target.value }))}
                />
              </div>
            )}

            {form.check_type === 'metric' && (
              <div>
                <p className={styles.pickerLabel}>Métricas a registrar</p>

                <div className={styles.metricPresets}>
                  {METRIC_PRESETS.map((preset) => (
                    <button
                      key={preset.label}
                      type="button"
                      className={styles.presetBtn}
                      onClick={() => setForm((f) => ({ ...f, metric_config: preset.fields.map((x) => ({ ...x })) }))}
                    >
                      {preset.label}
                    </button>
                  ))}
                </div>

                <div className={styles.metricList}>
                  {form.metric_config?.map((m, idx) => (
                    <div key={idx} className={styles.metricBlock}>
                      <div className={styles.metricRow}>
                        <input
                          className={`input ${styles.metricLabel}`}
                          placeholder="Nome (ex: Distância)"
                          value={m.label}
                          onChange={(e) => {
                            const label = e.target.value
                            const key = label.toLowerCase().replace(/\s+/g, '_').replace(/[^a-z0-9_]/g, '')
                            updateMetricField(idx, { label, key })
                          }}
                        />
                        <input
                          className={`input ${styles.metricUnit}`}
                          placeholder="Unidade"
                          value={m.unit}
                          onChange={(e) => updateMetricField(idx, { unit: e.target.value })}
                        />
                        <button
                          type="button"
                          onClick={() => setForm((f) => {
                            const config = [...(f.metric_config || [])]
                            config.splice(idx, 1)
                            return { ...f, metric_config: config }
                          })}
                          className={styles.metricRemove}
                        >×</button>
                      </div>

                      <div className={styles.metricTargetRow}>
                        <label className={styles.metricTargetToggle}>
                          <input
                            type="checkbox"
                            checked={!!m.is_target}
                            onChange={(e) => updateMetricField(idx, {
                              is_target: e.target.checked,
                              target_value: e.target.checked ? m.target_value : undefined,
                            })}
                          />
                          <span>Este é um campo de meta (pré-preenchido no check-in)</span>
                        </label>

                        {m.is_target && (
                          <input
                            type="number"
                            step="any"
                            className={`input ${styles.metricTargetValue}`}
                            placeholder={`Meta padrão em ${m.unit || 'unidade'}`}
                            value={m.target_value ?? ''}
                            onChange={(e) => updateMetricField(idx, {
                              target_value: e.target.value === '' ? undefined : parseFloat(e.target.value),
                            })}
                          />
                        )}
                      </div>
                    </div>
                  ))}
                  <button
                    type="button"
                    onClick={() => setForm((f) => ({ ...f, metric_config: [...(f.metric_config || []), { key: '', label: '', unit: '' }] }))}
                    className={styles.metricAdd}
                  >
                    + Adicionar campo
                  </button>
                </div>
              </div>
            )}

            {error && <p className={styles.formError}>{error}</p>}

            <div className={styles.modalActions}>
              <button type="button" className="btn btn-ghost" onClick={onClose}>
                Cancelar
              </button>
              <button type="submit" className="btn btn-primary" disabled={isPending}>
                {isPending ? 'Salvando...' : isEdit ? 'Salvar Alterações' : 'Criar Hábito'}
              </button>
            </div>
          </form>
        </motion.div>
      </div>
    </>
  )
}
