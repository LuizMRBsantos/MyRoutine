import { useMemo } from 'react'
import { motion } from 'framer-motion'
import type { Task } from '@/types/task'
import { CATEGORY_META } from '@/types/task'
import styles from './MonthCalendarView.module.css'

interface MonthCalendarViewProps {
  viewDate: Date
  tasks: Task[]
  onSelectDay: (dateStr: string) => void
  onSelectTask: (task: Task) => void
}

const WEEKDAYS = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb']

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`
}

export function MonthCalendarView({
  viewDate,
  tasks,
  onSelectDay,
  onSelectTask,
}: MonthCalendarViewProps) {
  const year = viewDate.getFullYear()
  const month = viewDate.getMonth()

  const today = new Date()
  const todayStr = `${today.getFullYear()}-${pad(today.getMonth() + 1)}-${pad(today.getDate())}`

  // Agrupar tarefas por data (YYYY-MM-DD)
  const tasksByDate = useMemo(() => {
    const map = new Map<string, Task[]>()
    for (const t of tasks) {
      const list = map.get(t.date) ?? []
      list.push(t)
      map.set(t.date, list)
    }
    // Ordenar tarefas por horário dentro de cada dia
    map.forEach((list) => {
      list.sort((a, b) => {
        const timeA = a.start_time ?? '99:99'
        const timeB = b.start_time ?? '99:99'
        return timeA.localeCompare(timeB)
      })
    })
    return map
  }, [tasks])

  const daysInMonth = useMemo(() => {
    const firstDayOfWeek = new Date(year, month, 1).getDay() // 0 = Dom
    const totalDays = new Date(year, month + 1, 0).getDate()

    const cells: { dateStr?: string; dayNum?: number; isToday?: boolean }[] = []

    // Dias vazios antes do dia 1
    for (let i = 0; i < firstDayOfWeek; i++) {
      cells.push({})
    }

    // Dias do mês
    for (let d = 1; d <= totalDays; d++) {
      const dateStr = `${year}-${pad(month + 1)}-${pad(d)}`
      cells.push({
        dateStr,
        dayNum: d,
        isToday: dateStr === todayStr,
      })
    }

    return cells
  }, [year, month, todayStr])

  return (
    <motion.div
      className={styles.container}
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -12 }}
      transition={{ duration: 0.3 }}
    >
      <div className={styles.header}>
        <div>
          <h2 className={styles.title}>
            <span>🗓️</span>
            <span>Visão Macro do Mês</span>
          </h2>
          <p className={styles.subtitle}>
            Acompanhe provas, compromissos e tarefas ao longo de todo o mês
          </p>
        </div>
      </div>

      <div className={styles.weekdays}>
        {WEEKDAYS.map((w) => (
          <div key={w}>{w}</div>
        ))}
      </div>

      <div className={styles.grid}>
        {daysInMonth.map((cell, index) => {
          if (!cell.dayNum || !cell.dateStr) {
            return <div key={`empty-${index}`} className={`${styles.dayCell} ${styles.dayCellEmpty}`} />
          }

          const dayTasks = tasksByDate.get(cell.dateStr) ?? []
          const visibleTasks = dayTasks.slice(0, 3)
          const hiddenCount = dayTasks.length - visibleTasks.length

          return (
            <div
              key={cell.dateStr}
              className={`${styles.dayCell} ${cell.isToday ? styles.dayCellToday : ''}`}
              onClick={() => onSelectDay(cell.dateStr!)}
            >
              <div className={styles.dayHeader}>
                <span
                  className={`${styles.dayNumber} ${
                    cell.isToday ? styles.dayNumberToday : ''
                  }`}
                >
                  {cell.dayNum}
                </span>
                <button
                  type="button"
                  className={styles.addTaskIcon}
                  title="Adicionar tarefa neste dia"
                  onClick={(e) => {
                    e.stopPropagation()
                    onSelectDay(cell.dateStr!)
                  }}
                >
                  +
                </button>
              </div>

              <div className={styles.tasksContainer}>
                {visibleTasks.map((t) => {
                  const meta = CATEGORY_META[t.category] ?? CATEGORY_META.other
                  const timeShort = t.start_time ? t.start_time.slice(0, 5) : ''
                  return (
                    <div
                      key={t.id}
                      className={styles.taskPill}
                      style={{
                        backgroundColor: meta.color + '22',
                        color: meta.color,
                        borderColor: meta.color + '44',
                      }}
                      title={`${meta.label}: ${t.title}${t.start_time ? ` (${t.start_time})` : ''}`}
                      onClick={(e) => {
                        e.stopPropagation()
                        onSelectTask(t)
                      }}
                    >
                      <span>{meta.icon}</span>
                      {timeShort && <span className={styles.taskTime}>{timeShort}</span>}
                      <span className={styles.taskTitle}>{t.title}</span>
                    </div>
                  )
                })}
                {hiddenCount > 0 && (
                  <div className={styles.moreCount}>
                    +{hiddenCount} {hiddenCount === 1 ? 'tarefa' : 'tarefas'}
                  </div>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </motion.div>
  )
}
