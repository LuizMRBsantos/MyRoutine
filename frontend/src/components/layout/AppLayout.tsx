import { NavLink, Outlet } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useAuthStore } from '@/store/authStore'
import api from '@/services/api'
import styles from './AppLayout.module.css'

const navItems = [
  { to: '/', icon: '⊞', label: 'Dashboard' },
  { to: '/habits', icon: '✦', label: 'Hábitos' },
  { to: '/planner', icon: '📅', label: 'Planner' },
  { to: '/finance', icon: '◈', label: 'Finanças' },
  { to: '/health', icon: '◉', label: 'Saúde' },
  { to: '/studies', icon: '◆', label: 'Estudos' },
]

export function AppLayout() {
  const { user, refreshToken, logout } = useAuthStore()

  const handleLogout = async () => {
    // Revoga o refresh token no servidor antes de limpar o estado local.
    try {
      if (refreshToken) {
        await api.post('/auth/logout', { refresh_token: refreshToken })
      }
    } catch {
      // Logout local acontece de qualquer forma
    }
    logout()
  }

  return (
    <div className={styles.layout}>
      {/* ─── Sidebar ─────────────────────────────────── */}
      <aside className={styles.sidebar}>
        {/* Logo */}
        <div className={styles.logo}>
          <div className={styles.logoMark}>M</div>
          <span className={styles.logoText}>MyRoutine</span>
        </div>

        {/* Navigation */}
        <nav className={styles.nav}>
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) =>
                `${styles.navItem} ${isActive ? styles.navItemActive : ''}`
              }
            >
              <span className={styles.navIcon}>{item.icon}</span>
              <span className={styles.navLabel}>{item.label}</span>
            </NavLink>
          ))}
        </nav>

        {/* User profile */}
        <div className={styles.sidebarFooter}>
          <div className={styles.userInfo}>
            <div className={styles.avatar}>
              {user?.name?.charAt(0).toUpperCase() ?? 'U'}
            </div>
            <div className={styles.userMeta}>
              <span className={styles.userName}>{user?.name}</span>
              <span className={styles.userEmail}>{user?.email}</span>
            </div>
          </div>
          <button className={styles.logoutBtn} onClick={handleLogout} title="Sair">
            ⎋
          </button>
        </div>
      </aside>

      {/* ─── Main ───────────────────────────────────── */}
      <motion.main
        className={styles.main}
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: [0.4, 0, 0.2, 1] }}
      >
        <Outlet />
      </motion.main>
    </div>
  )
}
