import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

// O cliente HTTP é mockado: o que está sob teste é o desempacotamento do
// envelope — se o backend mudar de {habits: [...]} para array puro (ou
// vice-versa), a UI quebra em silêncio sem este teste.
const mockGet = vi.fn()
const mockPost = vi.fn()
const mockPut = vi.fn()

vi.mock('@/services/api', () => ({
  default: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
    put: (...args: unknown[]) => mockPut(...args),
    delete: vi.fn(),
  },
}))

const { useHabits, useHabitStats, useHeatmap, useHabitLogs } = await import('./useHabits')

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('useHabits', () => {
  it('desempacota o envelope {habits: [...]}', async () => {
    mockGet.mockResolvedValue({
      data: { habits: [{ id: '1', name: 'Correr', current_streak: 3 }] },
    })

    const { result } = renderHook(() => useHabits(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockGet).toHaveBeenCalledWith('/habits')
    expect(result.current.data).toHaveLength(1)
    expect(result.current.data![0].name).toBe('Correr')
  })

  it('propaga o estado de erro quando a API falha', async () => {
    mockGet.mockRejectedValue(new Error('network down'))

    const { result } = renderHook(() => useHabits(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

describe('useHabitStats', () => {
  it('lê stats sem envelope', async () => {
    mockGet.mockResolvedValue({
      data: { total_habits: 2, completed_today: 1, current_streak: 5, habit_stats: [] },
    })

    const { result } = renderHook(() => useHabitStats(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data!.current_streak).toBe(5)
  })
})

describe('useHeatmap', () => {
  it('desempacota o envelope {heatmap: [...]}', async () => {
    mockGet.mockResolvedValue({
      data: { heatmap: [{ date: '2026-08-05', habits_completed: 2 }] },
    })

    const { result } = renderHook(() => useHeatmap(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data![0].habits_completed).toBe(2)
  })
})

describe('useHabitLogs', () => {
  it('desempacota o envelope {logs: [...]} e envia o intervalo', async () => {
    mockGet.mockResolvedValue({
      data: { logs: [{ id: 'l1', logged_date: '2026-08-05', source_type: 'manual' }] },
    })

    const { result } = renderHook(
      () => useHabitLogs('habit-1', '2026-07-01', '2026-08-05'),
      { wrapper }
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockGet).toHaveBeenCalledWith('/habits/habit-1/logs', {
      params: { from: '2026-07-01', to: '2026-08-05' },
    })
    expect(result.current.data![0].source_type).toBe('manual')
  })

  it('não busca nada sem um id de hábito', () => {
    renderHook(() => useHabitLogs(undefined), { wrapper })
    expect(mockGet).not.toHaveBeenCalled()
  })
})
