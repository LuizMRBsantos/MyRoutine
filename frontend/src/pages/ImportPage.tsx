import { useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import {
  usePreviewImport, useCreateImport, usePendingEntries,
  useApproveEntry, useDecideEntry, useCategoryRules, useDeleteRule,
} from '@/hooks/useImports'
import { useCards } from '@/hooks/useFinance'
import { FINANCE_CATEGORIES, categoryLabel, formatCents } from '@/types/finance'
import type { CSVMapping, ImportEntry } from '@/types/import'
import { toast } from '@/lib/toast'
import styles from './ImportPage.module.css'

const EXPENSE_CATEGORIES = Object.entries(FINANCE_CATEGORIES).filter(([k]) => k !== 'salario')

function formatDate(iso: string) {
  return new Date(iso + 'T12:00').toLocaleDateString('pt-BR', { day: '2-digit', month: 'short' })
}

export function ImportPage() {
  const preview = usePreviewImport()
  const createImport = useCreateImport()
  const { data: pending = [], isLoading } = usePendingEntries()
  const { data: rules = [] } = useCategoryRules()
  const { data: cards = [] } = useCards()
  const approve = useApproveEntry()
  const decide = useDecideEntry()
  const deleteRule = useDeleteRule()

  const [content, setContent] = useState('')
  const [filename, setFilename] = useState('')
  const [mapping, setMapping] = useState<CSVMapping | null>(null)
  const [cardId, setCardId] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)
  // Categoria escolhida por linha, iniciando na sugestão do servidor
  const [categories, setCategories] = useState<Record<string, string>>({})

  const handleFile = async (file: File) => {
    const text = await file.text()
    setContent(text)
    setFilename(file.name)
    preview.mutate(text, {
      onSuccess: (result) => {
        setMapping(result.mapping)
        if (!result.complete) {
          toast.info('Não reconheci todas as colunas — confira o mapeamento abaixo.')
        }
      },
      onError: () => toast.error('Não consegui ler esse arquivo'),
    })
  }

  const handleImport = () => {
    if (!mapping) return
    createImport.mutate(
      { filename, content, mapping, credit_card_id: cardId || undefined },
      {
        onSuccess: (batch) => {
          setContent('')
          setMapping(null)
          // Sem limpar o valor, escolher o mesmo arquivo de novo não dispara
          // o onChange e a tela fica inerte.
          if (fileInputRef.current) fileInputRef.current.value = ''
          const parts = [`${batch.new_count} novo(s)`]
          if (batch.duplicate_count) parts.push(`${batch.duplicate_count} já importado(s)`)
          if (batch.matched_count) parts.push(`${batch.matched_count} possível duplicata`)
          toast.success(parts.join(' · '))
        },
      }
    )
  }

  const categoryFor = (entry: ImportEntry) =>
    categories[entry.id] ?? entry.suggested_category ?? 'other'

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <div>
          <h1 className={styles.pageTitle}>Importar extrato</h1>
          <p className={styles.pageSubtitle}>
            Nada entra nas suas contas antes de você aprovar linha a linha.
          </p>
        </div>
        <Link to="/finance" className="btn btn-ghost">← Finanças</Link>
      </div>

      {/* ── Upload ── */}
      <section className={`glass-card ${styles.section}`}>
        <h2 className={styles.sectionTitle}>1. Escolher arquivo</h2>
        <input
          ref={fileInputRef}
          type="file"
          accept=".csv,text/csv,text/plain"
          className={styles.fileInput}
          aria-label="Arquivo do extrato"
          onChange={(e) => {
            const file = e.target.files?.[0]
            if (file) handleFile(file)
          }}
        />
        <p className={styles.hint}>
          Exporte o extrato em CSV pelo app do banco ou do cartão. Eu identifico as
          colunas sozinho na maioria dos casos.
        </p>

        {mapping && (
          <div className={styles.mappingBox}>
            <h3 className={styles.mappingTitle}>2. Conferir as colunas</h3>
            <div className={styles.mappingGrid}>
              {([
                ['Data', 'date_column'],
                ['Valor', 'amount_column'],
                ['Descrição', 'description_column'],
              ] as const).map(([label, field]) => (
                <label key={field} className={styles.mappingField}>
                  <span className={styles.mappingLabel}>{label}</span>
                  <input
                    type="number"
                    min="0"
                    aria-label={`Coluna de ${label}`}
                    className="input"
                    value={mapping[field]}
                    onChange={(e) =>
                      setMapping({ ...mapping, [field]: Number(e.target.value) })
                    }
                  />
                </label>
              ))}
            </div>

            <label className={styles.checkboxRow}>
              <input
                type="checkbox"
                checked={mapping.negative_is_expense}
                onChange={(e) =>
                  setMapping({ ...mapping, negative_is_expense: e.target.checked })
                }
              />
              <span>Valor negativo significa despesa</span>
            </label>

            {cards.length > 0 && (
              <label className={styles.checkboxRow}>
                <select
                  aria-label="Cartão do extrato"
                  className="input"
                  value={cardId}
                  onChange={(e) => setCardId(e.target.value)}
                >
                  <option value="">Extrato de conta (sem cartão)</option>
                  {cards.map((c) => (
                    <option key={c.id} value={c.id}>💳 {c.name}</option>
                  ))}
                </select>
              </label>
            )}

            {preview.data && preview.data.sample.length > 0 && (
              <div className={styles.sample}>
                <p className={styles.sampleTitle}>Primeiras linhas lidas assim:</p>
                {preview.data.sample.map((s, i) => (
                  <p key={i} className={styles.sampleRow}>
                    {formatDate(s.occurred_on)} · {s.raw_description} ·{' '}
                    <strong>{s.kind === 'income' ? '+' : '−'}{formatCents(s.amount_cents)}</strong>
                  </p>
                ))}
              </div>
            )}

            <button
              className="btn btn-primary"
              onClick={handleImport}
              disabled={createImport.isPending}
            >
              {createImport.isPending ? 'Importando…' : 'Importar para revisão'}
            </button>
          </div>
        )}
      </section>

      {/* ── Reconciliation inbox ── */}
      <section className={`glass-card ${styles.section}`}>
        <div className={styles.inboxHeader}>
          <h2 className={styles.sectionTitle}>Para revisar</h2>
          {pending.length > 0 && (
            <span className={styles.counter}>{pending.length} aguardando</span>
          )}
        </div>

        {isLoading ? (
          <p className={styles.hint}>Carregando…</p>
        ) : pending.length === 0 ? (
          <p className={styles.hint}>
            Nada para revisar. Importe um extrato acima para começar.
          </p>
        ) : (
          <ul className={styles.entryList}>
            <AnimatePresence>
              {pending.map((entry) => (
                <motion.li
                  key={entry.id}
                  className={styles.entry}
                  initial={{ opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, height: 0, marginBottom: 0 }}
                >
                  <div className={styles.entryMain}>
                    <div className={styles.entryInfo}>
                      <span className={styles.entryDescription}>{entry.raw_description}</span>
                      <span className={styles.entryMeta}>
                        {formatDate(entry.occurred_on)}
                        {entry.suggested_category && ' · categoria sugerida'}
                      </span>
                    </div>
                    <span className={`${styles.entryAmount} ${entry.kind === 'income' ? styles.income : ''}`}>
                      {entry.kind === 'income' ? '+' : '−'}{formatCents(entry.amount_cents)}
                    </span>
                  </div>

                  {/* Possível duplicata: mostra lado a lado antes de decidir */}
                  {entry.matched_transaction_id && (
                    <div className={styles.matchWarning}>
                      Parece o seu lançamento{' '}
                      <strong>{entry.matched_transaction_description}</strong>
                      {entry.matched_transaction_date && ` de ${formatDate(entry.matched_transaction_date)}`}.
                      Se for o mesmo gasto, marque como duplicata para não contar duas vezes.
                    </div>
                  )}

                  <div className={styles.entryActions}>
                    <select
                      aria-label={`Categoria de ${entry.raw_description}`}
                      className={`input ${styles.categorySelect}`}
                      value={categoryFor(entry)}
                      onChange={(e) =>
                        setCategories((c) => ({ ...c, [entry.id]: e.target.value }))
                      }
                    >
                      {EXPENSE_CATEGORIES.map(([key, meta]) => (
                        <option key={key} value={key}>{meta.icon} {meta.label}</option>
                      ))}
                      <option value="salario">💼 Salário</option>
                    </select>

                    <button
                      className="btn btn-primary"
                      disabled={approve.isPending}
                      onClick={() =>
                        approve.mutate(
                          { id: entry.id, category: categoryFor(entry) },
                          { onSuccess: () => toast.success('Lançado') }
                        )
                      }
                    >
                      Aprovar
                    </button>
                    <button
                      className="btn btn-ghost"
                      onClick={() => decide.mutate({ id: entry.id, status: 'matched' })}
                      title="Já existe nas minhas transações"
                    >
                      É duplicata
                    </button>
                    <button
                      className={styles.ignoreBtn}
                      onClick={() => decide.mutate({ id: entry.id, status: 'ignored' })}
                      title="Não quero registrar isso"
                    >
                      Ignorar
                    </button>
                  </div>
                </motion.li>
              ))}
            </AnimatePresence>
          </ul>
        )}
      </section>

      {/* ── Learned rules ── */}
      {rules.length > 0 && (
        <section className={`glass-card ${styles.section}`}>
          <h2 className={styles.sectionTitle}>Regras aprendidas</h2>
          <p className={styles.hint}>
            Cada vez que você categoriza, eu guardo o padrão. Na próxima importação
            a sugestão já vem pronta.
          </p>
          <ul className={styles.ruleList}>
            {rules.map((rule) => (
              <li key={rule.id} className={styles.rule}>
                <code className={styles.rulePattern}>{rule.pattern}</code>
                <span className={styles.ruleArrow}>→</span>
                <span className={styles.ruleCategory}>{categoryLabel(rule.category)}</span>
                <button
                  className={styles.removeBtn}
                  title="Esquecer regra"
                  onClick={() => deleteRule.mutate(rule.id)}
                >
                  ×
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  )
}
