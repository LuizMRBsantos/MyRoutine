import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { useJournal, useSaveJournal, useRegisterJournalItem, useUnregisterJournalItem } from '@/hooks/useJournal'
import { useHabits } from '@/hooks/useHabits'
import { apiErrorMessage } from '@/lib/toast'
import { lineKey, parseNoteContent, resolveHabitForWorkout, type ParsedLine } from '@/lib/trackDay/parser'
import { categoryIcon, categoryLabel, formatCents } from '@/types/finance'
import type { Journal, JournalItem } from '@/types/journal'
import styles from './JournalPage.module.css'

// Diário (Track Day): texto livre do dia, estilo Bullet Journal. O que o leitor
// reconhece (gastos, treinos) aparece ao lado e só vira registro com um toque
// em "Registrar" — o texto muda enquanto é digitado ("$ 5" antes de "$ 50").

const AUTOSAVE_MS = 800

const PLACEHOLDER = [
  '• Revisar o relatório',
  'x Treino da manhã',
  '$ 32,50 Almoço #alimentacao',
  '🏃 corrida 5km 30min RPE 6',
  '- Conversa boa com [[Ana]]',
].join('\n')

const WORKOUT_NAMES = { run: 'Corrida', bike: 'Ciclismo', swim: 'Natação' } as const
const WORKOUT_ICONS = { run: '🏃', bike: '🚴', swim: '🏊' } as const

function toISO(d: Date): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function shiftDay(iso: string, delta: number): string {
  const [y, m, d] = iso.split('-').map(Number)
  return toISO(new Date(y, m - 1, d + delta))
}

function dayLabel(iso: string): string {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('pt-BR', { weekday: 'long', day: 'numeric', month: 'long' })
}

function registerErrorMessage(err: unknown): string {
  const msg = apiErrorMessage(err, '')
  if (msg === 'habit already checked in that day') {
    return 'Esse hábito já foi marcado neste dia fora do diário — nada foi alterado.'
  }
  if (msg.includes('deadline passed')) return 'O prazo desse hábito já tinha passado nesse dia.'
  if (msg === 'invalid reference') return 'Hábito não encontrado.'
  return 'Não foi possível registrar. Tente de novo.'
}

type SaveState = { kind: 'idle' } | { kind: 'saving' } | { kind: 'saved'; at: Date } | { kind: 'error' }

export function JournalPage() {
  const [date, setDate] = useState(() => toISO(new Date()))
  const journal = useJournal(date)

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <div>
          <h1 className={styles.pageTitle}>Diário</h1>
          <p className={styles.dateLabel}>{dayLabel(date)}</p>
        </div>
      </div>

      {journal.isLoading && <p className={styles.hint}>Carregando…</p>}
      {journal.isError && <p className={styles.hint}>Não deu para carregar o diário agora.</p>}
      {journal.data && (
        // key: ao trocar de dia, o editor recomeça com o texto daquele dia.
        <JournalDay key={date} date={date} journal={journal.data} onChangeDate={setDate} />
      )}
    </div>
  )
}

function JournalDay({ date, journal, onChangeDate }: {
  date: string
  journal: Journal
  onChangeDate: (date: string) => void
}) {
  const save = useSaveJournal(date)
  const [content, setContent] = useState(journal.content)
  const [saveState, setSaveState] = useState<SaveState>(
    journal.updated_at ? { kind: 'saved', at: new Date(journal.updated_at) } : { kind: 'idle' }
  )
  const savedContent = useRef(journal.content)
  const dirty = content !== savedContent.current

  const saveNow = useCallback(async (): Promise<boolean> => {
    if (content === savedContent.current) return true
    const snapshot = content
    setSaveState({ kind: 'saving' })
    try {
      await save.mutateAsync(snapshot)
      savedContent.current = snapshot
      setSaveState({ kind: 'saved', at: new Date() })
      return true
    } catch {
      setSaveState({ kind: 'error' })
      return false
    }
  }, [content, save])

  // Salvamento automático depois de uma pausa na digitação.
  useEffect(() => {
    if (!dirty) return
    const t = setTimeout(() => { void saveNow() }, AUTOSAVE_MS)
    return () => clearTimeout(t)
  }, [content, dirty, saveNow])

  // Voltou a internet com texto pendente: salva.
  useEffect(() => {
    const onOnline = () => { if (content !== savedContent.current) void saveNow() }
    window.addEventListener('online', onOnline)
    return () => window.removeEventListener('online', onOnline)
  }, [content, saveNow])

  const goTo = async (next: string) => {
    if (dirty && !(await saveNow())) return // não troca de dia perdendo texto
    onChangeDate(next)
  }

  const today = toISO(new Date())

  return (
    <>
      <div className={styles.dayNav}>
        <button type="button" className="btn btn-ghost" aria-label="Dia anterior" onClick={() => goTo(shiftDay(date, -1))}>‹</button>
        <button type="button" className="btn btn-ghost" disabled={date === today} onClick={() => goTo(today)}>Hoje</button>
        <button type="button" className="btn btn-ghost" aria-label="Próximo dia" onClick={() => goTo(shiftDay(date, 1))}>›</button>
      </div>

      <div className={styles.layout}>
        <section className={`glass-card ${styles.editorCard}`}>
          <textarea
            className={`input ${styles.editor}`}
            aria-label="Texto do dia"
            placeholder={PLACEHOLDER}
            value={content}
            onChange={e => setContent(e.target.value)}
            onBlur={() => { void saveNow() }}
            maxLength={20000}
          />
          <div className={styles.editorFooter}>
            <SaveIndicator state={saveState} dirty={dirty} onRetry={() => { void saveNow() }} />
          </div>
          <p className={styles.syntax}>
            <code>•</code> tarefa · <code>x</code> feita · <code>&gt;</code> adiada · <code>-</code> nota ·{' '}
            <code>○</code> evento · <code>$ 50 Almoço #alimentacao</code> gasto ·{' '}
            <code>corrida 5km 30min RPE 6</code> treino · <code>[[nome]]</code> link
          </p>
        </section>

        <RecognizedPanel date={date} content={content} journal={journal} saveNow={saveNow} />
      </div>
    </>
  )
}

function SaveIndicator({ state, dirty, onRetry }: { state: SaveState; dirty: boolean; onRetry: () => void }) {
  if (state.kind === 'error') {
    return (
      <span className={styles.saveError} role="status">
        Não salvo (sem conexão?).{' '}
        <button type="button" className="btn btn-ghost" onClick={onRetry}>Tentar de novo</button>
      </span>
    )
  }
  let text = 'Salvamento automático'
  if (state.kind === 'saving') text = 'Salvando…'
  else if (dirty) text = 'Alterações não salvas…'
  else if (state.kind === 'saved') {
    text = `Salvo às ${state.at.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })}`
  }
  return <span className={styles.saveStatus} role="status">{text}</span>
}

function RecognizedPanel({ date, content, journal, saveNow }: {
  date: string
  content: string
  journal: Journal
  saveNow: () => Promise<boolean>
}) {
  const habits = useHabits()
  const register = useRegisterJournalItem(date)
  const unregister = useUnregisterJournalItem(date)
  const [errors, setErrors] = useState<Record<string, string>>({})

  const recognized = useMemo(
    () => parseNoteContent(content).filter(l => lineKey(l.originalText) && (l.transaction || l.health)),
    [content]
  )
  const itemsById = useMemo(() => new Map(journal.items.map(i => [i.source_id, i])), [journal.items])
  const currentIds = useMemo(() => new Set(Object.values(journal.line_ids)), [journal.line_ids])
  // Registrado a partir de uma linha que foi editada ou apagada do texto.
  const orphans = journal.items.filter(i => !currentIds.has(i.source_id))

  const registeredFor = (line: ParsedLine): JournalItem | undefined => {
    const id = journal.line_ids[lineKey(line.originalText)]
    return id ? itemsById.get(id) : undefined
  }

  const setError = (key: string, message?: string) =>
    setErrors(prev => {
      const next = { ...prev }
      if (message) next[key] = message
      else delete next[key]
      return next
    })

  const handleRegister = async (line: ParsedLine) => {
    const key = lineKey(line.originalText)
    setError(key)
    // A etiqueta vem do texto salvo: salva antes de registrar.
    if (!(await saveNow())) {
      setError(key, 'Salve o texto primeiro (sem conexão?).')
      return
    }
    try {
      if (line.transaction) {
        await register.mutateAsync({
          line: line.originalText,
          kind: 'transaction',
          expense: {
            amount_cents: line.transaction.amountCents,
            category: line.transaction.financeCategory,
            description: line.transaction.description || 'Registro do diário',
          },
        })
      } else if (line.health) {
        const habit = resolveHabitForWorkout(line.health.type, habits.data ?? [])
        if (!habit) return
        const metrics: Record<string, number> = {}
        if (line.health.distance != null) metrics.km = line.health.distance
        if (line.health.time != null) metrics.time_min = line.health.time
        if (line.health.rpe != null) metrics.rpe = line.health.rpe
        await register.mutateAsync({
          line: line.originalText,
          kind: 'workout',
          workout: { habit_id: habit.id, metrics, time_minutes: line.health.time },
        })
      }
    } catch (err) {
      setError(key, registerErrorMessage(err))
    }
  }

  const handleUndo = async (sourceId: string, key: string) => {
    setError(key)
    try {
      await unregister.mutateAsync(sourceId)
    } catch {
      setError(key, 'Não foi possível desfazer. Tente de novo.')
    }
  }

  return (
    <section className={`glass-card ${styles.panel}`} aria-label="Reconhecido no texto">
      <h2 className={styles.panelTitle}>Reconhecido no texto</h2>

      {recognized.length === 0 && (
        <p className={styles.hint}>
          Escreva um gasto (<code>$ 50 Almoço #alimentacao</code>) ou um treino
          (<code>corrida 5km 30min</code>) e ele aparece aqui para você registrar.
        </p>
      )}

      {recognized.length > 0 && (
        <ul className={styles.items}>
          {recognized.map((line, i) => {
            const key = lineKey(line.originalText)
            const done = registeredFor(line)
            const habit = line.health ? resolveHabitForWorkout(line.health.type, habits.data ?? []) : null
            return (
              <li key={`${i}-${key}`} className={styles.item}>
                <div className={styles.itemMain}>
                  {line.transaction ? (
                    <>
                      <span className={styles.itemTitle}>
                        {categoryIcon(line.transaction.financeCategory)} {formatCents(line.transaction.amountCents)} · {line.transaction.description}
                      </span>
                      <span className={styles.itemMeta}>Gasto · {categoryLabel(line.transaction.financeCategory)}</span>
                    </>
                  ) : line.health ? (
                    <>
                      <span className={styles.itemTitle}>
                        {WORKOUT_ICONS[line.health.type]} {WORKOUT_NAMES[line.health.type]}
                        {line.health.distance != null && ` · ${line.health.distance.toLocaleString('pt-BR')} km`}
                        {line.health.time != null && ` · ${line.health.time} min`}
                        {line.health.rpe != null && ` · RPE ${line.health.rpe}`}
                      </span>
                      <span className={styles.itemMeta}>
                        {habit ? `Treino → hábito ${habit.name}` : (
                          <>Sem hábito de Saúde para isso. <Link to="/habits">Crie um</Link> com “{WORKOUT_NAMES[line.health.type]}” no nome.</>
                        )}
                      </span>
                    </>
                  ) : null}
                  {errors[key] && <span className={styles.itemError} role="alert">{errors[key]}</span>}
                </div>
                <div className={styles.itemActions}>
                  {done ? (
                    <>
                      <span className={styles.registered}>✓ Registrado</span>
                      <button type="button" className="btn btn-ghost" onClick={() => handleUndo(done.source_id, key)}>
                        Desfazer
                      </button>
                    </>
                  ) : (
                    <button
                      type="button"
                      className="btn btn-primary"
                      disabled={register.isPending || (!!line.health && !habit)}
                      onClick={() => { void handleRegister(line) }}
                    >
                      Registrar
                    </button>
                  )}
                </div>
              </li>
            )
          })}
        </ul>
      )}

      {orphans.length > 0 && (
        <>
          <h3 className={styles.subTitle}>Registrado, mas não está mais no texto</h3>
          <ul className={styles.items}>
            {orphans.map(item => (
              <li key={item.source_id} className={styles.item}>
                <div className={styles.itemMain}>
                  <span className={styles.itemTitle}>
                    {item.kind === 'transaction'
                      ? `💸 ${formatCents(item.amount_cents ?? 0)} · ${item.label}`
                      : `🏃 ${item.label}`}
                  </span>
                  <span className={styles.itemMeta}>A linha foi editada ou apagada; o registro continua valendo.</span>
                  {errors[item.source_id] && <span className={styles.itemError} role="alert">{errors[item.source_id]}</span>}
                </div>
                <div className={styles.itemActions}>
                  <button type="button" className="btn btn-ghost" onClick={() => handleUndo(item.source_id, item.source_id)}>
                    Desfazer
                  </button>
                </div>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  )
}
