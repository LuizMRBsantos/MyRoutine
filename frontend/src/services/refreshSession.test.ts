import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { AxiosError, type AxiosAdapter } from 'axios'

// Cada "aba" é uma instância isolada dos módulos (store, refreshSession e
// axios), todas compartilhando o mesmo localStorage — como abas do mesmo
// navegador. O servidor falso reproduz a rotação: cada refresh token só vale
// uma vez; apresentar um já girado responde 401.

const user = { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '' }
const PERSIST_KEY = 'myroutine-auth'

let validToken: string | null
let rotation: number
let refreshGate: (() => Promise<void>) | null
const sentTokens: string[] = []
let active = 0
let maxActive = 0
// Sem internet / servidor fora: o pedido falha sem resposta nenhuma.
let networkDown = false

const adapter: AxiosAdapter = async (config) => {
  const body = JSON.parse(String(config.data ?? '{}'))
  if (networkDown) throw new AxiosError('Network Error', 'ERR_NETWORK', config)
  sentTokens.push(body.refresh_token)
  active++
  maxActive = Math.max(maxActive, active)
  try {
    if (refreshGate) await refreshGate()
    if (body.refresh_token !== validToken) {
      validToken = null // reuso detectado: o servidor revoga tudo
      throw new AxiosError('status 401', 'ERR_BAD_REQUEST', config, null, {
        status: 401, statusText: '', data: {}, headers: {}, config,
      })
    }
    rotation++
    validToken = `r${rotation}`
    return {
      status: 200, statusText: 'OK', headers: {}, config,
      data: { access_token: `a${rotation}`, refresh_token: validToken, user },
    }
  } finally {
    active--
  }
}

async function openTab(state: { accessToken: string | null; refreshToken: string | null }) {
  vi.resetModules()
  const axios = (await import('axios')).default
  axios.defaults.adapter = adapter
  const { useAuthStore } = await import('@/store/authStore')
  const { refreshSession } = await import('./refreshSession')
  // Estado em memória da aba (pode divergir do persistido, como numa aba que
  // ficou aberta enquanto outra girava o token).
  useAuthStore.setState({ ...state, user, isAuthenticated: !!state.refreshToken })
  return { useAuthStore, refreshSession }
}

function persistedRefreshToken(): string | null {
  const raw = localStorage.getItem(PERSIST_KEY)
  return raw ? JSON.parse(raw).state.refreshToken : null
}

function setPersisted(refreshToken: string | null) {
  localStorage.setItem(PERSIST_KEY, JSON.stringify({
    state: { refreshToken, user: refreshToken ? user : null, isAuthenticated: !!refreshToken },
    version: 0,
  }))
}

// navigator.locks falso: exclusivo por nome, em fila (como a Web Locks API).
function installFakeLocks() {
  const queues = new Map<string, Promise<unknown>>()
  const request = vi.fn((name: string, cb: () => Promise<unknown>) => {
    const prev = queues.get(name) ?? Promise.resolve()
    const run = prev.then(() => cb())
    queues.set(name, run.catch(() => undefined))
    return run
  })
  Object.defineProperty(navigator, 'locks', { value: { request }, configurable: true })
  return request
}

function removeLocks() {
  Object.defineProperty(navigator, 'locks', { value: undefined, configurable: true })
}

beforeEach(() => {
  validToken = 'r0'
  rotation = 0
  refreshGate = null
  networkDown = false
  sentTokens.length = 0
  active = 0
  maxActive = 0
})

afterEach(() => {
  removeLocks()
  vi.restoreAllMocks()
})

describe('refreshSession — coordenação entre abas', () => {
  it('usa o refresh token persistido, não o antigo em memória (outra aba já girou)', async () => {
    installFakeLocks()
    const tabA = await openTab({ accessToken: null, refreshToken: 'r0' })
    const tabB = await openTab({ accessToken: null, refreshToken: 'r0' })

    // Aba A gira r0 -> r1 (persistido passa a ser r1).
    await tabA.refreshSession()
    expect(persistedRefreshToken()).toBe('r1')
    // Aba B continua com r0 em memória.
    expect(tabB.useAuthStore.getState().refreshToken).toBe('r0')

    const access = await tabB.refreshSession()

    expect(sentTokens).toEqual(['r0', 'r1'])
    expect(access).toBe('a2')
    expect(tabB.useAuthStore.getState().isAuthenticated).toBe(true)
    expect(persistedRefreshToken()).toBe('r2')
  })

  it('com navigator.locks, renovações concorrentes de abas diferentes rodam em série', async () => {
    const request = installFakeLocks()
    const tabA = await openTab({ accessToken: null, refreshToken: 'r0' })
    const tabB = await openTab({ accessToken: null, refreshToken: 'r0' })
    refreshGate = () => new Promise((r) => setTimeout(r, 20))

    const [a, b] = await Promise.all([tabA.refreshSession(), tabB.refreshSession()])

    expect(request).toHaveBeenCalledTimes(2)
    expect(request.mock.calls.every(([name]) => name === 'myroutine-auth-refresh')).toBe(true)
    expect(maxActive).toBe(1)
    expect(sentTokens).toEqual(['r0', 'r1'])
    expect([a, b]).toEqual(['a1', 'a2'])
    expect(tabA.useAuthStore.getState().isAuthenticated).toBe(true)
    expect(tabB.useAuthStore.getState().isAuthenticated).toBe(true)
    expect(persistedRefreshToken()).toBe('r2')
  })

  it('sem navigator.locks, ainda renova relendo o estado persistido', async () => {
    removeLocks()
    const tab = await openTab({ accessToken: null, refreshToken: 'r0' })
    // Outra aba girou o token e persistiu.
    validToken = 'r7'
    setPersisted('r7')

    const access = await tab.refreshSession()

    expect(sentTokens).toEqual(['r7'])
    expect(access).toBe('a1')
    expect(tab.useAuthStore.getState().refreshToken).toBe('r1')
  })

  it('se outra aba fez logout, não chama /auth/refresh e desloga localmente', async () => {
    installFakeLocks()
    const tab = await openTab({ accessToken: 'velho', refreshToken: 'r0' })
    setPersisted(null)

    await expect(tab.refreshSession()).rejects.toThrow()

    expect(sentTokens).toEqual([])
    expect(tab.useAuthStore.getState().isAuthenticated).toBe(false)
    expect(tab.useAuthStore.getState().refreshToken).toBeNull()
  })

  it('se a chave persistida sumiu (storage limpo), também desloga sem chamar o servidor', async () => {
    installFakeLocks()
    const tab = await openTab({ accessToken: 'velho', refreshToken: 'r0' })
    localStorage.removeItem(PERSIST_KEY)

    await expect(tab.refreshSession()).rejects.toThrow()

    expect(sentTokens).toEqual([])
    expect(tab.useAuthStore.getState().isAuthenticated).toBe(false)
  })

  it('mantém uma única renovação por aba: chamadas concorrentes pegam a mesma trava uma vez', async () => {
    const request = installFakeLocks()
    const tab = await openTab({ accessToken: null, refreshToken: 'r0' })
    refreshGate = () => new Promise((r) => setTimeout(r, 10))

    const results = await Promise.all([
      tab.refreshSession(), tab.refreshSession(), tab.refreshSession(),
    ])

    expect(request).toHaveBeenCalledTimes(1)
    expect(sentTokens).toEqual(['r0'])
    expect(results).toEqual(['a1', 'a1', 'a1'])
  })

  it('refresh recusado: logout uma vez e a trava é liberada para a próxima aba', async () => {
    installFakeLocks()
    const tab = await openTab({ accessToken: null, refreshToken: 'r0' })
    validToken = 'outro' // servidor não aceita r0
    const logout = vi.fn(tab.useAuthStore.getState().logout)
    tab.useAuthStore.setState({ logout })

    const results = await Promise.allSettled([tab.refreshSession(), tab.refreshSession()])

    expect(results.every((r) => r.status === 'rejected')).toBe(true)
    expect(sentTokens).toEqual(['r0'])
    expect(logout).toHaveBeenCalledTimes(1)
    expect(persistedRefreshToken()).toBeNull()

    // A trava não ficou presa: outra aba consegue seguir (e vê o logout).
    const other = await openTab({ accessToken: null, refreshToken: 'r0' })
    setPersisted(null)
    await expect(other.refreshSession()).rejects.toThrow()
    expect(sentTokens).toEqual(['r0'])
  })
})

describe('refreshSession — sem internet', () => {
  it('não desloga quando o servidor não responde, e a sessão volta com a rede', async () => {
    setPersisted('r0')
    const tab = await openTab({ accessToken: null, refreshToken: 'r0' })

    networkDown = true
    await expect(tab.refreshSession()).rejects.toThrow()

    // A sessão continua: abrir o app no metrô não pode apagar o login.
    expect(tab.useAuthStore.getState().isAuthenticated).toBe(true)
    expect(persistedRefreshToken()).toBe('r0')

    networkDown = false
    await expect(tab.refreshSession()).resolves.toBe('a1')
    expect(tab.useAuthStore.getState().isAuthenticated).toBe(true)
  })
})
