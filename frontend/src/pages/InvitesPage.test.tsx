import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { AxiosAdapter } from 'axios'
import api from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { InvitesPage } from './InvitesPage'

const createdInvite = {
  id: 'i1', email: 'bia@x.com', token: 'tok123', expires_at: '2026-10-06T12:00:00Z',
}
let listed: unknown[] = []

const adapter: AxiosAdapter = async (config) => {
  if (config.method === 'get' && config.url === '/admin/invites') {
    return { status: 200, statusText: 'OK', data: { invites: listed }, headers: {}, config }
  }
  if (config.method === 'post' && config.url === '/admin/invites') {
    listed = [{ ...createdInvite, status: 'pending', created_at: '2026-09-29T12:00:00Z', used_at: null }]
    return { status: 201, statusText: 'Created', data: createdInvite, headers: {}, config }
  }
  throw new Error(`unexpected ${config.method} ${config.url}`)
}

const originalAdapter = api.defaults.adapter

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <InvitesPage />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  listed = []
  api.defaults.adapter = adapter
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
  vi.restoreAllMocks()
})

describe('InvitesPage', () => {
  it('gera o link do convite e lista o convite como aguardando', async () => {
    useAuthStore.setState({
      user: { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '', is_admin: true },
      isAuthenticated: true,
    })
    const user = userEvent.setup()
    renderPage()

    expect(await screen.findByText('Nenhum convite ainda.')).toBeInTheDocument()

    await user.type(screen.getByLabelText('E-mail da pessoa convidada'), 'bia@x.com')
    await user.click(screen.getByRole('button', { name: 'Gerar convite' }))

    const link = await screen.findByLabelText('Link do convite')
    expect(link).toHaveValue(`${window.location.origin}/register?convite=tok123`)
    const message = screen.getByLabelText('Mensagem sugerida') as HTMLTextAreaElement
    expect(message.value).toContain(`${window.location.origin}/register?convite=tok123`)
    expect(await screen.findByText('⏳ Aguardando')).toBeInTheDocument()
  })

  it('não mostra nada de admin para quem não é admin', () => {
    useAuthStore.setState({
      user: { id: '2', name: 'Ana', email: 'a@x.com', createdAt: '' },
      isAuthenticated: true,
    })
    renderPage()

    expect(screen.getByText('Esta página é só para administradores.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Gerar convite' })).not.toBeInTheDocument()
  })
})
