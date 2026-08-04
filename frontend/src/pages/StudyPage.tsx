import { useState } from 'react'
import { useStudySessions, useStudySummary, useCreateStudySession, useDeleteStudySession } from '@/hooks/useStudy'
import { useHabits } from '@/hooks/useHabits'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { toast } from '@/lib/toast'
import type { StudySession } from '@/types/study'
import styles from './StudyPage.module.css'

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

export function StudyPage() {
  const to = toDateStr(new Date())
  const from = toDateStr(new Date(Date.now() - 29 * 24 * 60 * 60 * 1000))

  const { data: sessions = [], isLoading, isError } = useStudySessions(from, to)
  const { data: summary } = useStudySummary()
  const { data: habits = [] } = useHabits()
  const createSession = useCreateStudySession()
  const deleteSession = useDeleteStudySession()

  const studyHabits = habits.filter((h) => h.category === 'study')

  const [subject, setSubject] = useState('')
  const [topic, setTopic] = useState('')
  const [duration, setDuration] = useState('')
  const [habitId, setHabitId] = useState('')
  const [deleting, setDeleting] = useState<StudySession | null>(null)

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault()
    const minutes = parseInt(duration)
    if (!subject.trim()) {
      toast.error('Informe a matéria')
      return
    }
    if (!Number.isFinite(minutes) || minutes <= 0) {
      toast.error('Informe a duração em minutos')
      return
    }
    createSession.mutate(
      {
        subject: subject.trim(),
        topic: topic.trim() || undefined,
        duration_minutes: minutes,
        habit_id: habitId || undefined,
      },
      {
        onSuccess: () => {
          setTopic('')
          setDuration('')
          toast.success('Sessão registrada')
        },
      }
    )
  }

  const maxSubjectMinutes = Math.max(...(summary?.by_subject.map((s) => s.total_minutes) ?? [0]), 1)

  return (
    <div className={styles.page}>
      {/* ── Header ── */}
      <div className={styles.header}>
        <div>
          <h1 className={styles.pageTitle}>Estudos</h1>
          <p className={styles.pageSubtitle}>Sessões de estudo por matéria — últimas 4 semanas</p>
        </div>
      </div>

      {/* ── Tiles ── */}
      <div className={styles.statsGrid}>
        <div className={`glass-card ${styles.statCard}`}>
          <p className={styles.statLabel}>Tempo estudado (30d)</p>
          <p className={styles.statValue}>{formatMinutes(summary?.total_minutes_30d ?? 0)}</p>
        </div>
        <div className={`glass-card ${styles.statCard}`}>
          <p className={styles.statLabel}>Sessões (30d)</p>
          <p className={styles.statValue}>{summary?.sessions_30d ?? 0}</p>
        </div>
        <div className={`glass-card ${styles.statCard}`}>
          <p className={styles.statLabel}>Matérias ativas</p>
          <p className={styles.statValue}>{summary?.by_subject.length ?? 0}</p>
        </div>
      </div>

      {/* ── New session ── */}
      <form onSubmit={handleCreate} className={`glass-card ${styles.addCard}`}>
        <div className={styles.addFields}>
          <input
            className="input"
            placeholder="Matéria (ex: OAC)"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
          />
          <input
            className="input"
            placeholder="Tópico (opcional)"
            value={topic}
            onChange={(e) => setTopic(e.target.value)}
          />
          <input
            className="input"
            type="number"
            min="1"
            placeholder="Minutos"
            value={duration}
            onChange={(e) => setDuration(e.target.value)}
          />
          {studyHabits.length > 0 && (
            <select
              className="input"
              value={habitId}
              onChange={(e) => setHabitId(e.target.value)}
              title="Vincular a um hábito de estudo gera o check-in automaticamente"
            >
              <option value="">Sem vínculo com hábito</option>
              {studyHabits.map((h) => (
                <option key={h.id} value={h.id}>{h.icon} {h.name}</option>
              ))}
            </select>
          )}
          <button type="submit" className="btn btn-primary" disabled={createSession.isPending}>
            Registrar
          </button>
        </div>
        {habitId && (
          <p className={styles.linkHint}>
            Ao registrar, o check-in do hábito vinculado é feito automaticamente.
          </p>
        )}
      </form>

      <div className={styles.columns}>
        {/* ── Sessions ── */}
        <section className={`glass-card ${styles.section}`}>
          <h2 className={styles.sectionTitle}>Sessões recentes</h2>
          {isLoading ? (
            <p className={styles.hint}>Carregando...</p>
          ) : isError ? (
            <p className={styles.hint}>Não foi possível carregar as sessões.</p>
          ) : sessions.length === 0 ? (
            <p className={styles.hint}>Nenhuma sessão registrada nos últimos 30 dias.</p>
          ) : (
            <ul className={styles.sessionList}>
              {sessions.map((s) => (
                <li key={s.id} className={styles.sessionItem}>
                  <div className={styles.sessionBody}>
                    <span className={styles.sessionTitle}>
                      {s.subject}
                      {s.topic && <span className={styles.sessionTopic}> — {s.topic}</span>}
                    </span>
                    <span className={styles.sessionMeta}>
                      {new Date(s.studied_on + 'T12:00').toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' })}
                      {' · '}{formatMinutes(s.duration_minutes)}
                    </span>
                  </div>
                  <button
                    className={styles.deleteBtn}
                    title="Remover sessão"
                    onClick={() => setDeleting(s)}
                  >
                    ×
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>

        {/* ── By subject ── */}
        <section className={`glass-card ${styles.section}`}>
          <h2 className={styles.sectionTitle}>Por matéria (30d)</h2>
          {!summary || summary.by_subject.length === 0 ? (
            <p className={styles.hint}>Sem dados ainda.</p>
          ) : (
            <ul className={styles.subjectList}>
              {summary.by_subject.map((s) => (
                <li
                  key={s.subject}
                  className={styles.subjectRow}
                  title={`${s.subject}: ${formatMinutes(s.total_minutes)} em ${s.sessions} sessões`}
                >
                  <span className={styles.subjectLabel}>{s.subject}</span>
                  <div className={styles.subjectTrack}>
                    <div
                      className={styles.subjectBar}
                      style={{ width: `${Math.min((s.total_minutes / maxSubjectMinutes) * 100, 100)}%` }}
                    />
                  </div>
                  <span className={styles.subjectValue}>{formatMinutes(s.total_minutes)}</span>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>

      <ConfirmDialog
        open={!!deleting}
        title="Remover sessão?"
        message={`A sessão de ${deleting?.subject} (${deleting ? formatMinutes(deleting.duration_minutes) : ''}) será removida. Um check-in de hábito já gerado por ela é preservado.`}
        confirmLabel="Remover"
        onConfirm={() => {
          if (deleting) deleteSession.mutate(deleting.id)
          setDeleting(null)
        }}
        onCancel={() => setDeleting(null)}
      />
    </div>
  )
}
