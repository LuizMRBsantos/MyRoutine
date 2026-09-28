import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import axios, { AxiosError, type AxiosAdapter } from 'axios'
import { useAuthStore } from '@/store/authStore'
import api from '@/services/api'
import { ProtectedRoute } from './ProtectedRoute'

// A renovação usa o axios puro (fora dos interceptors do `api`); espionamos o
// `axios.post` para contar quantas vezes o /auth/refresh é chamado.
const mockPost = vi.fn()
const originalApiAdapter = api.defaults.adapter

function renderGuarded() {
  return render(
    <MemoryRouter initialEntries={['/']}>
      <Routes>
        <Route path="/" element={<ProtectedRoute><div>conteúdo privado</div></ProtectedRoute>} />
        <Route path="/login" element={<div>tela de login</div>} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  mockPost.mockReset()
  vi.spyOn(axios, 'post').mockImplementation((...args: unknown[]) => mockPost(...args))
  useAuthStore.setState({
    accessToken: null, refreshToken: null, user: null, isAuthenticated: false,
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  api.defaults.adapter = originalApiAdapter
})

describe('ProtectedRoute', () => {
  it('manda para o login quem não está autenticado', () => {
    renderGuarded()
    expect(screen.getByText('tela de login')).toBeInTheDocument()
  })

  it('renderiza direto quando o access token já está em memória', () => {
    useAuthStore.setState({
      accessToken: 'token-vivo', refreshToken: 'r1',
      user: { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '' },
      isAuthenticated: true,
    })

    renderGuarded()
    expect(screen.getByText('conteúdo privado')).toBeInTheDocument()
    expect(mockPost).not.toHaveBeenCalled()
  })

  // Regressão: o refresh rotaciona o token, o que mudava as dependências do
  // efeito e disparava o cleanup antes do .finally — a tela ficava presa em
  // "Carregando…" para sempre depois de um reload com sessão salva.
  it('sai do estado de carregando após o refresh rotacionar o token', async () => {
    useAuthStore.setState({
      accessToken: null, refreshToken: 'refresh-antigo',
      user: { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '' },
      isAuthenticated: true,
    })

    mockPost.mockResolvedValue({
      data: {
        access_token: 'novo-access',
        refresh_token: 'refresh-NOVO',   // rotacionado: muda a dependência
        user: { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '' },
      },
    })

    renderGuarded()
    expect(screen.getByText('Carregando…')).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.getByText('conteúdo privado')).toBeInTheDocument()
    })
    expect(screen.queryByText('Carregando…')).not.toBeInTheDocument()
    expect(useAuthStore.getState().accessToken).toBe('novo-access')
  })

  it('só tenta o refresh uma vez', async () => {
    useAuthStore.setState({
      accessToken: null, refreshToken: 'r1',
      user: { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '' },
      isAuthenticated: true,
    })
    mockPost.mockResolvedValue({
      data: { access_token: 'a', refresh_token: 'b', user: { id: '1', name: 'L', email: 'e', createdAt: '' } },
    })

    renderGuarded()
    await waitFor(() => expect(screen.getByText('conteúdo privado')).toBeInTheDocument())
    expect(mockPost).toHaveBeenCalledTimes(1)
  })

  it('desloga e manda para o login quando o refresh falha', async () => {
    useAuthStore.setState({
      accessToken: null, refreshToken: 'expirado',
      user: { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '' },
      isAuthenticated: true,
    })
    mockPost.mockRejectedValue(new Error('401'))

    renderGuarded()
    await waitFor(() => expect(screen.getByText('tela de login')).toBeInTheDocument())
    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })

  it('reload + request 401 simultânea fazem um refresh só', async () => {
    useAuthStore.setState({
      accessToken: null, refreshToken: 'r1',
      user: { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '' },
      isAuthenticated: true,
    })

    // Request protegida: só passa com o access token renovado.
    const adapter: AxiosAdapter = async (config) => {
      if (config.headers?.Authorization !== 'Bearer novo-access') {
        throw new AxiosError('401', 'ERR_BAD_REQUEST', config, null, {
          status: 401, statusText: '', data: {}, headers: {}, config,
        })
      }
      return { status: 200, statusText: 'OK', data: 'ok', headers: {}, config }
    }
    api.defaults.adapter = adapter

    // Toda renovação fica pendente até o teste liberar todas juntas.
    const waiters: ((v: unknown) => void)[] = []
    const resolveRefresh = (v: unknown) => waiters.forEach((res) => res(v))
    mockPost.mockImplementation(() => new Promise((res) => { waiters.push(res) }))

    renderGuarded()
    const request = api.get('/habits')

    await waitFor(() => expect(mockPost).toHaveBeenCalled())
    // Dá tempo para o 401 da request chegar ao interceptor com o refresh ainda pendente.
    await new Promise((r) => setTimeout(r, 10))
    resolveRefresh({
      data: {
        access_token: 'novo-access', refresh_token: 'r2',
        user: { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '' },
      },
    })

    await expect(request).resolves.toMatchObject({ data: 'ok' })
    await waitFor(() => expect(screen.getByText('conteúdo privado')).toBeInTheDocument())
    expect(mockPost).toHaveBeenCalledTimes(1)
  })
})
