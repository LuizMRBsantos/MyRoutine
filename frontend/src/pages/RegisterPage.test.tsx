import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { AxiosError, type AxiosAdapter, type InternalAxiosRequestConfig } from 'axios'
import api from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { RegisterPage } from './RegisterPage'

// Adapter falso roteando por URL: o convite "bom" existe para ana@x.com; o
// resto é inválido. O corpo do cadastro fica guardado para as asserções.
let registerBody: Record<string, unknown> | null = null
let registerError: string | null = null

function fail(config: InternalAxiosRequestConfig, status: number, error: string): never {
  throw new AxiosError(`status ${status}`, 'ERR_BAD_REQUEST', config, null, {
    status, statusText: '', data: { error }, headers: {}, config,
  })
}

const adapter: AxiosAdapter = async (config) => {
  if (config.url?.startsWith('/auth/invites/')) {
    if (config.url === '/auth/invites/bom') {
      return { status: 200, statusText: 'OK', data: { email: 'ana@x.com' }, headers: {}, config }
    }
    fail(config, 404, 'invite is invalid or expired')
  }
  if (config.url === '/auth/register') {
    registerBody = JSON.parse(String(config.data))
    if (registerError) fail(config, 403, registerError)
    return {
      status: 201, statusText: 'Created', headers: {}, config,
      data: { access_token: 'a', refresh_token: 'r', user: { id: '1', name: 'Ana', email: 'ana@x.com', createdAt: '' } },
    }
  }
  fail(config, 404, 'not found')
}

const originalAdapter = api.defaults.adapter

beforeEach(() => {
  registerBody = null
  registerError = null
  api.defaults.adapter = adapter
  useAuthStore.setState({ accessToken: null, refreshToken: null, user: null, isAuthenticated: false })
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
})

function renderAt(url: string) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <Routes>
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/" element={<div>início</div>} />
      </Routes>
    </MemoryRouter>
  )
}

describe('RegisterPage (cadastro por convite)', () => {
  it('com convite válido, preenche e trava o e-mail e envia o código', async () => {
    const user = userEvent.setup()
    renderAt('/register?convite=bom')

    const email = await screen.findByDisplayValue('ana@x.com')
    expect(email).toHaveAttribute('readonly')

    await user.type(screen.getByLabelText('Nome'), 'Ana')
    await user.type(screen.getByLabelText('Senha'), 'senha-forte-123')
    await user.click(screen.getByRole('button', { name: 'Criar conta' }))

    await screen.findByText('início')
    expect(registerBody).toMatchObject({ email: 'ana@x.com', invite_token: 'bom' })
    expect(useAuthStore.getState().isAuthenticated).toBe(true)
  })

  it('com convite inválido, avisa e não mostra o formulário', async () => {
    renderAt('/register?convite=velho')

    expect(await screen.findByRole('alert')).toHaveTextContent('não é mais válido')
    expect(screen.queryByRole('button', { name: 'Criar conta' })).not.toBeInTheDocument()
  })

  it('sem convite, avisa do beta fechado e traduz a recusa do servidor', async () => {
    const user = userEvent.setup()
    registerError = 'an invite is required to register'
    renderAt('/register')

    expect(screen.getByText(/beta fechado/)).toBeInTheDocument()
    await user.type(screen.getByLabelText('Nome'), 'Zé')
    await user.type(screen.getByLabelText('E-mail'), 'ze@x.com')
    await user.type(screen.getByLabelText('Senha'), 'senha-forte-123')
    await user.click(screen.getByRole('button', { name: 'Criar conta' }))

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent('você precisa de um convite')
    )
    expect(registerBody).not.toHaveProperty('invite_token')
  })
})
