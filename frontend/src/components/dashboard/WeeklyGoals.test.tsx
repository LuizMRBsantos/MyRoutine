import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { AxiosAdapter } from 'axios'
import api from '@/services/api'
import { WeeklyGoals } from './WeeklyGoals'

let goals: { id: string; title: string; week: string; done: boolean; created_at: string }[] = []
const calls: string[] = []

const adapter: AxiosAdapter = async (config) => {
  const ok = (data: unknown, status = 200) => ({ status, statusText: 'OK', data, headers: {}, config })
  const body = config.data ? JSON.parse(String(config.data)) : {}
  calls.push(`${config.method} ${config.url}`)
  if (config.method === 'get') return ok(goals)
  if (config.method === 'post') {
    const goal = { id: 'g1', title: body.title, week: '2026-10-05', done: false, created_at: '' }
    goals = [...goals, goal]
    return ok(goal, 201)
  }
  if (config.method === 'patch') {
    goals = goals.map(g => (g.id === 'g1' ? { ...g, done: body.done } : g))
    return ok(goals[0])
  }
  if (config.method === 'delete') {
    goals = []
    return ok('', 204)
  }
  throw new Error(`unexpected ${config.method} ${config.url}`)
}

const original = api.defaults.adapter
beforeEach(() => { goals = []; calls.length = 0; api.defaults.adapter = adapter })
afterEach(() => { api.defaults.adapter = original })

describe('WeeklyGoals', () => {
  it('adiciona, marca como feita e apaga uma meta da semana', async () => {
    const user = userEvent.setup()
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <WeeklyGoals />
      </QueryClientProvider>
    )

    expect(await screen.findByText('O que você quer fazer esta semana?')).toBeInTheDocument()

    await user.type(screen.getByLabelText('Nova meta da semana'), 'Lista 3 de C2')
    await user.click(screen.getByRole('button', { name: 'Adicionar' }))
    const box = await screen.findByRole('checkbox', { name: 'Lista 3 de C2' })
    expect(screen.getByText('Semana de 5 a 11 de out.')).toBeInTheDocument()
    expect(screen.getByLabelText('Nova meta da semana')).toHaveValue('')

    await user.click(box)
    await waitFor(() => expect(box).toBeChecked())
    expect(calls).toContain('patch /weekly-goals/g1')

    await user.click(screen.getByRole('button', { name: 'Apagar meta: Lista 3 de C2' }))
    await waitFor(() => expect(screen.queryByRole('checkbox')).not.toBeInTheDocument())
  })
})
