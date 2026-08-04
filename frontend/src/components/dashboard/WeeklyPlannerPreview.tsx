import { useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useNavigate } from 'react-router-dom'
import { useWeekTasks, useAdvanceTaskStatus, useCreateTask } from '@/hooks/useTasks'
import { AddTaskModal } from '@/components/planner/AddTaskModal'
import type { Task, TaskStatus } from '@/types/task'
import { CATEGORY_META } from '@/types/task'
import styles from './WeeklyPlannerPreview.module.css'

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`
}

function toLocalStr(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

const STATUS_SHORT: Record<TaskStatus, { label: string; color: string; bg: string }> = {
  planned:     { label: 'Plan', color: '#0071E3', bg: '#0071E318' },
  in_progress: { label: 'Em andamento', color: '#FF9F0A', bg: '#FF9F0A18' },
  done:        { label: 'Terminado', color: '#34C759', bg: '#34C75918' },
  reviewed:    { label: 'Revisado', color: '#AF52DE', bg: '#AF52DE18' },
}

export function WeeklyPlannerPreview() {
  const navigate = useNavigate()
  const advanceTask = useAdvanceTaskStatus()
  const createTask = useCreateTask()

  const [selectedAddDate, setSelectedAddDate] = useState<string | null>(null)

  const today = useMemo(() => new Date(), [])
  const todayStr = useMemo(() => toLocalStr(today), [today])

  // Limites da semana (Dom - Sáb)
  const weekDays = useMemo(() => {
    const dayOfWeek = today.getDay() // 0=Dom
    const startObj = new Date(today)
    startObj.setDate(today.getDate() - dayOfWeek)

    const days = []
    for (let i = 0; i < 7; i++) {
      const d = new Date(startObj)
      d.setDate(startObj.getDate() + i)
      const dStr = toLocalStr(d)
      days.push({
        dateStr: dStr,
        dayName: d.toLocaleDateString('pt-BR', { weekday: 'short' }),
        dayNum: d.getDate(),
        isToday: dStr === todayStr,
      })
    }
    return days
  }, [today, todayStr])

  const startStr = weekDays[0].dateStr
  const endStr = weekDays[6].dateStr

  const { data: weekTasks = [], isLoading } = useWeekTasks(startStr, endStr)

  // Agrupar tarefas por dia
  const tasksByDate = useMemo(() => {
    const map = new Map<string, Task[]>()
    for (const t of weekTasks) {
      const list = map.get(t.date) ?? []
      list.push(t)
      map.set(t.date, list)
    }
    map.forEach((list) => {
      list.sort((a, b) => (a.start_time ?? '99:99').localeCompare(b.start_time ?? '99:99'))
    })
    return map
  }, [weekTasks])

  return (
    <motion.div
      className={styles.card}
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, delay: 0.1 }}
    >
      <div className={styles.header}>
        <div>
          <h2 className={styles.title}>
            <span>🗓️</span>
            <span>Planejamento da Semana</span>
          </h2>
          <p className={styles.subtitle}>
            Acompanhe compromissos, treinos e estudos dos seus dias da semana
          </p>
        </div>
        <button
          type="button"
          className={styles.linkBtn}
          onClick={() => navigate('/planner')}
        >
          Ver no Planner →
        </button>
      </div>

      <div className={styles.weekGrid}>
        {weekDays.map((day) => {
          const dayTasks = tasksByDate.get(day.dateStr) ?? []

          return (
            <div
              key={day.dateStr}
              className={`${styles.dayColumn} ${
                day.isToday ? styles.dayColumnToday : ''
              }`}
            >
              <div className={styles.dayHeader}>
                <span className={styles.dayName}>{day.dayName}</span>
                <span
                  className={`${styles.dayNumber} ${
                    day.isToday ? styles.dayNumberToday : ''
                  }`}
                >
                  {day.dayNum}
                </span>
              </div>

              <div className={styles.taskList}>
                {isLoading ? (
                  <div className={styles.emptyMsg}>Carregando...</div>
                ) : dayTasks.length === 0 ? (
                  <div className={styles.emptyMsg}>Sem tarefas</div>
                ) : (
                  dayTasks.map((t) => {
                    const meta = CATEGORY_META[t.category] ?? CATEGORY_META.other
                    const statusMeta = STATUS_SHORT[t.status] ?? STATUS_SHORT.planned
                    const timeShort = t.start_time ? t.start_time.slice(0, 5) : ''

                    return (
                      <div
                        key={t.id}
                        className={styles.taskItem}
                        style={{
                          backgroundColor: meta.color + '18',
                          color: meta.color,
                          borderColor: meta.color + '44',
                        }}
                        title={`Clique para avançar status: ${t.title} (${statusMeta.label})`}
                        onClick={() => advanceTask.mutate(t.id)}
                      >
                        <div className={styles.taskInfo}>
                          <span>{meta.icon}</span>
                          {timeShort && (
                            <span className={styles.taskTime}>{timeShort}</span>
                          )}
                          <span className={styles.taskTitle}>{t.title}</span>
                        </div>
                        <span
                          className={styles.statusIndicator}
                          style={{
                            background: statusMeta.bg,
                            color: statusMeta.color,
                          }}
                        >
                          {statusMeta.label}
                        </span>
                      </div>
                    )
                  })
                )}
              </div>

              <button
                type="button"
                className={styles.addBtn}
                onClick={() => setSelectedAddDate(day.dateStr)}
              >
                + Tarefa
              </button>
            </div>
          )
        })}
      </div>

      {/* Modal para adicionar tarefa direto do Dashboard */}
      <AnimatePresence>
        {selectedAddDate && (
          <AddTaskModal
            date={selectedAddDate}
            onSave={(input) => {
              createTask.mutate(input, {
                onSuccess: () => setSelectedAddDate(null),
              })
            }}
            onClose={() => setSelectedAddDate(null)}
            isPending={createTask.isPending}
          />
        )}
      </AnimatePresence>
    </motion.div>
  )
}
