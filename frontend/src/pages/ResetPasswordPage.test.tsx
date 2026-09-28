import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { AxiosError, type AxiosAdapter, type InternalAxiosRequestConfig } from 'axios'
import api from '@/services/api'
import { ResetPasswordPage } from './ResetPasswordPage'

// O link "bom" pertence a ana@x.com; qualquer outro é inválido.
let resetBodies: unknown[] = []

function fail(config: InternalAxiosRequestConfig, status: number): never {
  throw new AxiosError(`status ${status}`, 'ERR_BAD_REQUEST', config, null, {
    status, statusText: '', data: { error: 'reset link is invalid or expired' }, headers: {}, config,
  })
}

const adapter: AxiosAdapter = async (config) => {
  if (config.url === '/auth/password-resets/bom') {
    if (config.method === 'get') {
      return { status: 200, statusText: 'OK', data: { email: 'ana@x.com' }, headers: {}, config }
    }
    resetBodies.push(JSON.parse(String(config.data)))
    return { status: 204, statusText: 'No Content', data: '', headers: {}, config }
  }
  fail(config, 404)
}

const originalAdapter = api.defaults.adapter

beforeEach(() => {
  resetBodies = []
  api.defaults.adapter = adapter
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
})

function renderAt(url: string) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <Routes>
        <Route path="/redefinir-senha" element={<ResetPasswordPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('ResetPasswordPage', () => {
  it('com link válido, troca a senha e avisa que as sessões caíram', async () => {
    const user = userEvent.setup()
    renderAt('/redefinir-senha?codigo=bom')

    expect(await screen.findByText('Para a conta ana@x.com')).toBeInTheDocument()
    await user.type(screen.getByLabelText('Nova senha'), 'senha-nova-22')
    await user.type(screen.getByLabelText('Repita a nova senha'), 'senha-nova-22')
    await user.click(screen.getByRole('button', { name: 'Salvar nova senha' }))

    expect(await screen.findByText(/Senha alterada/)).toBeInTheDocument()
    expect(resetBodies).toEqual([{ password: 'senha-nova-22' }])
  })

  it('não envia quando as senhas não conferem', async () => {
    const user = userEvent.setup()
    renderAt('/redefinir-senha?codigo=bom')

    await screen.findByText('Para a conta ana@x.com')
    await user.type(screen.getByLabelText('Nova senha'), 'senha-nova-22')
    await user.type(screen.getByLabelText('Repita a nova senha'), 'senha-errada-33')
    await user.click(screen.getByRole('button', { name: 'Salvar nova senha' }))

    expect(screen.getByRole('alert')).toHaveTextContent('As senhas não conferem')
    expect(resetBodies).toEqual([])
  })

  it('com link inválido, orienta a pedir um novo', async () => {
    renderAt('/redefinir-senha?codigo=velho')

    expect(await screen.findByRole('alert')).toHaveTextContent('não é mais válido')
    expect(screen.queryByLabelText('Nova senha')).not.toBeInTheDocument()
  })
})
