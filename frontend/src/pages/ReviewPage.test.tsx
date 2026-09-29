import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen, within, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { AxiosAdapter } from 'axios'
import api from '@/services/api'
import { ReviewPage } from './ReviewPage'

type Review = { id: string; status: 'migrated' | 'discarded'; reviewed_at: string }
let reviews: Record<string, Review> = {}
let days: { habit_id: string; habit_name: string; habit_icon: string; habit_color: string; date: string }[] = []
const posts: unknown[] = []

const adapter: AxiosAdapter = async (config) => {
  const ok = (data: unknown) => ({ status: 200, statusText: 'OK', data, headers: {}, config })
  if (config.url === '/reviews/missed') {
    return ok({ missed_days: days.map(d => ({ ...d, review: reviews[`${d.habit_id}|${d.date}`] })) })
  }
  const m = config.url?.match(/^\/habits\/(.+)\/review$/)
  if (m && config.method === 'post') {
    const body = JSON.parse(String(config.data))
    posts.push({ habit: m[1], ...body })
    reviews[`${m[1]}|${body.review_date}`] = { id: 'r', status: body.status, reviewed_at: '' }
    return ok({})
  }
  throw new Error(`unexpected ${config.method} ${config.url}`)
}

const originalAdapter = api.defaults.adapter

beforeEach(() => {
  reviews = {}
  posts.length = 0
  days = [
    { habit_id: 'h1', habit_name: 'Correr', habit_icon: '🏃', habit_color: '#34C759', date: '2026-09-28' },
    { habit_id: 'h2', habit_name: 'Ler', habit_icon: '📖', habit_color: '#0071E3', date: '2026-09-28' },
    { habit_id: 'h1', habit_name: 'Correr', habit_icon: '🏃', habit_color: '#34C759', date: '2026-09-26' },
  ]
  api.defaults.adapter = adapter
})

afterEach(() => {
  api.defaults.adapter = originalAdapter
})

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}><ReviewPage /></QueryClientProvider>)
}

describe('ReviewPage (revisão da semana)', () => {
  it('agrupa por dia e deixa decidir e mudar de ideia', async () => {
    const user = userEvent.setup()
    renderPage()

    const monday = await screen.findByRole('region', { name: /28 de setembro/ })
    expect(within(monday).getByText('Correr')).toBeInTheDocument()
    expect(within(monday).getByText('Ler')).toBeInTheDocument()
    expect(screen.getByRole('region', { name: /26 de setembro/ })).toBeInTheDocument()

    const correr = within(monday).getByRole('group', { name: 'Decisão para Correr' })
    await user.click(within(correr).getByRole('button', { name: '→ Migrar' }))
    await waitFor(() =>
      expect(within(correr).getByRole('button', { name: '→ Migrar' })).toHaveAttribute('aria-pressed', 'true')
    )

    await user.click(within(correr).getByRole('button', { name: '· Descartar' }))
    await waitFor(() =>
      expect(within(correr).getByRole('button', { name: '· Descartar' })).toHaveAttribute('aria-pressed', 'true')
    )
    expect(posts).toEqual([
      { habit: 'h1', review_date: '2026-09-28', status: 'migrated' },
      { habit: 'h1', review_date: '2026-09-28', status: 'discarded' },
    ])
  })

  it('não mostra contador de dias sem registro (habit-review-flow)', async () => {
    renderPage()
    await screen.findByRole('region', { name: /28 de setembro/ })
    expect(document.body.textContent).not.toMatch(/\d+\s+dias?\s+(sem|perdid|falh)/i)
    expect(screen.getByText(/nada vira falha/)).toBeInTheDocument()
  })

  it('semana sem buracos: frase neutra, sem comemoração', async () => {
    days = []
    renderPage()
    expect(await screen.findByText('Nenhum dia sem registro nos últimos 7 dias.')).toBeInTheDocument()
    expect(document.body.textContent).not.toMatch(/ótim|parab|incrível/i)
  })
})
