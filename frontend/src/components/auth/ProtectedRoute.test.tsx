import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'

const mockPost = vi.fn()
vi.mock('axios', () => ({
  default: { post: (...args: unknown[]) => mockPost(...args) },
}))

const { ProtectedRoute } = await import('./ProtectedRoute')

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
  useAuthStore.setState({
    accessToken: null, refreshToken: null, user: null, isAuthenticated: false,
  })
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
})
