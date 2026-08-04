import { useMemo } from 'react'
import type { HeatmapEntry } from '@/types/habit'
import styles from './HabitHeatmap.module.css'

interface HabitHeatmapProps {
  data: HeatmapEntry[]
  days?: number
}

export function HabitHeatmap({ data, days = 90 }: HabitHeatmapProps) {
  const cells = useMemo(() => {
    const map = new Map(data.map(e => [e.date, e.habits_completed]))
    const result = []
    const today = new Date()

    for (let i = days - 1; i >= 0; i--) {
      const d = new Date(today)
      d.setDate(d.getDate() - i)
      const pad = (n: number) => (n < 10 ? `0${n}` : `${n}`)
      const dateStr = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
      const count = map.get(dateStr) ?? 0
      result.push({ date: dateStr, count })
    }
    return result
  }, [data, days])

  const maxCount = useMemo(() => Math.max(1, ...cells.map(c => c.count)), [cells])

  const getLevel = (count: number) => {
    if (count === 0) return 0
    const ratio = count / maxCount
    if (ratio < 0.25) return 1
    if (ratio < 0.5) return 2
    if (ratio < 0.75) return 3
    return 4
  }

  return (
    <div className={styles.wrapper}>
      <div className={styles.grid}>
        {cells.map(cell => (
          <div
            key={cell.date}
            className={styles.cell}
            data-level={getLevel(cell.count)}
            title={`${cell.date}: ${cell.count} hábito${cell.count !== 1 ? 's' : ''}`}
          />
        ))}
      </div>
      <div className={styles.legend}>
        <span className={styles.legendLabel}>Menos</span>
        {[0, 1, 2, 3, 4].map(l => (
          <div key={l} className={styles.legendCell} data-level={l} />
        ))}
        <span className={styles.legendLabel}>Mais</span>
      </div>
    </div>
  )
}
