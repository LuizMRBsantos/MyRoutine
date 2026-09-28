import { useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import { refreshSession } from '@/services/refreshSession'

interface ProtectedRouteProps {
  children: React.ReactNode
}

// Após um reload, o access token não é persistido (só o refresh token).
// Este componente troca o refresh token por um access token novo antes de
// renderizar conteúdo protegido — sem flash de conteúdo nem de login.
export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)

  // Decidido uma única vez, na montagem: o refresh rotaciona o token e grava
  // o access token no store, então derivar isto do estado faria a condição
  // mudar no meio da própria troca.
  const [refreshing, setRefreshing] = useState(() => {
    const { isAuthenticated, accessToken, refreshToken } = useAuthStore.getState()
    return isAuthenticated && !accessToken && !!refreshToken
  })

  useEffect(() => {
    if (!refreshing) return

    // O efeito roda uma vez só (deps vazias), então este cleanup só dispara
    // na desmontagem de verdade — nunca no meio da requisição.
    let mounted = true

    // Mesma renovação compartilhada do interceptor do `api`: se uma request
    // receber 401 enquanto esta troca está em andamento, ela espera esta
    // Promise em vez de reapresentar o mesmo refresh token (o que o backend
    // trata como roubo). Em falha, refreshSession já faz o logout.
    refreshSession()
      .catch(() => {})
      .finally(() => {
        if (mounted) setRefreshing(false)
      })

    return () => { mounted = false }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

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

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}
