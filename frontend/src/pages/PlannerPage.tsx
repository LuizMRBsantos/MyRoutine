import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  useTasks, useWeekTasks, useMonthTasks,
  useCreateTask, useUpdateTask, useAdvanceTaskStatus, useDeleteTask,
} from '@/hooks/useTasks'
import { AddTaskModal } from '@/components/planner/AddTaskModal'
import { TaskBlock } from '@/components/planner/TaskBlock'
import { MonthCalendarView } from '@/components/planner/MonthCalendarView'
import { MonthlyGoalsPanel } from '@/components/planner/MonthlyGoalsPanel'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import type { Task } from '@/types/task'
import { CATEGORY_META } from '@/types/task'
import styles from './PlannerPage.module.css'

// ─── Date helpers ─────────────────────────────────────────────────────────────
function pad(n: number) {
  return n < 10 ? `0${n}` : `${n}`
}
function toDateStr(d: Date) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function getWeekBounds(d: Date) {
  const day = d.getDay() // 0=dom
  const start = new Date(d)
  start.setDate(d.getDate() - day)
  const end = new Date(start)
  end.setDate(start.getDate() + 6)
  return { start: toDateStr(start), end: toDateStr(end) }
}

function addDays(d: Date, n: number) {
  const r = new Date(d); r.setDate(r.getDate() + n); return r
}

const TABS = ['Hoje', 'Semana', 'Mês'] as const
type Tab = typeof TABS[number]

// ─── Page ─────────────────────────────────────────────────────────────────────
export function PlannerPage() {
  const [tab, setTab] = useState<Tab>('Hoje')
  const [viewDate, setViewDate] = useState(new Date())
  const [showAdd, setShowAdd] = useState(false)
  const [editingTask, setEditingTask] = useState<Task | null>(null)
  const [deletingTask, setDeletingTask] = useState<Task | null>(null)

  const today = new Date()
  const dateStr = toDateStr(viewDate)
  const { start: weekStart, end: weekEnd } = getWeekBounds(viewDate)

  // Queries
  const { data: dayTasks = [], isLoading: loadingDay } = useTasks(dateStr)
  const { data: weekTasks = [], isLoading: loadingWeek } = useWeekTasks(weekStart, weekEnd)
  const { data: monthTasks = [] } = useMonthTasks(viewDate)

  // Mutations
  const createTask = useCreateTask()
  const updateTask = useUpdateTask()
  const advanceTask = useAdvanceTaskStatus()
  const deleteTask = useDeleteTask()

  // ─── Nav helpers ──────────────────────────────────────────────────────────
  const navigate = (n: number) => {
    if (tab === 'Hoje') setViewDate(d => addDays(d, n))
    else if (tab === 'Semana') setViewDate(d => addDays(d, n * 7))
    else setViewDate(d => new Date(d.getFullYear(), d.getMonth() + n, 1))
  }

  const navLabel = () => {
    if (tab === 'Hoje') {
      return viewDate.toLocaleDateString('pt-BR', { weekday: 'long', day: 'numeric', month: 'long' })
    }
    if (tab === 'Semana') {
      const s = new Date(weekStart + 'T12:00')
      const e = new Date(weekEnd + 'T12:00')
      return `${s.toLocaleDateString('pt-BR', { day: 'numeric', month: 'short' })} — ${e.toLocaleDateString('pt-BR', { day: 'numeric', month: 'short', year: 'numeric' })}`
    }
    return viewDate.toLocaleDateString('pt-BR', { month: 'long', year: 'numeric' })
  }

  const isToday = toDateStr(viewDate) === toDateStr(today)

  // ─── Week view helpers ─────────────────────────────────────────────────────
  const weekDays = Array.from({ length: 7 }, (_, i) => {
    const d = new Date(weekStart + 'T12:00')
    d.setDate(d.getDate() + i)
    return { date: toDateStr(d), label: d.toLocaleDateString('pt-BR', { weekday: 'short', day: 'numeric' }), isToday: toDateStr(d) === toDateStr(today) }
  })

  // ─── Render ───────────────────────────────────────────────────────────────
  return (
    <div className={styles.page}>
      {/* ── Header ── */}
      <div className={styles.header}>
        <div>
          <h1 className={styles.pageTitle}>Planejamento</h1>
          <p className={styles.pageSubtitle}>Organize seu dia, semana e metas do mês</p>
        </div>

        <button className="btn btn-primary" onClick={() => setShowAdd(true)}>
          + Tarefa
        </button>
      </div>

      {/* ── Tabs ── */}
      <div className={styles.tabs}>
        {TABS.map(t => (
          <button
            key={t}
            className={`${styles.tab} ${tab === t ? styles.tabActive : ''}`}
            onClick={() => setTab(t)}
          >
            {t}
          </button>
        ))}
      </div>

      {/* ── Navigation ── */}
      <div className={styles.navRow}>
        <button className={styles.navBtn} onClick={() => navigate(-1)}>‹</button>
        <span className={styles.navLabel} style={{ textTransform: 'capitalize' }}>{navLabel()}</span>
        <button className={styles.navBtn} onClick={() => navigate(1)}>›</button>
        {!isToday && tab === 'Hoje' && (
          <button className={styles.todayBtn} onClick={() => setViewDate(today)}>Hoje</button>
        )}
      </div>

      <AnimatePresence mode="wait">

        {/* ════════════════ VISÃO DIÁRIA ════════════════ */}
        {tab === 'Hoje' && (
          <motion.div key="day" className={styles.content}
            initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }}
          >
            {loadingDay ? (
              <div className={styles.loading}>Carregando...</div>
            ) : dayTasks.length === 0 ? (
              <div className={styles.empty}>
                <p>Nenhuma tarefa planejada para este dia.</p>
                <button className="btn btn-primary" onClick={() => setShowAdd(true)}>+ Planejar agora</button>
              </div>
            ) : (
              <div className={styles.taskList}>
                <AnimatePresence>
                  {dayTasks.map(task => (
                    <TaskBlock
                      key={task.id}
                      task={task}
                      onAdvance={() => advanceTask.mutate(task.id)}
                      onEdit={() => setEditingTask(task)}
                      onDelete={() => setDeletingTask(task)}
                      isPending={advanceTask.isPending || deleteTask.isPending}
                    />
                  ))}
                </AnimatePresence>
              </div>
            )}

            {/* Summary bar */}
            {dayTasks.length > 0 && (
              <div className={styles.summaryBar}>
                {(['planned','in_progress','done','reviewed'] as const).map(s => {
                  const count = dayTasks.filter(t => t.status === s).length
                  if (!count) return null
                  const labels = { planned: 'Planejadas', in_progress: 'Fazendo', done: 'Terminadas', reviewed: 'Revisadas' }
                  return <span key={s} className={styles.summaryItem}>{count} {labels[s]}</span>
                })}
              </div>
            )}
          </motion.div>
        )}

        {/* ════════════════ VISÃO SEMANAL ════════════════ */}
        {tab === 'Semana' && (
          <motion.div key="week" className={styles.content}
            initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }}
          >
            {loadingWeek ? (
              <div className={styles.loading}>Carregando...</div>
            ) : (
              <div className={styles.weekGrid}>
                {weekDays.map(day => {
                  const tasks = weekTasks.filter(t => t.date === day.date)
                  const hasExam = tasks.some(t => t.category === 'exam')
                  return (
                    <div key={day.date} className={`${styles.weekDay} ${day.isToday ? styles.weekDayToday : ''}`}>
                      <div className={styles.weekDayHeader}>
                        <span className={styles.weekDayLabel} style={{ textTransform: 'capitalize' }}>{day.label}</span>
                        {hasExam && <span className={styles.examBadge}>📝 Prova</span>}
                      </div>
                      <div className={styles.weekTaskList}>
                        {tasks.length === 0 ? (
                          <p className={styles.weekEmpty}>—</p>
                        ) : tasks.map(task => {
                          const meta = CATEGORY_META[task.category]
                          const isDone = task.status === 'done' || task.status === 'reviewed'
                          return (
                            <div
                              key={task.id}
                              className={`${styles.weekTaskChip} ${isDone ? styles.weekTaskChipDone : ''}`}
                              style={{ borderLeftColor: task.color ?? meta.color }}
                              title={task.title}
                            >
                              <span>{meta.icon}</span>
                              <span className={styles.weekTaskTitle}>{task.start_time?.slice(0, 5)} {task.title}</span>
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </motion.div>
        )}

        {/* ════════════════ VISÃO MENSAL ════════════════ */}
        {tab === 'Mês' && (
          <motion.div key="month" className={styles.content}
            initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -8 }}
          >
            {/* Metas do mês */}
            <MonthlyGoalsPanel viewDate={viewDate} />

            {/* Calendário Macro do Mês */}
            <MonthCalendarView
              viewDate={viewDate}
              tasks={monthTasks}
              onSelectDay={(dStr) => {
                const parts = dStr.split('-').map(Number)
                if (parts.length === 3) {
                  setViewDate(new Date(parts[0], parts[1] - 1, parts[2]))
                }
                setShowAdd(true)
              }}
              onSelectTask={(task) => setEditingTask(task)}
            />
          </motion.div>
        )}
      </AnimatePresence>

      {/* ── Delete confirmation ── */}
      <ConfirmDialog
        open={!!deletingTask}
        title="Excluir tarefa?"
        message={`"${deletingTask?.title}" será removida do planejamento.`}
        confirmLabel="Excluir"
        onConfirm={() => {
          if (deletingTask) deleteTask.mutate(deletingTask.id)
          setDeletingTask(null)
        }}
        onCancel={() => setDeletingTask(null)}
      />

      {/* ── Add / Edit Task Modal ── */}
      {(showAdd || editingTask) && (
        <AddTaskModal
          date={dateStr}
          editTask={editingTask ?? undefined}
          onSave={input => {
            if (editingTask) {
              // Modo edição — PATCH
              updateTask.mutate(
                { id: editingTask.id, input },
                { onSuccess: () => setEditingTask(null) }
              )
            } else {
              // Modo criação
              createTask.mutate(input, { onSuccess: () => setShowAdd(false) })
            }
          }}
          onClose={() => { setShowAdd(false); setEditingTask(null) }}
          isPending={createTask.isPending || updateTask.isPending}
        />
      )}
    </div>
  )
}
