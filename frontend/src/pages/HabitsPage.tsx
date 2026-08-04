import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useHabits, useHabitStats, useCheckIn, useUndoCheckIn, useCreateHabit, useDeleteHabit } from '@/hooks/useHabits'
import { HabitCard } from '@/components/habits/HabitCard'
import { HabitDrawer } from '@/components/habits/HabitDrawer'
import { WeeklyReview } from '@/components/habits/WeeklyReview'
import type { Habit, CreateHabitInput, CheckInInput } from '@/types/habit'
import styles from './HabitsPage.module.css'

const ICONS = ['⭐', '🔥', '💪', '📚', '🧘', '🏃', '💧', '🌱', '🎯', '✍️', '🙏', '💤', '🎵', '🥗', '🧠']
const COLORS = ['#0071E3', '#34C759', '#FF9F0A', '#FF3B30', '#AF52DE', '#FF2D55', '#5AC8FA', '#5856D6']

const TIME_GROUPS = [
  { key: 'morning', label: '☀️ Manhã' },
  { key: 'afternoon', label: '🌤 Tarde' },
  { key: 'evening', label: '🌙 Noite' },
  { key: 'anytime', label: '✦ Qualquer hora' },
] as const

const METRIC_PRESETS = [
  {
    label: '🏃 Corrida',
    fields: [
      // is_target=true: distância é a META do dia — pré-preenchida no check-in
      { key: 'km',       label: 'Distância',  unit: 'km',   is_target: true,  target_value: undefined },
      { key: 'time_min', label: 'Tempo',      unit: 'min',  is_target: false },
      { key: 'calories', label: 'Calorias',   unit: 'kcal', is_target: false },
    ],
  },
  {
    label: '🏋 Academia',
    fields: [
      { key: 'time_min', label: 'Tempo',      unit: 'min',  is_target: false },
      { key: 'calories', label: 'Calorias',   unit: 'kcal', is_target: false },
    ],
  },
]

export function HabitsPage() {
  const { data: habits = [], isLoading } = useHabits()
  const { data: stats } = useHabitStats()
  const checkIn = useCheckIn()
  const undoCheckIn = useUndoCheckIn()
  const createHabit = useCreateHabit()
  const deleteHabit = useDeleteHabit()

  const [showCreate, setShowCreate] = useState(false)
  const [selectedHabit, setSelectedHabit] = useState<Habit | null>(null)
  const [newHabit, setNewHabit] = useState<CreateHabitInput>({
    name: '', icon: '⭐', color: '#0071E3',
    frequency: 'daily', target_days: [1, 2, 3, 4, 5, 6, 7],
    time_of_day: 'anytime',
    check_type: 'simple',
  })

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

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newHabit.name.trim()) return

    const payload: CreateHabitInput = { ...newHabit }
    if (payload.check_type !== 'timed') delete payload.timer_minutes
    if (payload.check_type !== 'deadline' || !payload.deadline_time) delete payload.deadline_time
    if (payload.check_type !== 'metric' || !payload.metric_config?.length) delete payload.metric_config

    await createHabit.mutateAsync(payload)
    setShowCreate(false)
    setNewHabit({ name: '', icon: '⭐', color: '#0071E3', frequency: 'daily', target_days: [1,2,3,4,5,6,7], time_of_day: 'anytime', check_type: 'simple' })
  }

  const addMetricField = () => {
    setNewHabit(h => ({
      ...h,
      metric_config: [...(h.metric_config || []), { key: '', label: '', unit: '' }]
    }))
  }

  const updateMetricField = (index: number, field: Partial<import('@/types/habit').MetricField>) => {
    setNewHabit(h => {
      const config = [...(h.metric_config || [])]
      config[index] = { ...config[index], ...field }
      return { ...h, metric_config: config }
    })
  }

  const removeMetricField = (index: number) => {
    setNewHabit(h => {
      const config = [...(h.metric_config || [])]
      config.splice(index, 1)
      return { ...h, metric_config: config }
    })
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
                        onDelete={() => deleteHabit.mutate(habit.id)}
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
            <div className={styles.modalWrapper}>
              <motion.div
                className={styles.modal}
                initial={{ opacity: 0, y: 32, scale: 0.96 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: 16, scale: 0.97 }}
                transition={{ type: 'spring', damping: 25, stiffness: 300 }}
              >
                <h2 className={styles.modalTitle}>Novo Hábito</h2>

                <form onSubmit={handleCreate} className={styles.createForm}>
                  {/* Preview + name */}
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
                          className={`${styles.iconOption} ${newHabit.icon === icon ? styles.selected : ''}`}
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
                          className={`${styles.colorOption} ${newHabit.color === color ? styles.selected : ''}`}
                          style={{ background: color }}
                          onClick={() => setNewHabit(h => ({ ...h, color }))}
                        />
                      ))}
                    </div>
                  </div>

                  {/* Time of day */}
                  <div>
                    <p className={styles.pickerLabel}>Período do dia</p>
                    <div className={styles.timeGrid}>
                      {TIME_GROUPS.map(t => (
                        <button
                          key={t.key}
                          type="button"
                          className={`${styles.timeOption} ${newHabit.time_of_day === t.key ? styles.selected : ''}`}
                          style={newHabit.time_of_day === t.key ? { borderColor: newHabit.color, color: newHabit.color } : {}}
                          onClick={() => setNewHabit(h => ({ ...h, time_of_day: t.key }))}
                        >
                          {t.label}
                        </button>
                      ))}
                    </div>
                  </div>

                  {/* Check Type */}
                  <div>
                    <p className={styles.pickerLabel}>Tipo de Hábito</p>
                    <div className={styles.timeGrid}>
                      {[
                        { key: 'simple', label: 'Simples (Check)' },
                        { key: 'timed', label: 'Timer' },
                        { key: 'deadline', label: 'Horário Limite' },
                        { key: 'metric', label: 'Métricas (Km, Kg, etc)' }
                      ].map(t => (
                        <button
                          key={t.key}
                          type="button"
                          className={`${styles.timeOption} ${newHabit.check_type === t.key ? styles.selected : ''}`}
                          style={newHabit.check_type === t.key ? { borderColor: newHabit.color, color: newHabit.color } : {}}
                          onClick={() => setNewHabit(h => ({ ...h, check_type: t.key as any }))}
                        >
                          {t.label}
                        </button>
                      ))}
                    </div>
                  </div>

                  {/* Conditional Fields based on check_type */}
                  {newHabit.check_type === 'timed' && (
                    <div>
                      <p className={styles.pickerLabel}>Duração do Timer (minutos)</p>
                      <input
                        type="number"
                        min="1"
                        required
                        className="input"
                        placeholder="Ex: 30"
                        value={newHabit.timer_minutes || ''}
                        onChange={e => setNewHabit(h => ({ ...h, timer_minutes: parseInt(e.target.value) || undefined }))}
                      />
                    </div>
                  )}

                  {newHabit.check_type === 'deadline' && (
                    <div>
                      <p className={styles.pickerLabel}>Horário Limite</p>
                      <input
                        type="time"
                        required
                        className="input"
                        value={newHabit.deadline_time || ''}
                        onChange={e => setNewHabit(h => ({ ...h, deadline_time: e.target.value }))}
                      />
                    </div>
                  )}

                  {newHabit.check_type === 'metric' && (
                    <div>
                      <p className={styles.pickerLabel}>Métricas a registrar</p>

                      {/* ── Presets ── */}
                      <div className={styles.metricPresets}>
                        {METRIC_PRESETS.map(preset => (
                          <button
                            key={preset.label}
                            type="button"
                            className={styles.presetBtn}
                            onClick={() => setNewHabit(h => ({ ...h, metric_config: preset.fields.map(f => ({ ...f })) }))}
                          >
                            {preset.label}
                          </button>
                        ))}
                      </div>

                      <div className={styles.metricList}>
                        {newHabit.metric_config?.map((m, idx) => (
                          <div key={idx} className={styles.metricBlock}>
                            {/* Linha principal: nome + unidade + remover */}
                            <div className={styles.metricRow}>
                              <input
                                className={`input ${styles.metricLabel}`}
                                placeholder="Nome (ex: Distância)"
                                value={m.label}
                                onChange={e => {
                                  const label = e.target.value
                                  const key = label.toLowerCase().replace(/\s+/g, '_').replace(/[^a-z0-9_]/g, '')
                                  updateMetricField(idx, { label, key })
                                }}
                              />
                              <input
                                className={`input ${styles.metricUnit}`}
                                placeholder="Unidade"
                                value={m.unit}
                                onChange={e => updateMetricField(idx, { unit: e.target.value })}
                              />
                              <button
                                type="button"
                                onClick={() => removeMetricField(idx)}
                                className={styles.metricRemove}
                              >×</button>
                            </div>

                            {/* Toggle de meta — aparece embaixo de cada campo */}
                            <div className={styles.metricTargetRow}>
                              <label className={styles.metricTargetToggle}>
                                <input
                                  type="checkbox"
                                  checked={!!m.is_target}
                                  onChange={e => updateMetricField(idx, {
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
                                  onChange={e => updateMetricField(idx, {
                                    target_value: e.target.value === '' ? undefined : parseFloat(e.target.value)
                                  })}
                                />
                              )}
                            </div>
                          </div>
                        ))}
                        <button type="button" onClick={addMetricField} className={styles.metricAdd}>
                          + Adicionar campo
                        </button>
                      </div>
                    </div>
                  )}


                  <div className={styles.modalActions}>
                    <button type="button" className="btn btn-ghost" onClick={() => setShowCreate(false)}>
                      Cancelar
                    </button>
                    <button type="submit" className="btn btn-primary" disabled={createHabit.isPending}>
                      {createHabit.isPending ? 'Criando...' : 'Criar Hábito'}
                    </button>
                  </div>
                </form>
              </motion.div>
            </div>
          </>
        )}
      </AnimatePresence>

      {/* ─── Habit Drawer ─────────────────────────── */}
      <HabitDrawer habit={selectedHabit} onClose={() => setSelectedHabit(null)} />
    </div>
  )
}
