import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { AxiosAdapter } from 'axios'
import api from '@/services/api'
import { NotificationsSection } from './NotificationsSection'

const defaults = {
  task_reminders: true, task_lead_minutes: 15,
  morning_digest: true, morning_time: '07:00',
  evening_digest: false, evening_time: '21:00',
}
let enabled = true
const puts: unknown[] = []

const adapter: AxiosAdapter = async (config) => {
  const ok = (data: unknown) => ({ status: 200, statusText: 'OK', data, headers: {}, config })
  if (config.url === '/notifications/config') return ok({ enabled, vapid_public_key: enabled ? 'BKey' : undefined })
  if (config.method === 'get' && config.url === '/notifications/settings') return ok({ settings: defaults, devices: 1 })
  if (config.method === 'put' && config.url === '/notifications/settings') {
    const body = JSON.parse(String(config.data))
    puts.push(body)
    return ok({ settings: body, devices: 1 })
  }
  throw new Error(`unexpected ${config.method} ${config.url}`)
}

const originalAdapter = api.defaults.adapter

beforeEach(() => {
  enabled = true
  puts.length = 0
  api.defaults.adapter = adapter
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
  vi.restoreAllMocks()
})

function renderSection() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <NotificationsSection />
    </QueryClientProvider>
  )
}

describe('NotificationsSection', () => {
  it('mostra os padrões do produto e salva o resumo da noite quando ligado', async () => {
    const user = userEvent.setup()
    renderSection()

    const evening = await screen.findByRole('checkbox', { name: 'Resumo da noite' })
    expect(screen.getByRole('checkbox', { name: 'Lembrete de compromissos do Planner' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Resumo da manhã' })).toBeChecked()
    expect(evening).not.toBeChecked()
    expect(screen.getByText('Recebendo em 1 aparelho.')).toBeInTheDocument()

    await user.click(evening)
    await user.click(screen.getByRole('button', { name: 'Salvar preferências' }))

    await waitFor(() => expect(puts).toEqual([{ ...defaults, evening_digest: true }]))
  })

  it('avisa quando o servidor ainda não tem notificações', async () => {
    enabled = false
    renderSection()
    expect(await screen.findByText('As notificações ainda não estão disponíveis.')).toBeInTheDocument()
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
  })

  it('no iPhone fora da Tela de Início, explica que precisa instalar', async () => {
    vi.spyOn(navigator, 'userAgent', 'get').mockReturnValue(
      'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1')
    renderSection()
    expect(await screen.findByText(/Adicionar à Tela de Início/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Ligar neste aparelho' })).not.toBeInTheDocument()
  })
})
