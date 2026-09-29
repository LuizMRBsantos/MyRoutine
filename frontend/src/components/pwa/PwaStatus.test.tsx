import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

// O módulo virtual do plugin só existe no build; aqui simulamos o registro.
const sw = vi.hoisted(() => ({
  needRefresh: false,
  setNeedRefresh: vi.fn(),
  updateServiceWorker: vi.fn(),
}))
vi.mock('virtual:pwa-register/react', () => ({
  useRegisterSW: () => ({
    needRefresh: [sw.needRefresh, sw.setNeedRefresh],
    offlineReady: [false, vi.fn()],
    updateServiceWorker: sw.updateServiceWorker,
  }),
}))

import { PwaStatus } from './PwaStatus'

function setOnline(value: boolean) {
  Object.defineProperty(navigator, 'onLine', { configurable: true, get: () => value })
}

beforeEach(() => {
  sw.needRefresh = false
  sw.setNeedRefresh.mockReset()
  sw.updateServiceWorker.mockReset()
  setOnline(true)
})

afterEach(() => setOnline(true))

describe('PwaStatus', () => {
  it('não mostra nada online e sem atualização', () => {
    const { container } = render(<PwaStatus />)
    expect(container).toBeEmptyDOMElement()
  })

  it('avisa quando a conexão cai e some quando volta', () => {
    render(<PwaStatus />)

    act(() => {
      setOnline(false)
      window.dispatchEvent(new Event('offline'))
    })
    expect(screen.getByRole('status')).toHaveTextContent('Você está sem internet')

    act(() => {
      setOnline(true)
      window.dispatchEvent(new Event('online'))
    })
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('oferece a nova versão e só atualiza quando a pessoa aceita', async () => {
    sw.needRefresh = true
    const user = userEvent.setup()
    render(<PwaStatus />)

    expect(screen.getByRole('status')).toHaveTextContent('Nova versão do MyRoutine disponível')
    expect(sw.updateServiceWorker).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: 'Depois' }))
    expect(sw.setNeedRefresh).toHaveBeenCalledWith(false)

    await user.click(screen.getByRole('button', { name: 'Atualizar' }))
    expect(sw.updateServiceWorker).toHaveBeenCalledWith(true)
  })
})
