import { useEffect, useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { motion } from 'framer-motion'
import { useAuthStore } from '@/store/authStore'
import api from '@/services/api'
import styles from './AppLayout.module.css'

interface NavItem {
  to: string
  icon: string
  label: string
  // Rótulo da barra inferior do celular, onde o espaço é curto.
  short?: string
}

const navItems: NavItem[] = [
  { to: '/', icon: '⊞', label: 'Dashboard', short: 'Início' },
  // O diário é a principal porta de entrada de dados: fica logo no começo.
  { to: '/diario', icon: '✎', label: 'Diário' },
  { to: '/habits', icon: '✦', label: 'Hábitos' },
  { to: '/planner', icon: '📅', label: 'Planner' },
  { to: '/finance', icon: '◈', label: 'Finanças' },
  { to: '/health', icon: '◉', label: 'Saúde' },
  { to: '/studies', icon: '◆', label: 'Estudos' },
  // Revisão semanal: só por escolha, sem contador (habit-review-flow).
  { to: '/revisao', icon: '↺', label: 'Revisão' },
  { to: '/conta', icon: '⚙', label: 'Minha conta' },
]

// Só aparece para administradores (convites e links de redefinição de senha).
const adminNavItems: NavItem[] = [
  { to: '/acessos', icon: '✉', label: 'Acessos' },
]

// No celular, a barra inferior mostra os 4 módulos mais usados + "Mais";
// o resto (e o Sair) fica na folha que o "Mais" abre.
const MOBILE_PRIMARY_COUNT = 4

export function AppLayout() {
  const { user, refreshToken, logout } = useAuthStore()
  const visibleNavItems = user?.is_admin ? [...navItems, ...adminNavItems] : navItems
  const mobilePrimary = visibleNavItems.slice(0, MOBILE_PRIMARY_COUNT)
  const mobileMore = visibleNavItems.slice(MOBILE_PRIMARY_COUNT)

  const location = useLocation()
  const [moreOpen, setMoreOpen] = useState(false)
  const moreActive = mobileMore.some(item => location.pathname.startsWith(item.to))

  // Fecha a folha "Mais" ao navegar e com Esc.
  useEffect(() => setMoreOpen(false), [location.pathname])
  useEffect(() => {
    if (!moreOpen) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setMoreOpen(false)
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [moreOpen])

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
          {visibleNavItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              title={item.label}
              className={({ isActive }) =>
                `${styles.navItem} ${isActive ? styles.navItemActive : ''}`
              }
            >
              <span className={styles.navIcon} aria-hidden="true">{item.icon}</span>
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
          <button className={styles.logoutBtn} onClick={handleLogout} title="Sair" aria-label="Sair">
            ⎋
          </button>
        </div>
      </aside>

      {/* ─── Barra inferior (celular) ─────────────────── */}
      <nav className={styles.tabBar} aria-label="Navegação principal">
        {mobilePrimary.map(item => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            className={({ isActive }) => `${styles.tab} ${isActive ? styles.tabActive : ''}`}
          >
            <span className={styles.tabIcon} aria-hidden="true">{item.icon}</span>
            <span className={styles.tabLabel}>{item.short ?? item.label}</span>
          </NavLink>
        ))}
        <button
          type="button"
          className={`${styles.tab} ${moreActive || moreOpen ? styles.tabActive : ''}`}
          aria-expanded={moreOpen}
          aria-controls="more-sheet"
          onClick={() => setMoreOpen(open => !open)}
        >
          <span className={styles.tabIcon} aria-hidden="true">⋯</span>
          <span className={styles.tabLabel}>Mais</span>
        </button>
      </nav>

      {moreOpen && (
        <>
          <div className={styles.sheetBackdrop} onClick={() => setMoreOpen(false)} />
          <div id="more-sheet" className={styles.sheet} role="dialog" aria-label="Mais opções">
            <div className={styles.sheetUser}>
              <div className={styles.avatar}>{user?.name?.charAt(0).toUpperCase() ?? 'U'}</div>
              <div className={styles.userMeta}>
                <span className={styles.userName}>{user?.name}</span>
                <span className={styles.userEmail}>{user?.email}</span>
              </div>
            </div>
            {mobileMore.map(item => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) => `${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
              >
                <span className={styles.navIcon} aria-hidden="true">{item.icon}</span>
                <span className={styles.navLabel}>{item.label}</span>
              </NavLink>
            ))}
            <button type="button" className={`${styles.navItem} ${styles.sheetLogout}`} onClick={handleLogout}>
              <span className={styles.navIcon} aria-hidden="true">⎋</span>
              <span className={styles.navLabel}>Sair</span>
            </button>
          </div>
        </>
      )}

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
