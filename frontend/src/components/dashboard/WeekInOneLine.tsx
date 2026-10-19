import { useConsistency } from '@/hooks/useConsistency'
import styles from './WeekInOneLine.module.css'

const WEEKDAYS = ['seg', 'ter', 'qua', 'qui', 'sex', 'sáb', 'dom']

// Símbolo e nome acessível de cada dia (mesmos códigos de /habits/consistency).
const MARK: Record<string, [string, string]> = {
  x: ['●', 'feito'],
  o: ['○', 'sem registro'],
  '.': ['○', 'hoje, ainda em aberto'],
  '-': ['–', 'não é dia'],
  ' ': ['', 'antes do hábito existir'],
}

// A semana atual (segunda a domingo), hábito × dia: o que ainda dá para
// salvar nesta semana. Dias que ainda não chegaram ficam como "·".
export function WeekInOneLine() {
  const { data, isLoading } = useConsistency()
  if (isLoading || !data || data.habits.length === 0) return null

  const todayIdx = (new Date(`${data.to}T12:00:00`).getDay() + 6) % 7 // seg = 0

  return (
    <div className={styles.wrap}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th scope="col" className={styles.habitCol}>Hábito</th>
            {WEEKDAYS.map((d, i) => (
              <th key={d} scope="col" className={i === todayIdx ? styles.today : undefined}>{d}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.habits.map(h => {
            const week = h.days.slice(h.days.length - (todayIdx + 1))
            return (
              <tr key={h.id}>
                <th scope="row" className={styles.habitCol}>
                  <span aria-hidden="true">{h.icon}</span> {h.name}
                </th>
                {WEEKDAYS.map((d, i) => {
                  if (i > todayIdx) return <td key={d} className={styles.future} aria-label={`${d}: ainda não chegou`}>·</td>
                  const [sym, label] = MARK[week[i]] ?? MARK[' ']
                  return (
                    <td key={d} className={styles[`m_${week[i] === '.' ? 'today' : week[i] === ' ' ? 'none' : week[i] === '-' ? 'off' : week[i]}`]} aria-label={`${d}: ${label}`}>
                      {sym}
                    </td>
                  )
                })}
              </tr>
            )
          })}
        </tbody>
      </table>
      <p className={styles.legend}>● feito · ○ sem registro · – não é dia</p>
    </div>
  )
}
