import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { AxiosError, type AxiosAdapter } from 'axios'
import api from '@/services/api'
import { LoginPage } from './LoginPage'

let status = 401
const adapter: AxiosAdapter = async (config) => {
  throw new AxiosError(`status ${status}`, 'ERR_BAD_REQUEST', config, null, {
    status, statusText: '', data: { error: status === 429 ? 'too many requests' : 'invalid credentials' }, headers: {}, config,
  })
}
const originalAdapter = api.defaults.adapter
beforeEach(() => { api.defaults.adapter = adapter })
afterEach(() => { api.defaults.adapter = originalAdapter })

async function tryLogin() {
  const user = userEvent.setup()
  render(<MemoryRouter><LoginPage /></MemoryRouter>)
  await user.type(screen.getByLabelText(/e-mail/i), 'ana@x.com')
  await user.type(screen.getByLabelText(/senha/i), 'errada-123')
  await user.click(screen.getByRole('button', { name: 'Entrar' }))
}

describe('LoginPage — mensagens de erro em português', () => {
  it('senha errada', async () => {
    status = 401
    await tryLogin()
    expect(await screen.findByText('E-mail ou senha incorretos.')).toBeInTheDocument()
    expect(screen.queryByText(/invalid credentials/)).not.toBeInTheDocument()
  })

  it('muitas tentativas', async () => {
    status = 429
    await tryLogin()
    expect(await screen.findByText(/Muitas tentativas seguidas/)).toBeInTheDocument()
  })
})
