import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import type { CreateTaskInput, Task, TaskCategory, TaskPriority } from '@/types/task'
import { CATEGORY_META, notifyByDefault } from '@/types/task'
import { DatePickerCalendar } from './DatePickerCalendar'
import styles from './AddTaskModal.module.css'

interface AddTaskModalProps {
  date: string
  editTask?: Task            // se passado, modo edição
  defaultStartTime?: string  // se passado, horário pré-preenchido (ex: clique no grid do calendar)
  onSave: (input: CreateTaskInput) => void
  onClose: () => void
  isPending?: boolean
}

const CATEGORIES = Object.entries(CATEGORY_META) as [TaskCategory, typeof CATEGORY_META[TaskCategory]][]
const PRIORITIES: { key: TaskPriority; label: string }[] = [
  { key: 'high',   label: '🔴 Alta' },
  { key: 'medium', label: '🟡 Média' },
  { key: 'low',    label: '🟢 Baixa' },
]

function formatHumanDate(dateStr: string): { label: string; badge?: string } {
  const parts = dateStr.split('-').map(Number)
  if (parts.length !== 3 || isNaN(parts[0]) || isNaN(parts[1]) || isNaN(parts[2])) {
    return { label: dateStr }
  }
  const [y, m, d] = parts
  const dateObj = new Date(y, m - 1, d)
  const label = dateObj.toLocaleDateString('pt-BR', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })

  const now = new Date()
  const pad = (n: number) => (n < 10 ? `0${n}` : `${n}`)
  const todayStr = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`

  const tomorrowObj = new Date(now)
  tomorrowObj.setDate(now.getDate() + 1)
  const tomorrowStr = `${tomorrowObj.getFullYear()}-${pad(tomorrowObj.getMonth() + 1)}-${pad(tomorrowObj.getDate())}`

  const yesterdayObj = new Date(now)
  yesterdayObj.setDate(now.getDate() - 1)
  const yesterdayStr = `${yesterdayObj.getFullYear()}-${pad(yesterdayObj.getMonth() + 1)}-${pad(yesterdayObj.getDate())}`

  if (dateStr === todayStr) return { label, badge: 'Hoje' }
  if (dateStr === tomorrowStr) return { label, badge: 'Amanhã' }
  if (dateStr === yesterdayStr) return { label, badge: 'Ontem' }
  return { label }
}

export function AddTaskModal({ date, editTask, defaultStartTime, onSave, onClose, isPending }: AddTaskModalProps) {
  const isEdit = !!editTask
  const details = (editTask?.task_details ?? {}) as Record<string, unknown>

  // Estado pré-preenchido quando editando
  const [title, setTitle] = useState(editTask?.title ?? '')
  const [taskDate, setTaskDate] = useState(editTask?.date ?? date)
  const [showCalendar, setShowCalendar] = useState(false)
  const [category, setCategory] = useState<TaskCategory>(editTask?.category ?? 'other')
  const [startTime, setStartTime] = useState(editTask?.start_time?.slice(0, 5) ?? defaultStartTime ?? '')
  const [durationMin, setDurationMin] = useState(editTask?.duration_minutes?.toString() ?? '')
  const [priority, setPriority] = useState<TaskPriority>(editTask?.priority ?? 'medium')
  const [notes, setNotes] = useState(editTask?.notes ?? '')
  // null = segue o padrão da categoria até a pessoa mexer no "Me avisar antes".
  const [notifyChoice, setNotifyChoice] = useState<boolean | null>(editTask ? editTask.notify : null)
  const notify = notifyChoice ?? notifyByDefault(category)

  // Campos específicos por categoria — pré-preenchidos do task_details
  const [subject, setSubject] = useState((details.subject as string) ?? '')
  const [project, setProject] = useState((details.project as string) ?? '')
  const [workoutType, setWorkoutType] = useState((details.workout_type as string) ?? '')

  const handleTimeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let val = e.target.value.replace(/[^\d:]/g, '')
    if (val.length === 2 && startTime.length === 1 && !val.includes(':')) {
      val = val + ':'
    }
    if (val.length > 5) val = val.slice(0, 5)
    setStartTime(val)
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    let task_details: Record<string, unknown> | undefined

    if (category === 'study') {
      task_details = { subject: subject || undefined }
    } else if (category === 'exam') {
      task_details = { subject: subject || undefined }
    } else if (category === 'work') {
      task_details = { project: project || undefined }
    } else if (category === 'exercise') {
      task_details = { workout_type: workoutType || undefined }
    }

    onSave({
      title,
      date: taskDate,
      category,
      priority,
      start_time: startTime || undefined,
      duration_minutes: durationMin ? parseInt(durationMin) : undefined,
      notes: notes || undefined,
      task_details,
      notify,
    })
  }

  const meta = CATEGORY_META[category]
  const dateInfo = formatHumanDate(taskDate)

  return (
    <AnimatePresence>
      <>
        <motion.div
          className={styles.overlay}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          onClick={onClose}
        />
        <div className={styles.wrapper}>
          <motion.div
            className={styles.modal}
            initial={{ opacity: 0, y: 24, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 12, scale: 0.98 }}
            transition={{ type: 'spring', damping: 28, stiffness: 320 }}
          >
            {/* Header */}
            <div className={styles.header}>
              <div
                className={styles.headerIcon}
                style={{ background: meta.color + '20', color: meta.color }}
              >
                {meta.icon}
              </div>
              <div>
                <h2 className={styles.title}>
                  {isEdit ? 'Editar Tarefa' : 'Nova Tarefa'}
                </h2>
                <p className={styles.dateLabel}>{dateInfo.label}</p>
              </div>
              <button className={styles.closeBtn} onClick={onClose} type="button">
                ×
              </button>
            </div>

            <form onSubmit={handleSubmit} className={styles.form}>
              {/* Título */}
              <div>
                <p className={styles.label}>Tarefa</p>
                <input
                  className="input"
                  style={{ fontSize: '1.05rem', fontWeight: 500, padding: '12px 16px' }}
                  placeholder="O que você vai fazer?"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  required
                  autoFocus
                />
              </div>

              {/* Data — Botão com Calendário Bonito */}
              <div>
                <p className={styles.label}>Data</p>
                <button
                  type="button"
                  className={styles.dateSelectorBtn}
                  onClick={() => setShowCalendar((prev) => !prev)}
                >
                  <div className={styles.dateLeft}>
                    <span className={styles.dateIcon}>📅</span>
                    <span className={styles.dateText}>{dateInfo.label}</span>
                    {dateInfo.badge && (
                      <span className={styles.dateBadge}>{dateInfo.badge}</span>
                    )}
                  </div>
                  <span className={styles.dateChevron}>
                    {showCalendar ? '▲' : '▼'}
                  </span>
                </button>

                <AnimatePresence>
                  {showCalendar && (
                    <motion.div
                      initial={{ opacity: 0, height: 0 }}
                      animate={{ opacity: 1, height: 'auto' }}
                      exit={{ opacity: 0, height: 0 }}
                      transition={{ duration: 0.2 }}
                      style={{ overflow: 'hidden' }}
                    >
                      <DatePickerCalendar
                        selectedDate={taskDate}
                        onSelect={(newDate) => {
                          setTaskDate(newDate)
                          setShowCalendar(false)
                        }}
                      />
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>

              {/* Categoria */}
              <div>
                <p className={styles.label}>Categoria</p>
                <div className={styles.categoryGrid}>
                  {CATEGORIES.map(([key, m]) => (
                    <button
                      key={key}
                      type="button"
                      className={`${styles.catBtn} ${
                        category === key ? styles.catBtnActive : ''
                      }`}
                      style={
                        category === key
                          ? {
                              borderColor: m.color,
                              color: m.color,
                              background: m.color + '18',
                            }
                          : {}
                      }
                      onClick={() => setCategory(key)}
                    >
                      <span>{m.icon}</span>
                      <span>{m.label}</span>
                    </button>
                  ))}
                </div>
              </div>

              {/* Campos específicos por categoria */}
              {(category === 'study' || category === 'exam') && (
                <div>
                  <p className={styles.label}>Matéria</p>
                  <input
                    className="input"
                    placeholder="Ex: OAC, BD, PLP..."
                    value={subject}
                    onChange={(e) => setSubject(e.target.value)}
                  />
                </div>
              )}

              {category === 'work' && (
                <div>
                  <p className={styles.label}>Projeto</p>
                  <input
                    className="input"
                    placeholder="Ex: MyRoutine, Nexx..."
                    value={project}
                    onChange={(e) => setProject(e.target.value)}
                  />
                </div>
              )}

              {category === 'exercise' && (
                <div>
                  <p className={styles.label}>Tipo de treino</p>
                  <input
                    className="input"
                    placeholder="Ex: Corrida, Musculação..."
                    value={workoutType}
                    onChange={(e) => setWorkoutType(e.target.value)}
                  />
                </div>
              )}

              {/* Horário e Duração */}
              <div className={styles.timeRow}>
                <div>
                  <p className={styles.label}>Horário</p>
                  <input
                    type="text"
                    className="input"
                    placeholder="Ex: 05:30 ou 16:00"
                    maxLength={5}
                    value={startTime}
                    onChange={handleTimeChange}
                  />
                </div>
                <div>
                  <p className={styles.label}>Duração (min)</p>
                  <input
                    type="number"
                    min="1"
                    className="input"
                    placeholder="Ex: 90"
                    value={durationMin}
                    onChange={(e) => setDurationMin(e.target.value)}
                  />
                </div>
              </div>

              {/* Lembrete: só faz sentido com horário */}
              {startTime && (
                <label className={styles.notifyRow}>
                  <input
                    type="checkbox"
                    checked={notify}
                    onChange={(e) => setNotifyChoice(e.target.checked)}
                  />
                  Me avisar antes
                </label>
              )}

              {/* Prioridade */}
              <div>
                <p className={styles.label}>Prioridade</p>
                <div className={styles.priorityRow}>
                  {PRIORITIES.map((p) => (
                    <button
                      key={p.key}
                      type="button"
                      className={`${styles.priorityBtn} ${
                        priority === p.key ? styles.priorityBtnActive : ''
                      }`}
                      onClick={() => setPriority(p.key)}
                    >
                      {p.label}
                    </button>
                  ))}
                </div>
              </div>

              {/* Notas */}
              <div>
                <p className={styles.label}>
                  Notas <span className={styles.optional}>opcional</span>
                </p>
                <textarea
                  className={`input ${styles.textarea}`}
                  placeholder="Detalhes do planejamento..."
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  rows={2}
                />
              </div>

              {/* Ações */}
              <div className={styles.actions}>
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={onClose}
                >
                  Cancelar
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={isPending}
                  style={{ background: meta.color, borderColor: meta.color }}
                >
                  {isPending
                    ? 'Salvando...'
                    : isEdit
                    ? `Salvar ${meta.icon}`
                    : `Adicionar ${meta.icon}`}
                </button>
              </div>
            </form>
          </motion.div>
        </div>
      </>
    </AnimatePresence>
  )
}
