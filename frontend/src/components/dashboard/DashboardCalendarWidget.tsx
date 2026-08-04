import { useState, useEffect, useMemo, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useWeekTasks, useMonthTasks, useCreateTask, useUpdateTask } from '@/hooks/useTasks'
import { AddTaskModal } from '@/components/planner/AddTaskModal'
import { MonthCalendarView } from '@/components/planner/MonthCalendarView'
import type { Task } from '@/types/task'
import { CATEGORY_META } from '@/types/task'
import styles from './DashboardCalendarWidget.module.css'

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`
}

function toLocalStr(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

const START_HOUR = 5 // Grid começa às 05:00 (19 horas no total até 23:59)
const HOURS_COUNT = 19
const HOUR_HEIGHT = 60 // 1px = 1 minuto

export function DashboardCalendarWidget() {
  const [viewMode, setViewMode] = useState<'Semana' | 'Mês'>('Semana')
  const [viewDate, setViewDate] = useState(() => new Date())

  // Modais de Criação / Edição
  const [selectedAddDate, setSelectedAddDate] = useState<string | null>(null)
  const [selectedAddTime, setSelectedAddTime] = useState<string | undefined>(undefined)
  const [editingTask, setEditingTask] = useState<Task | null>(null)

  const scrollRef = useRef<HTMLDivElement>(null)

  // Relógio em tempo real para o ponteiro vermelho
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 60000)
    return () => clearInterval(id)
  }, [])

  const todayStr = useMemo(() => toLocalStr(now), [now])

  // Cálculo das datas da semana (Dom - Sáb)
  const weekDays = useMemo(() => {
    const dayOfWeek = viewDate.getDay() // 0=Dom
    const startObj = new Date(viewDate)
    startObj.setDate(viewDate.getDate() - dayOfWeek)

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
  }, [viewDate, todayStr])

  const startStr = weekDays[0].dateStr
  const endStr = weekDays[6].dateStr

  // Queries
  const { data: weekTasks = [] } = useWeekTasks(startStr, endStr)
  const { data: monthTasks = [] } = useMonthTasks(viewDate)

  // Mutations
  const createTask = useCreateTask()
  const updateTask = useUpdateTask()

  // Agrupar tarefas da semana por data
  const tasksByDate = useMemo(() => {
    const map = new Map<string, { allDay: Task[]; timed: Task[] }>()
    for (const t of weekTasks) {
      const entry = map.get(t.date) ?? { allDay: [], timed: [] }
      if (!t.start_time || parseInt(t.start_time.slice(0, 2), 10) < START_HOUR) {
        entry.allDay.push(t)
      } else {
        entry.timed.push(t)
      }
      map.set(t.date, entry)
    }
    return map
  }, [weekTasks])

  // Horas para renderizar no eixo esquerdo (05:00 - 23:00)
  const hoursList = useMemo(() => {
    const list = []
    for (let i = 0; i < HOURS_COUNT; i++) {
      list.push(START_HOUR + i)
    }
    return list
  }, [])

  // Cálculo da posição vertical (top px) da linha vermelha do tempo atual
  const currentTimeTopPx = useMemo(() => {
    const h = now.getHours()
    const m = now.getMinutes()
    if (h < START_HOUR || h >= START_HOUR + HOURS_COUNT) return null
    return (h - START_HOUR) * HOUR_HEIGHT + m
  }, [now])

  // Funções de navegação do calendário
  const navigateDate = (delta: number) => {
    if (viewMode === 'Semana') {
      const next = new Date(viewDate)
      next.setDate(viewDate.getDate() + delta * 7)
      setViewDate(next)
    } else {
      const next = new Date(viewDate.getFullYear(), viewDate.getMonth() + delta, 1)
      setViewDate(next)
    }
  }

  const handleTimeSlotClick = (dateStr: string, e: React.MouseEvent<HTMLDivElement>) => {
    const rect = e.currentTarget.getBoundingClientRect()
    const offsetY = e.clientY - rect.top + (e.currentTarget.scrollTop || 0)
    const hourClicked = START_HOUR + Math.floor(offsetY / HOUR_HEIGHT)
    const clampedHour = Math.max(START_HOUR, Math.min(START_HOUR + HOURS_COUNT - 1, hourClicked))

    setSelectedAddTime(`${pad(clampedHour)}:00`)
    setSelectedAddDate(dateStr)
  }

  return (
    <motion.div
      className={styles.container}
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4 }}
    >
      {/* ── Header e Controles ── */}
      <div className={styles.header}>
        <div className={styles.titleArea}>
          <h2 className={styles.title}>
            <span>🗓️</span>
            <span>Calendário Interativo</span>
          </h2>
          <p className={styles.subtitle}>
            Visão Google Calendar com ponteiro de tempo ao vivo e blocos proporcionais
          </p>
        </div>

        <div className={styles.controls}>
          <div className={styles.navGroup}>
            <button type="button" className={styles.navBtn} onClick={() => navigateDate(-1)}>
              ‹
            </button>
            <button
              type="button"
              className={styles.navToday}
              onClick={() => setViewDate(new Date())}
            >
              Hoje
            </button>
            <button type="button" className={styles.navBtn} onClick={() => navigateDate(1)}>
              ›
            </button>
          </div>

          <div className={styles.viewTabs}>
            <button
              type="button"
              className={`${styles.viewTab} ${viewMode === 'Semana' ? styles.viewTabActive : ''}`}
              onClick={() => setViewMode('Semana')}
            >
              Semana
            </button>
            <button
              type="button"
              className={`${styles.viewTab} ${viewMode === 'Mês' ? styles.viewTabActive : ''}`}
              onClick={() => setViewMode('Mês')}
            >
              Mês
            </button>
          </div>
        </div>
      </div>

      {/* ══════════════ VISÃO SEMANA (ESTILO GOOGLE CALENDAR) ══════════════ */}
      {viewMode === 'Semana' && (
        <div className={styles.weekView}>
          {/* Cabeçalho dos Dias da Semana */}
          <div className={styles.weekHeaderRow}>
            <div className={styles.timeGutterHeader}>GMT-3</div>
            {weekDays.map((day) => (
              <div
                key={day.dateStr}
                className={`${styles.dayHeaderCell} ${
                  day.isToday ? styles.dayHeaderToday : ''
                }`}
              >
                <span className={styles.dayName}>{day.dayName}</span>
                <span
                  className={`${styles.dayNum} ${
                    day.isToday ? styles.dayNumToday : ''
                  }`}
                >
                  {day.dayNum}
                </span>
              </div>
            ))}
          </div>

          {/* Seção O dia todo / Sem horário fixo */}
          <div className={styles.allDayRow}>
            <div className={styles.allDayLabel}>O dia todo</div>
            {weekDays.map((day) => {
              const dayEntry = tasksByDate.get(day.dateStr)
              const allDayTasks = dayEntry?.allDay ?? []
              return (
                <div
                  key={`allday-${day.dateStr}`}
                  className={styles.allDayCell}
                  onClick={() => {
                    setSelectedAddTime(undefined)
                    setSelectedAddDate(day.dateStr)
                  }}
                >
                  {allDayTasks.map((t) => {
                    const meta = CATEGORY_META[t.category] ?? CATEGORY_META.other
                    return (
                      <div
                        key={t.id}
                        className={styles.allDayPill}
                        style={{
                          backgroundColor: meta.color + '22',
                          color: meta.color,
                          borderColor: meta.color + '44',
                        }}
                        title={`${meta.label}: ${t.title}`}
                        onClick={(e) => {
                          e.stopPropagation()
                          setEditingTask(t)
                        }}
                      >
                        <span>{meta.icon}</span>
                        <span>{t.title}</span>
                      </div>
                    )
                  })}
                </div>
              )
            })}
          </div>

          {/* Grid do Tempo (05:00 às 23:00) */}
          <div className={styles.timeGridScroll} ref={scrollRef}>
            {/* Eixo de Horários */}
            <div className={styles.timeGutter}>
              {hoursList.map((h) => (
                <div key={h} className={styles.hourLabel}>
                  {pad(h)}:00
                </div>
              ))}
            </div>

            {/* Colunas dos 7 Dias da Semana */}
            {weekDays.map((day) => {
              const dayEntry = tasksByDate.get(day.dateStr)
              const timedTasks = dayEntry?.timed ?? []

              return (
                <div
                  key={`col-${day.dateStr}`}
                  className={`${styles.dayColumn} ${
                    day.isToday ? styles.dayColumnToday : ''
                  }`}
                  onClick={(e) => handleTimeSlotClick(day.dateStr, e)}
                >
                  {/* Linhas de Horário e Meia-Hora */}
                  {hoursList.map((h, i) => (
                    <div key={`line-${h}`}>
                      <div
                        className={styles.hourLine}
                        style={{ top: `${i * HOUR_HEIGHT}px` }}
                      />
                      <div
                        className={styles.halfHourLine}
                        style={{ top: `${i * HOUR_HEIGHT + HOUR_HEIGHT / 2}px` }}
                      />
                    </div>
                  ))}

                  {/* ── LINHA VERMELHA DO TEMPO ATUAL (SE HOJE) ── */}
                  {day.isToday && currentTimeTopPx !== null && (
                    <div
                      className={styles.redTimeLine}
                      style={{ top: `${currentTimeTopPx}px` }}
                    >
                      <div className={styles.redTimeDot} />
                    </div>
                  )}

                  {/* ── BLOCOS PROPORCIONAIS DE TAREFA ── */}
                  {timedTasks.map((t) => {
                    const parts = t.start_time!.split(':').map(Number)
                    const startH = parts[0]
                    const startM = parts[1] || 0
                    const topPx = (startH - START_HOUR) * HOUR_HEIGHT + startM
                    const duration = t.duration_minutes || 60
                    const heightPx = Math.max(26, duration * (HOUR_HEIGHT / 60))

                    const meta = CATEGORY_META[t.category] ?? CATEGORY_META.other

                    // Calcular horário final
                    const endMinTotal = startH * 60 + startM + duration
                    const endH = Math.floor(endMinTotal / 60)
                    const endM = endMinTotal % 60
                    const timeRangeStr = `${pad(startH)}:${pad(startM)} - ${pad(endH)}:${pad(endM)}`

                    return (
                      <div
                        key={t.id}
                        className={styles.taskBlock}
                        style={{
                          top: `${topPx}px`,
                          height: `${heightPx}px`,
                          backgroundColor: meta.color + '26',
                          borderLeft: `3.5px solid ${meta.color}`,
                          color: meta.color,
                        }}
                        title={`${meta.label}: ${t.title} (${timeRangeStr}) - Clique para editar`}
                        onClick={(e) => {
                          e.stopPropagation()
                          setEditingTask(t)
                        }}
                      >
                        <div className={styles.taskBlockHeader}>
                          <span className={styles.taskBlockTitle}>
                            {meta.icon} {t.title}
                          </span>
                          <span className={styles.taskBlockTime}>{timeRangeStr}</span>
                        </div>
                        {heightPx >= 45 && t.notes && (
                          <div className={styles.taskBlockDesc}>{t.notes}</div>
                        )}
                      </div>
                    )
                  })}
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* ══════════════ VISÃO MÊS ══════════════ */}
      {viewMode === 'Mês' && (
        <div className={styles.monthContainer}>
          <MonthCalendarView
            viewDate={viewDate}
            tasks={monthTasks}
            onSelectDay={(dStr) => {
              setSelectedAddTime(undefined)
              setSelectedAddDate(dStr)
            }}
            onSelectTask={(task) => setEditingTask(task)}
          />
        </div>
      )}

      {/* ── Modal Criar / Editar Tarefa ── */}
      <AnimatePresence>
        {(selectedAddDate || editingTask) && (
          <AddTaskModal
            date={selectedAddDate || editingTask?.date || todayStr}
            editTask={editingTask ?? undefined}
            defaultStartTime={selectedAddTime}
            onSave={(input) => {
              if (editingTask) {
                updateTask.mutate(
                  { id: editingTask.id, input },
                  {
                    onSuccess: () => {
                      setEditingTask(null)
                      setSelectedAddDate(null)
                    },
                  }
                )
              } else {
                createTask.mutate(input, {
                  onSuccess: () => {
                    setSelectedAddDate(null)
                  },
                })
              }
            }}
            onClose={() => {
              setSelectedAddDate(null)
              setEditingTask(null)
            }}
            isPending={createTask.isPending || updateTask.isPending}
          />
        )}
      </AnimatePresence>
    </motion.div>
  )
}
