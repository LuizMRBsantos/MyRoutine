import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import axios, { AxiosError, type AxiosAdapter, type InternalAxiosRequestConfig } from 'axios'
import api from './api'
import { useAuthStore } from '@/store/authStore'

const user = { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '' }

// Adapter falso: substitui a camada HTTP tanto do `api` quanto do axios puro
// (usado na renovação), roteando pela URL. Requests protegidas só passam com o
// access token "novo-access"; a renovação é controlada por `refreshImpl`.
let refreshImpl: () => Promise<unknown>
const refreshCalls: InternalAxiosRequestConfig[] = []
const protectedCalls: { url?: string; auth?: string }[] = []

function fail(config: InternalAxiosRequestConfig, status: number): never {
  throw new AxiosError(`status ${status}`, 'ERR_BAD_REQUEST', config, null, {
    status, statusText: '', data: {}, headers: {}, config,
  })
}

const adapter: AxiosAdapter = async (config) => {
  if (config.url?.endsWith('/auth/refresh')) {
    refreshCalls.push(config)
    try {
      const data = await refreshImpl()
      return { status: 200, statusText: 'OK', data, headers: {}, config }
    } catch {
      fail(config, 401)
    }
  }
  const auth = String(config.headers?.Authorization ?? '')
  protectedCalls.push({ url: config.url, auth })
  if (auth !== 'Bearer novo-access') fail(config, 401)
  return { status: 200, statusText: 'OK', data: { url: config.url }, headers: {}, config }
}

const originalApiAdapter = api.defaults.adapter
const originalAxiosAdapter = axios.defaults.adapter

beforeEach(() => {
  refreshCalls.length = 0
  protectedCalls.length = 0
  api.defaults.adapter = adapter
  axios.defaults.adapter = adapter
  useAuthStore.setState({
    accessToken: 'expirado', refreshToken: 'r1', user, isAuthenticated: true,
  })
})

afterEach(() => {
  api.defaults.adapter = originalApiAdapter
  axios.defaults.adapter = originalAxiosAdapter
  vi.restoreAllMocks()
})

// Renovação que só resolve quando o teste mandar — garante que as três
// requests recebem 401 enquanto ela ainda está em andamento.
// Se o código fizer mais de uma renovação, todas ficam pendentes e são
// liberadas juntas — assim o teste falha na contagem, não por timeout.
function deferredRefresh() {
  const waiters: { res: (v: unknown) => void; rej: (e: unknown) => void }[] = []
  refreshImpl = () => new Promise((res, rej) => { waiters.push({ res, rej }) })
  return {
    resolve: (v: unknown) => waiters.forEach((w) => w.res(v)),
    reject: (e: unknown) => waiters.forEach((w) => w.rej(e)),
  }
}

const until = async (cond: () => boolean) => {
  await vi.waitFor(() => { if (!cond()) throw new Error('ainda não') })
}

describe('api — renovação de sessão única', () => {
  it('3 requests paralelas com 401 fazem exatamente 1 refresh e são repetidas', async () => {
    const refresh = deferredRefresh()

    const pending = Promise.all([
      api.get('/habits'), api.get('/reviews'), api.get('/health'),
    ])

    await until(() => protectedCalls.length === 3 && refreshCalls.length >= 1)
    await new Promise((r) => setTimeout(r, 10))
    refresh.resolve({ access_token: 'novo-access', refresh_token: 'r2', user })

    const results = await pending
    expect(refreshCalls).toHaveLength(1)
    expect(results.map((r) => r.data.url)).toEqual(['/habits', '/reviews', '/health'])
    expect(useAuthStore.getState().accessToken).toBe('novo-access')
    expect(useAuthStore.getState().refreshToken).toBe('r2')
  })

  it('refresh falho: logout uma vez só e as 3 requests rejeitam', async () => {
    const refresh = deferredRefresh()
    const logout = vi.fn(useAuthStore.getState().logout)
    useAuthStore.setState({ logout })

    const pending = Promise.allSettled([
      api.get('/habits'), api.get('/reviews'), api.get('/health'),
    ])

    await until(() => protectedCalls.length === 3 && refreshCalls.length >= 1)
    await new Promise((r) => setTimeout(r, 10))
    refresh.reject(new Error('refresh inválido'))

    const results = await pending
    expect(results.every((r) => r.status === 'rejected')).toBe(true)
    expect(refreshCalls).toHaveLength(1)
    expect(logout).toHaveBeenCalledTimes(1)
    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })

  it('não entra em loop se a request repetida receber 401 de novo', async () => {
    // O refresh "funciona", mas devolve um token que o servidor não aceita.
    refreshImpl = async () => ({ access_token: 'outro', refresh_token: 'r2', user })

    await expect(api.get('/habits')).rejects.toBeInstanceOf(AxiosError)
    expect(refreshCalls).toHaveLength(1)
    expect(protectedCalls).toHaveLength(2)
  })

  it('libera para uma nova renovação depois que a anterior termina', async () => {
    refreshImpl = async () => ({ access_token: 'novo-access', refresh_token: 'r2', user })
    await api.get('/habits')

    useAuthStore.setState({ accessToken: 'expirado-de-novo' })
    await api.get('/habits')

    expect(refreshCalls).toHaveLength(2)
  })
})
