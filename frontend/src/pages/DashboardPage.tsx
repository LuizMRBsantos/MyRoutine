import { Link, useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import { useHabits } from '@/hooks/useHabits'
import { useConsistency } from '@/hooks/useConsistency'
import { TodayAgenda } from '@/components/dashboard/TodayAgenda'
import { TodayHabits } from '@/components/dashboard/TodayHabits'
import { WeeklyGoals } from '@/components/dashboard/WeeklyGoals'
import { DashboardCalendarWidget } from '@/components/dashboard/DashboardCalendarWidget'
import { WeekInOneLine } from '@/components/dashboard/WeekInOneLine'
import { HabitConsistency } from '@/components/dashboard/HabitConsistency'
import styles from './DashboardPage.module.css'

function greeting(hour: number): string {
  if (hour < 12) return 'Bom dia'
  if (hour < 18) return 'Boa tarde'
  return 'Boa noite'
}

// Início: o "Hoje" manda (agenda + hábitos lado a lado), a semana vem logo
// depois e a constância fica no fim, para olhar para trás. Sem anel de %,
// chama ou barra de progresso (product-constitution; dashboard-widget-design).
export function DashboardPage() {
  const { user } = useAuthStore()
  const navigate = useNavigate()
  const { data: habits = [], isLoading, isError } = useHabits()
  const { data: consistency } = useConsistency()
  const hasHabits = (consistency?.habits.length ?? 0) > 0

  const now = new Date()
  const firstName = user?.name?.split(' ')[0]
  const today = now.toLocaleDateString('pt-BR', { weekday: 'long', day: 'numeric', month: 'long' })

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div className={styles.titles}>
          <p className={styles.date}>{today}</p>
          <h1 className={styles.greeting}>{greeting(now.getHours())}{firstName ? `, ${firstName}` : ''}.</h1>
        </div>
        <div className={styles.actions}>
          <button type="button" className="btn btn-secondary" onClick={() => navigate('/diario')}>Escrever no diário</button>
          <button type="button" className="btn btn-primary" onClick={() => navigate('/planner')}>Nova tarefa</button>
        </div>
      </header>

      <div className={styles.today}>
        <section className={`glass-card ${styles.card}`} aria-labelledby="inicio-agenda">
          <div className={styles.cardHeader}>
            <h2 id="inicio-agenda" className={styles.cardTitle}>Agenda de hoje</h2>
            <Link to="/planner" className={styles.cardLink}>Planner</Link>
          </div>
          <TodayAgenda />
        </section>

        <section className={`glass-card ${styles.card}`} aria-labelledby="inicio-habitos">
          <div className={styles.cardHeader}>
            <h2 id="inicio-habitos" className={styles.cardTitle}>Hábitos de hoje</h2>
            <Link to="/habits" className={styles.cardLink}>Todos</Link>
          </div>
          <TodayHabits habits={habits} isLoading={isLoading} isError={isError} />
        </section>

        <WeeklyGoals />
      </div>

      <DashboardCalendarWidget />

      {hasHabits && (
        <section className={`glass-card ${styles.card}`} aria-labelledby="inicio-semana">
          <h2 id="inicio-semana" className={styles.cardTitle}>A semana em uma linha</h2>
          <WeekInOneLine />
        </section>
      )}

      {hasHabits && (
        <section className={`glass-card ${styles.card}`} aria-labelledby="inicio-constancia">
          <div className={styles.cardHeader}>
            <h2 id="inicio-constancia" className={styles.cardTitle}>Constância</h2>
            <span className={styles.cardHint}>últimos 90 dias, por hábito</span>
          </div>
          <HabitConsistency />
        </section>
      )}
    </div>
  )
}
