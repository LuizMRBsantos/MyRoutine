import axios from 'axios'
import { useAuthStore } from '@/store/authStore'

// O backend trata um refresh token apresentado duas vezes como roubo e revoga
// TODAS as sessões do usuário. Por isso toda renovação passa por aqui: se já
// existe uma em andamento, quem chegar depois recebe a mesma Promise em vez de
// disparar outro POST /auth/refresh com o mesmo token.
let inFlight: Promise<string> | null = null

async function doRefresh(): Promise<string> {
  const { refreshToken, setAuth, logout } = useAuthStore.getState()

  if (!refreshToken) {
    logout()
    throw new Error('sessão sem refresh token')
  }

  try {
    // axios puro (fora dos interceptors do `api`): um 401 aqui não pode
    // disparar outra renovação.
    const { data } = await axios.post('/api/v1/auth/refresh', {
      refresh_token: refreshToken,
    })
    setAuth(
      { accessToken: data.access_token, refreshToken: data.refresh_token },
      data.user
    )
    return data.access_token as string
  } catch (err) {
    // Dentro da Promise compartilhada: roda uma vez só, não uma por request.
    logout()
    throw err
  }
}

// Renova a sessão e devolve o novo access token. Chamadas concorrentes
// compartilham a mesma renovação; ao terminar (sucesso ou falha) libera
// para a próxima.
export function refreshSession(): Promise<string> {
  if (!inFlight) {
    inFlight = doRefresh().finally(() => {
      inFlight = null
    })
  }
  return inFlight
}
