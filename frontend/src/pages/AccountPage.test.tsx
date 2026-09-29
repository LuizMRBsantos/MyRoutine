import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AxiosError, type AxiosAdapter, type InternalAxiosRequestConfig } from 'axios'
import api from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { AccountPage } from './AccountPage'

// Servidor falso: senha correta é "senha-certa-1".
const calls: { method?: string; url?: string; body?: unknown }[] = []

function fail(config: InternalAxiosRequestConfig, status: number, error: string): never {
  throw new AxiosError(`status ${status}`, 'ERR_BAD_REQUEST', config, null, {
    status, statusText: '', data: { error }, headers: {}, config,
  })
}

const profile = {
  id: '1', name: 'Luiz', email: 'l@x.com', avatar_url: null,
  timezone: 'America/Sao_Paulo', created_at: '2026-01-01T00:00:00Z',
}

const adapter: AxiosAdapter = async (config) => {
  const body = config.data ? JSON.parse(String(config.data)) : undefined
  calls.push({ method: config.method, url: config.url, body })
  const ok = (data: unknown, status = 200) => ({ status, statusText: 'OK', data, headers: {}, config })

  if (config.method === 'get' && config.url === '/me') return ok(profile)
  if (config.method === 'patch' && config.url === '/me') return ok({ ...profile, ...body })
  if (config.method === 'put' && config.url === '/me/password') {
    if (body.current_password !== 'senha-certa-1') fail(config, 403, 'current password is incorrect')
    return ok({ message: 'password updated' })
  }
  if (config.method === 'delete' && config.url === '/me') {
    if (body.password !== 'senha-certa-1') fail(config, 403, 'password is incorrect')
    return ok('', 204)
  }
  if (config.method === 'get' && config.url === '/me/export') {
    return {
      status: 200, statusText: 'OK', config,
      headers: { 'content-disposition': 'attachment; filename="myroutine-dados-2026-09-29.json"' },
      data: new Blob(['{}'], { type: 'application/json' }),
    }
  }
  fail(config, 404, 'not found')
}

const originalAdapter = api.defaults.adapter

beforeEach(() => {
  calls.length = 0
  api.defaults.adapter = adapter
  useAuthStore.setState({
    accessToken: 'a', refreshToken: 'r', isAuthenticated: true,
    user: { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '' },
  })
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
  vi.restoreAllMocks()
})

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <AccountPage />
    </QueryClientProvider>
  )
}

describe('AccountPage', () => {
  it('salva nome e fuso e atualiza o nome do menu', async () => {
    const user = userEvent.setup()
    renderPage()

    const name = await screen.findByLabelText('Nome')
    await user.clear(name)
    await user.type(name, 'Luiz M.')
    await user.selectOptions(screen.getByLabelText('Fuso horário'), 'UTC')
    await user.click(screen.getByRole('button', { name: 'Salvar perfil' }))

    await waitFor(() => expect(useAuthStore.getState().user?.name).toBe('Luiz M.'))
    expect(calls.find(c => c.method === 'patch')?.body).toEqual({ name: 'Luiz M.', timezone: 'UTC' })
  })

  it('senha atual errada mostra o erro e não desloga; certa desloga', async () => {
    const user = userEvent.setup()
    renderPage()

    await user.type(await screen.findByLabelText('Senha atual'), 'errada-000')
    await user.type(screen.getByLabelText('Nova senha'), 'nova-senha-22')
    await user.type(screen.getByLabelText('Repita a nova senha'), 'nova-senha-22')
    await user.click(screen.getByRole('button', { name: 'Trocar senha' }))

    expect(await screen.findByText('A senha atual está incorreta.')).toBeInTheDocument()
    expect(useAuthStore.getState().isAuthenticated).toBe(true)

    await user.clear(screen.getByLabelText('Senha atual'))
    await user.type(screen.getByLabelText('Senha atual'), 'senha-certa-1')
    await user.click(screen.getByRole('button', { name: 'Trocar senha' }))

    await waitFor(() => expect(useAuthStore.getState().isAuthenticated).toBe(false))
  })

  it('apagar a conta pede confirmação em dois passos e a senha', async () => {
    const user = userEvent.setup()
    renderPage()

    expect(screen.queryByRole('button', { name: 'Apagar definitivamente' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Quero apagar minha conta' }))

    await user.type(screen.getByLabelText('Digite sua senha para confirmar'), 'errada-000')
    await user.click(screen.getByRole('button', { name: 'Apagar definitivamente' }))
    expect(await screen.findByText('Senha incorreta — nada foi apagado.')).toBeInTheDocument()
    expect(useAuthStore.getState().isAuthenticated).toBe(true)

    await user.clear(screen.getByLabelText('Digite sua senha para confirmar'))
    await user.type(screen.getByLabelText('Digite sua senha para confirmar'), 'senha-certa-1')
    await user.click(screen.getByRole('button', { name: 'Apagar definitivamente' }))

    await waitFor(() => expect(useAuthStore.getState().isAuthenticated).toBe(false))
    expect(calls.filter(c => c.method === 'delete')).toHaveLength(2)
  })

  it('baixa os dados com o nome de arquivo do servidor', async () => {
    const user = userEvent.setup()
    const createURL = vi.fn(() => 'blob:dados')
    const revokeURL = vi.fn()
    Object.assign(URL, { createObjectURL: createURL, revokeObjectURL: revokeURL })
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      expect(this.download).toBe('myroutine-dados-2026-09-29.json')
      expect(this.href).toBe('blob:dados')
    })
    renderPage()

    await user.click(screen.getByRole('button', { name: 'Baixar meus dados' }))

    await waitFor(() => expect(click).toHaveBeenCalledTimes(1))
    expect(revokeURL).toHaveBeenCalledWith('blob:dados')
  })
})
