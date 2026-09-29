import { motion } from 'framer-motion'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import { useHabits, useHabitStats, useHeatmap } from '@/hooks/useHabits'
import { TodayHabits } from '@/components/dashboard/TodayHabits'
import { HabitHeatmap } from '@/components/habits/HabitHeatmap'
import { DashboardCalendarWidget } from '@/components/dashboard/DashboardCalendarWidget'
import styles from './DashboardPage.module.css'

const MODULES = [
  { icon: '◈', label: 'Finanças', desc: 'Controle seus gastos', color: '#34C759', to: '/finance' },
  { icon: '◉', label: 'Saúde', desc: 'Treinos e métricas', color: '#FF9F0A', to: '/health' },
  { icon: '◆', label: 'Estudos', desc: 'Aprendizado ativo', color: '#AF52DE', to: '/studies' },
]

export function DashboardPage() {
  const { user } = useAuthStore()
  const navigate = useNavigate()
  const { data: habits = [], isLoading, isError } = useHabits()
  const { data: stats } = useHabitStats()
  const { data: heatmapData = [] } = useHeatmap()

  const getGreeting = () => {
    const hour = new Date().getHours()
    if (hour < 12) return 'Bom dia'
    if (hour < 18) return 'Boa tarde'
    return 'Boa noite'
  }

  const today = new Date().toLocaleDateString('pt-BR', {
    weekday: 'long', day: 'numeric', month: 'long',
  })

  const completedToday = stats?.completed_today ?? 0
  const totalHabits = stats?.total_habits ?? 0
  const completionPct = totalHabits > 0
    ? Math.round((completedToday / totalHabits) * 100)
    : 0

  return (
    <div className={styles.page}>
      {/* ─── Header ─────────────────────────────────── */}
      <motion.div
        className={styles.header}
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
      >
        <div>
          <p className={styles.greeting}>{getGreeting()},</p>
          <h1 className={styles.userName}>{user?.name?.split(' ')[0] ?? 'você'} ✦</h1>
          <p className={styles.date}>{today}</p>
        </div>

        {/* Day completion ring */}
        {totalHabits > 0 && (
          <div className={styles.ringWrapper} title={`${completedToday} de ${totalHabits} hábitos`}>
            <svg viewBox="0 0 44 44" className={styles.ring}>
              <circle cx="22" cy="22" r="18" className={styles.ringTrack} />
              <circle
                cx="22" cy="22" r="18"
                className={styles.ringFill}
                strokeDasharray={`${2 * Math.PI * 18}`}
                strokeDashoffset={`${2 * Math.PI * 18 * (1 - completionPct / 100)}`}
              />
            </svg>
            <span className={styles.ringLabel}>{completionPct}%</span>
          </div>
        )}
      </motion.div>

      <div className={styles.grid}>
        {/* ─── Today's Habits ─────────────────────────── */}
        <motion.div
          className={`glass-card ${styles.card} ${styles.habitsCard}`}
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1, duration: 0.4 }}
        >
          <div className={styles.cardHeader}>
            <h2 className={styles.cardTitle}>Hábitos de Hoje</h2>
            <button
              className={`btn btn-ghost ${styles.seeAll}`}
              onClick={() => navigate('/habits')}
            >
              Ver todos →
            </button>
          </div>
          <TodayHabits habits={habits} isLoading={isLoading} isError={isError} />
        </motion.div>

        {/* ─── Modules ──────────────────────────── */}
        <div className={styles.modulesGrid}>
          {MODULES.map((mod, i) => (
            <motion.div
              key={mod.label}
              className={`glass-card ${styles.moduleCard}`}
              style={{ cursor: 'pointer' }}
              onClick={() => navigate(mod.to)}
              initial={{ opacity: 0, y: 16 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.15 + i * 0.06, duration: 0.4 }}
            >
              <div
                className={styles.moduleIcon}
                style={{ background: mod.color + '18', color: mod.color }}
              >
                {mod.icon}
              </div>
              <div className={styles.moduleInfo}>
                <h3 className={styles.moduleLabel}>{mod.label}</h3>
                <p className={styles.moduleDesc}>{mod.desc}</p>
              </div>
            </motion.div>
          ))}
        </div>
      </div>

      {/* ─── Calendário Interativo Google Calendar (Semana/Mês) ────── */}
      <DashboardCalendarWidget />

      {/* ─── Consistência Heatmap ────────────────────── */}
      {heatmapData.length > 0 && (
        <motion.div
          className={`glass-card ${styles.heatmapCard}`}
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.25, duration: 0.4 }}
        >
          <h2 className={styles.cardTitle}>Consistência — 90 dias</h2>
          <HabitHeatmap data={heatmapData} days={90} />
        </motion.div>
      )}
    </div>
  )
}
