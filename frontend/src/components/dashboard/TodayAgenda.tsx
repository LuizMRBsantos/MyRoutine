import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTasks } from '@/hooks/useTasks'
import type { Task } from '@/types/task'
import styles from './TodayAgenda.module.css'

function localDate(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

function minutesOf(hhmm: string): number {
  const [h, m] = hhmm.split(':').map(Number)
  return h * 60 + (m || 0)
}

function untilLabel(mins: number): string {
  if (mins < 60) return `em ${mins} min`
  const h = Math.floor(mins / 60)
  const m = mins % 60
  return m ? `em ${h} h ${m} min` : `em ${h} h`
}

// Agenda de hoje: o que já passou fica discreto e o próximo compromisso ganha
// destaque com "em 40 min". Tarefas sem horário vêm por último.
export function TodayAgenda() {
  const [now, setNow] = useState(() => new Date())
  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 60_000)
    return () => clearInterval(id)
  }, [])
  const { data: tasks = [], isLoading, isError } = useTasks(localDate(now))

  if (isLoading) return <div className={styles.skeleton} />
  if (isError) return <p className={styles.empty} role="status">Não deu para carregar a agenda agora.</p>

  const pending = (t: Task) => t.status === 'planned' || t.status === 'in_progress'
  const timed = tasks.filter(t => t.start_time).sort((a, b) => a.start_time!.localeCompare(b.start_time!))
  const untimed = tasks.filter(t => !t.start_time)
  const nowMin = now.getHours() * 60 + now.getMinutes()
  const next = timed.find(t => pending(t) && minutesOf(t.start_time!) >= nowMin)

  if (tasks.length === 0) {
    return <p className={styles.empty}>Nada marcado para hoje. <Link to="/planner">Abrir o Planner</Link></p>
  }

  return (
    <ul className={styles.list}>
      {timed.map(t => {
        const start = minutesOf(t.start_time!)
        const past = !pending(t) || start + (t.duration_minutes ?? 0) < nowMin
        const isNext = t === next
        return (
          <li key={t.id} className={`${styles.item} ${isNext ? styles.next : ''} ${past && !isNext ? styles.past : ''}`}>
            <time className={styles.time}>{t.start_time!.slice(0, 5)}</time>
            <div className={styles.body}>
              <span className={styles.title}>{t.title}</span>
              <span className={styles.detail}>
                {isNext ? untilLabel(start - nowMin) : t.duration_minutes ? `${t.duration_minutes} min` : ' '}
                {isNext && t.notify ? ' · aviso ligado' : ''}
              </span>
            </div>
          </li>
        )
      })}
      {untimed.map(t => (
        <li key={t.id} className={`${styles.item} ${pending(t) ? '' : styles.past}`}>
          <span className={styles.time}>—</span>
          <div className={styles.body}>
            <span className={styles.title}>{t.title}</span>
            <span className={styles.detail}>sem horário</span>
          </div>
        </li>
      ))}
    </ul>
  )
}
