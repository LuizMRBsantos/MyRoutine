import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { AxiosAdapter } from 'axios'
import api from '@/services/api'
import { useCheckInDock } from '@/store/checkInDock'
import { CheckInDock } from './CheckInDock'

const reading = {
  id: 'read', name: 'Leitura', description: '', icon: '📖', color: '#0071E3', frequency: 'daily',
  target_days: [1, 2, 3, 4, 5, 6, 7], time_of_day: 'anytime', category: 'general', is_active: true,
  created_at: '', current_streak: 0, completed_today: false, scheduled_today: true,
  check_type: 'timed', timer_minutes: 20,
}

const adapter: AxiosAdapter = async (config) =>
  ({ status: 200, statusText: 'OK', data: config.url === '/habits' ? { habits: [reading] } : {}, headers: {}, config })
const original = api.defaults.adapter

beforeEach(() => { api.defaults.adapter = adapter; localStorage.clear() })
afterEach(() => { api.defaults.adapter = original; useCheckInDock.setState({ habitId: null }) })

describe('CheckInDock', () => {
  it('mostra o timer do hábito escolhido e fecha no ×', async () => {
    const user = userEvent.setup()
    useCheckInDock.setState({ habitId: 'read' })
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <CheckInDock />
      </QueryClientProvider>
    )

    expect(await screen.findByRole('complementary', { name: 'Check-in: Leitura' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Iniciar/ })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Fechar' }))
    await waitFor(() => expect(screen.queryByRole('complementary')).not.toBeInTheDocument())
    expect(useCheckInDock.getState().habitId).toBeNull()
  })
})
