import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, within, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AxiosError, type AxiosAdapter, type InternalAxiosRequestConfig } from 'axios'
import api from '@/services/api'
import { lineKey } from '@/lib/trackDay/parser'
import { JournalPage } from './JournalPage'

// Servidor falso com estado: texto do dia, etiquetas por linha e itens
// registrados — como o backend (/journal/{date}).
type Item = { source_id: string; kind: string; label: string; amount_cents?: number }
let content = ''
let items: Item[] = []
let habits: { id: string; name: string; category: string }[] = []
let workoutConflict = false
const calls: { method?: string; url?: string; body?: Record<string, unknown> }[] = []

function view() {
  const line_ids: Record<string, string> = {}
  for (const l of content.split('\n')) {
    const k = lineKey(l)
    if (k) line_ids[k] = `id:${k}`
  }
  return { date: '2026-09-29', content, updated_at: content ? '2026-09-29T12:00:00Z' : null, line_ids, items }
}

function fail(config: InternalAxiosRequestConfig, status: number, error: string): never {
  throw new AxiosError(`status ${status}`, 'ERR_BAD_REQUEST', config, null, {
    status, statusText: '', data: { error }, headers: {}, config,
  })
}

const adapter: AxiosAdapter = async (config) => {
  const body = config.data ? JSON.parse(String(config.data)) : undefined
  calls.push({ method: config.method, url: config.url, body })
  const ok = (data: unknown) => ({ status: 200, statusText: 'OK', data, headers: {}, config })
  const url = config.url ?? ''

  if (url === '/habits') return ok({ habits })
  if (url.startsWith('/journal/')) {
    if (config.method === 'get') return ok(view())
    if (config.method === 'put') { content = body.content; return ok(view()) }
    if (config.method === 'post') {
      if (body.kind === 'workout' && workoutConflict) fail(config, 409, 'habit already checked in that day')
      const id = `id:${lineKey(body.line)}`
      items = items.filter(i => i.source_id !== id)
      items.push(body.kind === 'transaction'
        ? { source_id: id, kind: 'transaction', label: body.expense.description, amount_cents: body.expense.amount_cents }
        : { source_id: id, kind: 'workout', label: 'Correr' })
      return ok(view())
    }
    if (config.method === 'delete') {
      const id = decodeURIComponent(url.split('/items/')[1])
      items = items.filter(i => i.source_id !== id)
      return ok(view())
    }
  }
  fail(config, 404, 'not found')
}

const originalAdapter = api.defaults.adapter

beforeEach(() => {
  content = ''
  items = []
  habits = []
  workoutConflict = false
  calls.length = 0
  api.defaults.adapter = adapter
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
})

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <JournalPage />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

const panel = () => screen.getByRole('region', { name: 'Reconhecido no texto' })

describe('JournalPage (diário)', () => {
  it('salva sozinho depois de uma pausa na digitação', async () => {
    renderPage()
    const editor = await screen.findByLabelText('Texto do dia')
    fireEvent.change(editor, { target: { value: '• Revisar relatório' } })

    await waitFor(() => expect(calls.some(c => c.method === 'put')).toBe(true), { timeout: 2000 })
    expect(calls.find(c => c.method === 'put')?.body).toEqual({ content: '• Revisar relatório' })
    expect(await screen.findByText(/Salvo às/)).toBeInTheDocument()
  })

  it('registra um gasto reconhecido e permite desfazer', async () => {
    const user = userEvent.setup()
    renderPage()
    fireEvent.change(await screen.findByLabelText('Texto do dia'), { target: { value: '$ 32,50 Almoço #alimentação' } })

    const row = await within(panel()).findByText(/R\$\s?32,50 · Almoço/)
    expect(within(panel()).getByText('Gasto · Alimentação')).toBeInTheDocument()
    await user.click(within(panel()).getByRole('button', { name: 'Registrar' }))

    expect(await within(panel()).findByText('✓ Registrado')).toBeInTheDocument()
    const post = calls.find(c => c.method === 'post')
    expect(post?.body).toEqual({
      line: '$ 32,50 Almoço #alimentação',
      kind: 'transaction',
      expense: { amount_cents: 3250, category: 'alimentacao', description: 'Almoço' },
    })
    // Salvou o texto antes de registrar (a etiqueta vem do texto salvo).
    expect(calls.findIndex(c => c.method === 'put')).toBeLessThan(calls.findIndex(c => c.method === 'post'))

    await user.click(within(panel()).getByRole('button', { name: 'Desfazer' }))
    expect(await within(panel()).findByRole('button', { name: 'Registrar' })).toBeInTheDocument()
    expect(row).toBeDefined()
  })

  it('treino sem hábito de Saúde correspondente não pode ser registrado', async () => {
    habits = [{ id: 'h9', name: 'Meditar', category: 'mind' }]
    renderPage()
    fireEvent.change(await screen.findByLabelText('Texto do dia'), { target: { value: 'corrida 5km 30min' } })

    expect(await within(panel()).findByText(/Sem hábito de Saúde para isso/)).toBeInTheDocument()
    expect(within(panel()).getByRole('button', { name: 'Registrar' })).toBeDisabled()
  })

  it('treino vai para o hábito certo com as medidas; conflito é explicado', async () => {
    const user = userEvent.setup()
    habits = [{ id: 'h1', name: 'Correr', category: 'health' }]
    renderPage()
    fireEvent.change(await screen.findByLabelText('Texto do dia'), { target: { value: 'corrida 1h30min 12km RPE 7' } })

    expect(await within(panel()).findByText('Treino → hábito Correr')).toBeInTheDocument()

    workoutConflict = true
    await user.click(within(panel()).getByRole('button', { name: 'Registrar' }))
    expect(await within(panel()).findByRole('alert')).toHaveTextContent('já foi marcado neste dia')

    workoutConflict = false
    await user.click(within(panel()).getByRole('button', { name: 'Registrar' }))
    expect(await within(panel()).findByText('✓ Registrado')).toBeInTheDocument()
    expect(calls.filter(c => c.method === 'post').at(-1)?.body).toEqual({
      line: 'corrida 1h30min 12km RPE 7',
      kind: 'workout',
      workout: { habit_id: 'h1', metrics: { km: 12, time_min: 90, rpe: 7 }, time_minutes: 90 },
    })
  })

  it('mostra o que foi registrado de uma linha que saiu do texto', async () => {
    content = '• só uma tarefa'
    items = [{ source_id: 'id:$ 10 Café', kind: 'transaction', label: 'Café', amount_cents: 1000 }]
    renderPage()
    await screen.findByLabelText('Texto do dia')

    expect(await within(panel()).findByText('Registrado, mas não está mais no texto')).toBeInTheDocument()
    expect(within(panel()).getByText(/R\$\s?10,00 · Café/)).toBeInTheDocument()
  })
})
