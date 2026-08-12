import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useMissedDays, useReviewDay } from '@/hooks/useWeeklyReview'
import type { MissedDay } from '@/hooks/useWeeklyReview'
import styles from './WeeklyReview.module.css'

/**
 * ESTACIONADO — não está montado em nenhuma tela por decisão de produto.
 *
 * Este componente anunciava sozinho "N dias sem registro esperando revisão"
 * na página de Hábitos. Isso contraria a skill habit-review-flow, que pede
 * uma revisão semanal opt-in na navegação e proíbe contador de dias falhados
 * em destaque — um dia sem log é estado neutro, não cobrança.
 *
 * O backend continua inteiro (habit_day_reviews, POST /habits/{id}/review,
 * GET /reviews/missed) e os hooks abaixo seguem funcionando. Para trazer a
 * revisão de volta, monte <WeeklyReview /> numa rota própria acessada por
 * escolha do usuário — nunca embutida numa tela que ele abre por outro
 * motivo.
 */
export function WeeklyReview() {
  const [open, setOpen] = useState(false)
  const { data: missedDays = [], isLoading } = useMissedDays()
  const reviewDay = useReviewDay()

  // Only show button if there are unreviewed missed days
  const unreviewed = missedDays.filter(d => !d.review)
  const hasUnreviewed = unreviewed.length > 0

  if (!hasUnreviewed && !open) return null

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr + 'T12:00:00')
    return d.toLocaleDateString('pt-BR', { weekday: 'short', day: 'numeric', month: 'short' })
  }

  const handleReview = (day: MissedDay, status: 'migrated' | 'discarded') => {
    reviewDay.mutate({ habitId: day.habit_id, reviewDate: day.date, status })
  }

  return (
    <>
      {/* Trigger banner */}
      {!open && (
        <motion.button
          className={styles.banner}
          onClick={() => setOpen(true)}
          initial={{ opacity: 0, y: -8 }}
          animate={{ opacity: 1, y: 0 }}
          id="weekly-review-btn"
        >
          <span className={styles.bannerIcon}>↩</span>
          <span className={styles.bannerText}>
            <strong>{unreviewed.length} dia{unreviewed.length !== 1 ? 's' : ''}</strong> sem registro esperando revisão
          </span>
          <span className={styles.bannerAction}>Revisar →</span>
        </motion.button>
      )}

      {/* Review panel */}
      <AnimatePresence>
        {open && (
          <motion.div
            className={styles.panel}
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.3 }}
          >
            <div className={styles.panelHeader}>
              <div>
                <h2 className={styles.panelTitle}>Revisão da Semana</h2>
                <p className={styles.panelSubtitle}>
                  Decida conscientemente o que fazer com os dias sem registro.
                </p>
              </div>
              <button className={styles.closeBtn} onClick={() => setOpen(false)}>×</button>
            </div>

            {isLoading ? (
              <p className={styles.loading}>Carregando...</p>
            ) : missedDays.length === 0 ? (
              <p className={styles.allGood}>✦ Sem dias para revisar — ótima semana!</p>
            ) : (
              <div className={styles.list}>
                {missedDays.map((day) => (
                  <div
                    key={`${day.habit_id}-${day.date}`}
                    className={`${styles.item} ${day.review ? styles.itemReviewed : ''}`}
                  >
                    {/* Habit info */}
                    <div
                      className={styles.itemIcon}
                      style={{ background: day.habit_color + '18', color: day.habit_color }}
                    >
                      {day.habit_icon}
                    </div>
                    <div className={styles.itemInfo}>
                      <span className={styles.itemName}>{day.habit_name}</span>
                      <span className={styles.itemDate}>{formatDate(day.date)}</span>
                    </div>

                    {/* Actions or review status */}
                    {day.review ? (
                      <span className={`${styles.reviewedBadge} ${styles[day.review.status]}`}>
                        {day.review.status === 'migrated' ? '→ Migrado' : '· Descartado'}
                      </span>
                    ) : (
                      <div className={styles.actions}>
                        <button
                          className={styles.migrateBtn}
                          onClick={() => handleReview(day, 'migrated')}
                          title="Migrar para próxima semana"
                          disabled={reviewDay.isPending}
                        >
                          → Migrar
                        </button>
                        <button
                          className={styles.discardBtn}
                          onClick={() => handleReview(day, 'discarded')}
                          title="Descartar conscientemente"
                          disabled={reviewDay.isPending}
                        >
                          · Descartar
                        </button>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}

            {unreviewed.length === 0 && missedDays.length > 0 && (
              <div className={styles.doneRow}>
                <span className={styles.doneText}>✓ Todos os dias foram revisados</span>
                <button className="btn btn-ghost" onClick={() => setOpen(false)}>
                  Fechar
                </button>
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </>
  )
}
