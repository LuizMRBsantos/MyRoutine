import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { AxiosAdapter } from 'axios'
import api from '@/services/api'
import { HabitConsistency } from './HabitConsistency'
import { TodayAgenda } from './TodayAgenda'
import { WeekInOneLine } from './WeekInOneLine'

const days = ' '.repeat(80) + 'xo-xo-oxo.' // 90 days: seg x, ter o, qua (hoje) ainda em aberto
const consistency = {
  from: '2029-12-14', to: '2030-03-13',
  habits: [{ id: 'h1', name: 'Leitura', icon: '📖', color: '#A99AD6', days, done: 4, scheduled: 7, previous_done: 0, previous_scheduled: 0 }],
}
const tasks = [
  { id: 't1', title: 'Aula de Cálculo 2', date: '2030-03-13', start_time: '09:00', duration_minutes: 90, status: 'planned', notify: false, category: 'study' },
  { id: 't2', title: 'Reunião com orientador', date: '2030-03-13', start_time: '14:00', duration_minutes: 60, status: 'planned', notify: true, category: 'work' },
  { id: 't3', title: 'Comprar pão', date: '2030-03-13', status: 'planned', notify: false, category: 'other' },
]

const adapter: AxiosAdapter = async (config) => {
  const data = config.url === '/habits/consistency' ? consistency : config.url === '/tasks' ? { tasks } : {}
  return { status: 200, statusText: 'OK', data, headers: {}, config }
}
const original = api.defaults.adapter

beforeEach(() => {
  api.defaults.adapter = adapter
  vi.useFakeTimers({ shouldAdvanceTime: true })
  vi.setSystemTime(new Date(2030, 2, 13, 13, 20))
})
afterEach(() => { api.defaults.adapter = original; vi.useRealTimers() })

const wrap = (ui: React.ReactNode) => render(
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    <MemoryRouter>{ui}</MemoryRouter>
  </QueryClientProvider>
)

describe('Início — blocos novos', () => {
  it('constância mostra contagens concretas e um quadrinho por dia, sem porcentagem', async () => {
    const { container } = wrap(<HabitConsistency />)
    const strip = await screen.findByRole('img', { name: /Leitura: 4 de 7 dias programados nos últimos 90 dias; hábito novo/ })
    expect(strip.children).toHaveLength(90)
    expect(container.textContent).not.toMatch(/%/)
  })

  it('agenda destaca o próximo compromisso com quanto falta', async () => {
    wrap(<TodayAgenda />)
    expect(await screen.findByText('Reunião com orientador')).toBeInTheDocument()
    expect(screen.getByText(/em 40 min · aviso ligado/)).toBeInTheDocument()
    expect(screen.getByText('sem horário')).toBeInTheDocument()
  })

  it('a semana em uma linha vai de segunda até hoje e deixa o resto em aberto', async () => {
    wrap(<WeekInOneLine />)
    expect(await screen.findByRole('cell', { name: 'seg: feito' })).toBeInTheDocument()
    expect(screen.getByRole('cell', { name: 'qua: hoje, ainda em aberto' })).toBeInTheDocument()
    expect(screen.getByRole('cell', { name: 'dom: ainda não chegou' })).toBeInTheDocument()
  })
})
