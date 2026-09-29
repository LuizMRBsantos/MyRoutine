import axios from 'axios'
import { useAuthStore } from '@/store/authStore'

// O backend trata um refresh token apresentado duas vezes como roubo e revoga
// TODAS as sessões do usuário. Por isso toda renovação passa por aqui:
// - dentro da aba, quem chegar durante uma renovação recebe a mesma Promise;
// - entre abas (ou PWA + navegador), a Web Locks API garante que só uma
//   renove por vez, e cada uma relê o token persistido já dentro da trava —
//   se outra aba girou A→B, esta usa B em vez do A (revogado) em memória.
let inFlight: Promise<string> | null = null

const LOCK_NAME = 'myroutine-auth-refresh'

// Sem sessão persistida (outra aba fez logout ou o storage foi limpo).
const LOGGED_OUT = Symbol('logged-out')

// Lê o refresh token direto do storage do persist (localStorage é a fonte da
// verdade compartilhada entre abas). Se o storage não estiver disponível
// (ex.: bloqueado pelo navegador), o persist também não funciona, então o
// único token que existe é o da memória.
async function readPersistedRefreshToken(): Promise<string | typeof LOGGED_OUT | null> {
  const { name, storage } = useAuthStore.persist.getOptions()
  if (!name || !storage) return null
  try {
    const stored = await storage.getItem(name)
    return stored?.state?.refreshToken ?? LOGGED_OUT
  } catch {
    return null
  }
}

function withCrossTabLock<T>(fn: () => Promise<T>): Promise<T> {
  const locks = typeof navigator !== 'undefined' ? navigator.locks : undefined
  if (!locks?.request) return fn() // navegador sem Web Locks: segue sem trava
  return locks.request(LOCK_NAME, fn) as Promise<T>
}

async function doRefresh(): Promise<string> {
  const { setAuth, logout } = useAuthStore.getState()

  const persisted = await readPersistedRefreshToken()
  if (persisted === LOGGED_OUT) {
    logout()
    throw new Error('sessão encerrada em outra aba')
  }
  const refreshToken = persisted ?? useAuthStore.getState().refreshToken

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
    // setAuth persiste o novo refresh token de forma síncrona (localStorage),
    // antes de a trava ser liberada para a próxima aba.
    setAuth(
      { accessToken: data.access_token, refreshToken: data.refresh_token },
      data.user
    )
    return data.access_token as string
  } catch (err) {
    // Sem resposta do servidor (sem internet, servidor fora do ar), a sessão
    // pode estar perfeitamente válida. Deslogar aqui trancaria a pessoa fora
    // do app instalado — e sem rede ela nem conseguiria entrar de novo. Só uma
    // recusa de verdade do servidor encerra a sessão.
    if (axios.isAxiosError(err) && !err.response) throw err
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
    inFlight = withCrossTabLock(doRefresh).finally(() => {
      inFlight = null
    })
  }
  return inFlight
}
