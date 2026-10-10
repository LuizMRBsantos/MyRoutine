import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { AxiosAdapter } from 'axios'
import api from '@/services/api'
import { useCheckInDock } from '@/store/checkInDock'
import type { Habit } from '@/types/habit'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TodayHabits } from './TodayHabits'

function renderWith(props: { isError?: boolean }) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <TodayHabits habits={[]} isLoading={false} {...props} />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe('TodayHabits', () => {
  it('lista vazia de verdade convida a criar um hábito (sem recarregar a página)', () => {
    renderWith({})
    expect(screen.getByText(/Nenhum hábito para hoje/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Criar hábito/ })).toHaveAttribute('href', '/habits')
  })

  it('falha ao carregar não finge que os hábitos sumiram', () => {
    renderWith({ isError: true })
    expect(screen.getByRole('status')).toHaveTextContent('Não deu para carregar seus hábitos agora')
    expect(screen.queryByText(/Nenhum hábito para hoje/)).not.toBeInTheDocument()
  })
})

function habit(over: Partial<Habit>): Habit {
  return {
    id: 'h', name: 'Hábito', description: '', icon: '⭐', color: '#0071E3', frequency: 'daily',
    target_days: [1, 2, 3, 4, 5, 6, 7], time_of_day: 'anytime', category: 'general', is_active: true,
    created_at: '', current_streak: 0, completed_today: false, scheduled_today: true, check_type: 'simple',
    ...over,
  }
}

const posts: string[] = []
const adapter: AxiosAdapter = async (config) => {
  posts.push(`${config.method} ${config.url}`)
  return { status: 201, statusText: 'OK', data: {}, headers: {}, config }
}
const original = api.defaults.adapter

describe('TodayHabits — só o dia e check-in que precisa de mais que um toque', () => {
  beforeEach(() => { posts.length = 0; api.defaults.adapter = adapter; useCheckInDock.setState({ habitId: null }) })
  afterEach(() => { api.defaults.adapter = original })

  function renderHabits(habits: Habit[]) {
    return render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <TodayHabits habits={habits} isLoading={false} />
        </MemoryRouter>
      </QueryClientProvider>
    )
  }

  it('esconde hábitos que não são de hoje', () => {
    renderHabits([
      habit({ id: 'run', name: 'Correr', scheduled_today: false }),
      habit({ id: 'read', name: 'Ler' }),
    ])
    expect(screen.getByText('Ler')).toBeInTheDocument()
    expect(screen.queryByText('Correr')).not.toBeInTheDocument()
  })

  it('sem nenhum hábito hoje não convida a criar outro', () => {
    renderHabits([habit({ name: 'Correr', scheduled_today: false })])
    expect(screen.getByText('Nenhum hábito programado para hoje.')).toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })

  it('hábito com timer abre a janelinha em vez de registrar direto', async () => {
    const user = userEvent.setup()
    renderHabits([habit({ id: 'read', name: 'Leitura', check_type: 'timed', timer_minutes: 20 })])
    await user.click(document.getElementById('dashboard-checkin-read')!)
    expect(useCheckInDock.getState().habitId).toBe('read')
    expect(posts).toEqual([])
  })

  it('hábito simples registra com um toque', async () => {
    const user = userEvent.setup()
    renderHabits([habit({ id: 'water', name: 'Água' })])
    await user.click(document.getElementById('dashboard-checkin-water')!)
    expect(posts).toEqual(['post /habits/water/checkin'])
    expect(useCheckInDock.getState().habitId).toBeNull()
  })
})

