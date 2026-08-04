import { useState } from 'react'
import { useActivities, useHealthSummary, useBodyMetrics, useUpsertBodyMetric } from '@/hooks/useHealth'
import { toast } from '@/lib/toast'
import styles from './HealthPage.module.css'

function pad(n: number) {
  return n < 10 ? `0${n}` : `${n}`
}

function toDateStr(d: Date) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function formatMinutes(min: number): string {
  if (min < 60) return `${min}min`
  const h = Math.floor(min / 60)
  const m = min % 60
  return m > 0 ? `${h}h${pad(m)}` : `${h}h`
}

export function HealthPage() {
  const to = toDateStr(new Date())
  const from = toDateStr(new Date(Date.now() - 29 * 24 * 60 * 60 * 1000))

  const { data: activities = [], isLoading, isError } = useActivities(from, to)
  const { data: summary } = useHealthSummary(4)
  const { data: bodyMetrics = [] } = useBodyMetrics()
  const upsertBodyMetric = useUpsertBodyMetric()

  const [weight, setWeight] = useState('')

  const thisWeek = summary?.weeks[0]
  const latestWeight = bodyMetrics.find((m) => m.weight_kg != null)

  const handleSaveWeight = (e: React.FormEvent) => {
    e.preventDefault()
    const value = Number(weight.replace(',', '.'))
    if (!Number.isFinite(value) || value <= 0) {
      toast.error('Informe um peso válido')
      return
    }
    upsertBodyMetric.mutate(
      { weight_kg: value },
      { onSuccess: () => { setWeight(''); toast.success('Peso registrado') } }
    )
  }

  const maxKm = Math.max(...(summary?.weeks.map((w) => w.total_km) ?? [0]), 1)

  return (
    <div className={styles.page}>
      {/* ── Header ── */}
      <div className={styles.header}>
        <div>
          <h1 className={styles.pageTitle}>Saúde</h1>
          <p className={styles.pageSubtitle}>
            Treinos vêm dos seus hábitos de categoria Saúde — uma única fonte de verdade.
          </p>
        </div>
      </div>

      {/* ── Week tiles ── */}
      <div className={styles.statsGrid}>
        <div className={`glass-card ${styles.statCard}`}>
          <p className={styles.statLabel}>Sessões esta semana</p>
          <p className={styles.statValue}>{thisWeek?.sessions ?? 0}</p>
        </div>
        <div className={`glass-card ${styles.statCard}`}>
          <p className={styles.statLabel}>Volume (km)</p>
          <p className={styles.statValue}>{(thisWeek?.total_km ?? 0).toFixed(1)}</p>
        </div>
        <div className={`glass-card ${styles.statCard}`}>
          <p className={styles.statLabel}>Tempo de treino</p>
          <p className={styles.statValue}>{formatMinutes(thisWeek?.total_minutes ?? 0)}</p>
        </div>
        <div className={`glass-card ${styles.statCard}`}>
          <p className={styles.statLabel}>Peso atual</p>
          <p className={styles.statValue}>
            {latestWeight?.weight_kg != null ? `${latestWeight.weight_kg} kg` : '—'}
          </p>
        </div>
      </div>

      <div className={styles.columns}>
        {/* ── Activities ── */}
        <section className={`glass-card ${styles.section}`}>
          <h2 className={styles.sectionTitle}>Atividades (30 dias)</h2>
          {isLoading ? (
            <p className={styles.hint}>Carregando...</p>
          ) : isError ? (
            <p className={styles.hint}>Não foi possível carregar as atividades.</p>
          ) : activities.length === 0 ? (
            <p className={styles.hint}>
              Nenhum treino registrado. Crie um hábito com categoria "Saúde" e dê check-in —
              ele aparece aqui automaticamente.
            </p>
          ) : (
            <ul className={styles.activityList}>
              {activities.map((a) => (
                <li key={a.log_id} className={styles.activityItem}>
                  <span
                    className={styles.activityIcon}
                    style={{ background: a.color + '18', color: a.color }}
                  >
                    {a.icon}
                  </span>
                  <div className={styles.activityBody}>
                    <span className={styles.activityTitle}>{a.habit_name}</span>
                    <span className={styles.activityMeta}>
                      {new Date(a.date + 'T12:00').toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' })}
                      {a.metrics?.km != null && ` · ${a.metrics.km} km`}
                      {a.metrics?.time_min != null && ` · ${a.metrics.time_min} min`}
                      {a.timer_seconds != null && ` · ${Math.round(a.timer_seconds / 60)} min`}
                      {a.metrics?.rpe != null && ` · RPE ${a.metrics.rpe}`}
                      {a.source_type === 'track_day' && ' · via diário'}
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>

        <div className={styles.sideColumn}>
          {/* ── Weekly volume ── */}
          <section className={`glass-card ${styles.section}`}>
            <h2 className={styles.sectionTitle}>Volume semanal (km)</h2>
            {!summary || summary.weeks.length === 0 ? (
              <p className={styles.hint}>Sem dados ainda.</p>
            ) : (
              <ul className={styles.volumeList}>
                {summary.weeks.map((w) => (
                  <li key={w.week_start} className={styles.volumeRow} title={`${w.sessions} sessões · ${w.total_km.toFixed(1)} km · ${formatMinutes(w.total_minutes)}${w.avg_rpe > 0 ? ` · RPE médio ${w.avg_rpe.toFixed(1)}` : ''}`}>
                    <span className={styles.volumeLabel}>
                      {new Date(w.week_start + 'T12:00').toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' })}
                    </span>
                    <div className={styles.volumeTrack}>
                      <div
                        className={styles.volumeBar}
                        style={{ width: `${Math.min((w.total_km / maxKm) * 100, 100)}%` }}
                      />
                    </div>
                    <span className={styles.volumeValue}>{w.total_km.toFixed(1)}</span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          {/* ── Body metrics ── */}
          <section className={`glass-card ${styles.section}`}>
            <h2 className={styles.sectionTitle}>Peso corporal</h2>
            <form onSubmit={handleSaveWeight} className={styles.weightForm}>
              <input
                className="input"
                placeholder="Ex: 72,5"
                inputMode="decimal"
                value={weight}
                onChange={(e) => setWeight(e.target.value)}
              />
              <button type="submit" className="btn btn-ghost" disabled={upsertBodyMetric.isPending}>
                Registrar hoje
              </button>
            </form>
            {bodyMetrics.length > 0 && (
              <ul className={styles.weightList}>
                {bodyMetrics.slice(0, 8).map((m) => (
                  <li key={m.id} className={styles.weightRow}>
                    <span className={styles.weightDate}>
                      {new Date(m.measured_on + 'T12:00').toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' })}
                    </span>
                    <span className={styles.weightValue}>
                      {m.weight_kg != null ? `${m.weight_kg} kg` : '—'}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>
      </div>
    </div>
  )
}
