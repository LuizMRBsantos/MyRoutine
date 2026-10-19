import { useConsistency, type HabitConsistency as Row } from '@/hooks/useConsistency'
import styles from './HabitConsistency.module.css'

const CELL: Record<string, string> = { x: 'done', o: 'open', '-': 'off', '.': 'today', ' ': 'none' }

function daysLabel(n: number): string {
  return n === 1 ? '1 dia' : `${n} dias`
}

// Constância de cada hábito nos últimos 90 dias: uma faixa com um quadrinho
// por dia e contagens concretas ("52 de 78 dias programados"), comparadas aos
// 90 dias anteriores. Sem porcentagem, nota ou cor de alarme
// (product-constitution) — dia sem registro é neutro.
export function HabitConsistency() {
  const { data, isLoading, isError } = useConsistency()

  if (isLoading) return <div className={styles.skeleton} />
  if (isError) return <p className={styles.note} role="status">Não deu para carregar a constância agora.</p>
  if (!data || data.habits.length === 0) return <p className={styles.note}>Crie um hábito para ver a constância dele aqui.</p>

  return (
    <div className={styles.list}>
      {data.habits.map(h => <HabitRow key={h.id} habit={h} />)}
      <div className={styles.footer}>
        <span>90 dias atrás</span>
        <span className={styles.legend}>
          <span className={`${styles.cell} ${styles.done}`} aria-hidden="true" /> feito
          <span className={`${styles.cell} ${styles.open}`} aria-hidden="true" /> sem registro
          <span className={`${styles.cell} ${styles.off}`} aria-hidden="true" /> não é dia
        </span>
        <span>hoje</span>
      </div>
    </div>
  )
}

function HabitRow({ habit }: { habit: Row }) {
  const before = habit.previous_scheduled > 0
    ? `antes: ${habit.previous_done} de ${habit.previous_scheduled}`
    : 'hábito novo'
  const summary = `${habit.name}: ${habit.done} de ${daysLabel(habit.scheduled)} programados nos últimos 90 dias; ${before}.`

  return (
    <div className={styles.row}>
      <div className={styles.head}>
        <span className={styles.name}><span aria-hidden="true">{habit.icon}</span> {habit.name}</span>
        <span className={styles.counts}>
          <strong>{habit.done}</strong> de {daysLabel(habit.scheduled)}
          <span className={styles.before}> · {before}</span>
        </span>
      </div>
      <div className={styles.strip} role="img" aria-label={summary}>
        {[...habit.days].map((c, i) => (
          <span key={i} className={`${styles.cell} ${styles[CELL[c] ?? 'none']}`} />
        ))}
      </div>
    </div>
  )
}
