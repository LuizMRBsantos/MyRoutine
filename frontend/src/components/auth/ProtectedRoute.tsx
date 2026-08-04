import { useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'
import axios from 'axios'
import { useAuthStore } from '@/store/authStore'

interface ProtectedRouteProps {
  children: React.ReactNode
}

// Após um reload, o access token não é persistido (só o refresh token).
// Este componente troca o refresh token por um access token novo antes de
// renderizar conteúdo protegido — sem flash de conteúdo nem de login.
export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const { isAuthenticated, accessToken, refreshToken, setAuth, logout } = useAuthStore()
  const needsRefresh = isAuthenticated && !accessToken && !!refreshToken
  const [refreshing, setRefreshing] = useState(needsRefresh)

  useEffect(() => {
    if (!needsRefresh) return
    let cancelled = false

    axios
      .post('/api/v1/auth/refresh', { refresh_token: refreshToken })
      .then(({ data }) => {
        if (cancelled) return
        setAuth(
          { accessToken: data.access_token, refreshToken: data.refresh_token },
          data.user
        )
      })
      .catch(() => {
        if (!cancelled) logout()
      })
      .finally(() => {
        if (!cancelled) setRefreshing(false)
      })

    return () => { cancelled = true }
  }, [needsRefresh, refreshToken, setAuth, logout])

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  if (refreshing) {
    return (
      <div style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: 'var(--color-text-tertiary)',
        fontSize: 'var(--font-size-sm)',
      }}>
        Carregando…
      </div>
    )
  }

  return <>{children}</>
}
