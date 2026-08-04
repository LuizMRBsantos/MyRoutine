import { useState } from 'react'
import styles from './DatePickerCalendar.module.css'

interface DatePickerCalendarProps {
  selectedDate: string // YYYY-MM-DD
  onSelect: (dateStr: string) => void
}

const WEEKDAYS = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb']

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`
}

function toDateStr(year: number, month: number, day: number): string {
  return `${year}-${pad(month + 1)}-${pad(day)}`
}

function parseDateStr(str: string) {
  const parts = str.split('-').map(Number)
  if (parts.length === 3 && !isNaN(parts[0]) && !isNaN(parts[1]) && !isNaN(parts[2])) {
    return { year: parts[0], month: parts[1] - 1, day: parts[2] }
  }
  const now = new Date()
  return { year: now.getFullYear(), month: now.getMonth(), day: now.getDate() }
}

export function DatePickerCalendar({ selectedDate, onSelect }: DatePickerCalendarProps) {
  const initial = parseDateStr(selectedDate)
  const [viewYear, setViewYear] = useState(initial.year)
  const [viewMonth, setViewMonth] = useState(initial.month) // 0-indexed

  const today = new Date()
  const todayStr = toDateStr(today.getFullYear(), today.getMonth(), today.getDate())

  const navigateMonth = (diff: number) => {
    let nextMonth = viewMonth + diff
    let nextYear = viewYear
    if (nextMonth < 0) {
      nextMonth = 11
      nextYear -= 1
    } else if (nextMonth > 11) {
      nextMonth = 0
      nextYear += 1
    }
    setViewYear(nextYear)
    setViewMonth(nextMonth)
  }

  const goToToday = () => {
    setViewYear(today.getFullYear())
    setViewMonth(today.getMonth())
    onSelect(todayStr)
  }

  // Título em português
  const monthTitle = new Date(viewYear, viewMonth, 1).toLocaleDateString('pt-BR', {
    month: 'long',
    year: 'numeric',
  })

  // Cálculos do grid de 42 dias
  const firstDayOfWeek = new Date(viewYear, viewMonth, 1).getDay() // 0-6
  const daysInCurrentMonth = new Date(viewYear, viewMonth + 1, 0).getDate()
  const daysInPrevMonth = new Date(viewYear, viewMonth, 0).getDate()

  const days: Array<{
    dateStr: string
    dayNumber: number
    isCurrentMonth: boolean
    isToday: boolean
    isSelected: boolean
  }> = []

  // Dias do mês anterior
  for (let i = firstDayOfWeek - 1; i >= 0; i--) {
    const d = daysInPrevMonth - i
    let prevYear = viewYear
    let prevMonth = viewMonth - 1
    if (prevMonth < 0) {
      prevMonth = 11
      prevYear -= 1
    }
    const ds = toDateStr(prevYear, prevMonth, d)
    days.push({
      dateStr: ds,
      dayNumber: d,
      isCurrentMonth: false,
      isToday: ds === todayStr,
      isSelected: ds === selectedDate,
    })
  }

  // Dias do mês atual
  for (let d = 1; d <= daysInCurrentMonth; d++) {
    const ds = toDateStr(viewYear, viewMonth, d)
    days.push({
      dateStr: ds,
      dayNumber: d,
      isCurrentMonth: true,
      isToday: ds === todayStr,
      isSelected: ds === selectedDate,
    })
  }

  // Dias do próximo mês até completar 42 células (ou 35 se já couber)
  const totalCells = days.length > 35 ? 42 : 35
  const remaining = totalCells - days.length
  for (let d = 1; d <= remaining; d++) {
    let nextYear = viewYear
    let nextMonth = viewMonth + 1
    if (nextMonth > 11) {
      nextMonth = 0
      nextYear += 1
    }
    const ds = toDateStr(nextYear, nextMonth, d)
    days.push({
      dateStr: ds,
      dayNumber: d,
      isCurrentMonth: false,
      isToday: ds === todayStr,
      isSelected: ds === selectedDate,
    })
  }

  return (
    <div className={styles.container}>
      {/* Header */}
      <div className={styles.header}>
        <span className={styles.monthTitle}>{monthTitle}</span>
        <div className={styles.navGroup}>
          <button type="button" className={styles.todayBtn} onClick={goToToday}>
            Hoje
          </button>
          <button
            type="button"
            className={styles.iconBtn}
            onClick={() => navigateMonth(-1)}
            title="Mês anterior"
          >
            ‹
          </button>
          <button
            type="button"
            className={styles.iconBtn}
            onClick={() => navigateMonth(1)}
            title="Próximo mês"
          >
            ›
          </button>
        </div>
      </div>

      {/* Weekdays */}
      <div className={styles.weekdays}>
        {WEEKDAYS.map((w) => (
          <div key={w} className={styles.weekday}>
            {w}
          </div>
        ))}
      </div>

      {/* Days Grid */}
      <div className={styles.grid}>
        {days.map((item) => (
          <button
            key={item.dateStr}
            type="button"
            onClick={() => onSelect(item.dateStr)}
            className={`
              ${styles.dayBtn}
              ${!item.isCurrentMonth ? styles.dayDimmed : ''}
              ${item.isToday ? styles.dayToday : ''}
              ${item.isSelected ? styles.daySelected : ''}
            `}
          >
            {item.dayNumber}
          </button>
        ))}
      </div>
    </div>
  )
}
