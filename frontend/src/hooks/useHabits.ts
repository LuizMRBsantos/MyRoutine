import api from '@/services/api'
import type { Habit, HabitLog, HeatmapEntry, HabitStats, CreateHabitInput, CheckInInput } from '@/types/habit'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

// ─── Query Keys ──────────────────────────────────────────────────────────────
export const habitKeys = {
  all: ['habits'] as const,
  list: () => [...habitKeys.all, 'list'] as const,
  detail: (id: string) => [...habitKeys.all, 'detail', id] as const,
  logs: (id: string) => [...habitKeys.all, 'logs', id] as const,
  stats: () => [...habitKeys.all, 'stats'] as const,
  heatmap: () => [...habitKeys.all, 'heatmap'] as const,
}

// ─── API Calls ────────────────────────────────────────────────────────────────
const habitsApi = {
  list: async (): Promise<Habit[]> => {
    const { data } = await api.get('/habits')
    return data.habits
  },

  create: async (input: CreateHabitInput): Promise<Habit> => {
    const { data } = await api.post('/habits', input)
    return data
  },

  update: async (id: string, input: Partial<CreateHabitInput>): Promise<Habit> => {
    const { data } = await api.put(`/habits/${id}`, input)
    return data
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/habits/${id}`)
  },

  checkIn: async (id: string, input: CheckInInput & { date?: string }): Promise<HabitLog> => {
    const { data } = await api.post(`/habits/${id}/checkin`, input)
    return data
  },

  undoCheckIn: async (id: string, date?: string): Promise<void> => {
    const params = date ? `?date=${date}` : ''
    await api.delete(`/habits/${id}/checkin${params}`)
  },

  getLogs: async (id: string, from?: string, to?: string): Promise<HabitLog[]> => {
    const { data } = await api.get(`/habits/${id}/logs`, { params: { from, to } })
    return data.logs
  },

  getStats: async (): Promise<HabitStats> => {
    const { data } = await api.get('/habits/stats')
    return data
  },

  getHeatmap: async (): Promise<HeatmapEntry[]> => {
    const { data } = await api.get('/habits/heatmap')
    return data.heatmap
  },
}

// ─── Hooks ────────────────────────────────────────────────────────────────────
export function useHabits() {
  return useQuery({
    queryKey: habitKeys.list(),
    queryFn: habitsApi.list,
  })
}

export function useHabitStats() {
  return useQuery({
    queryKey: habitKeys.stats(),
    queryFn: habitsApi.getStats,
  })
}

export function useHeatmap() {
  return useQuery({
    queryKey: habitKeys.heatmap(),
    queryFn: habitsApi.getHeatmap,
  })
}

export function useCreateHabit() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: habitsApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.list() })
      queryClient.invalidateQueries({ queryKey: habitKeys.stats() })
    },
  })
}

export function useUpdateHabit() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Partial<CreateHabitInput> }) =>
      habitsApi.update(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.all })
    },
  })
}

export function useHabitLogs(id: string | undefined, from?: string, to?: string) {
  return useQuery({
    queryKey: [...habitKeys.logs(id ?? ''), from, to],
    queryFn: () => habitsApi.getLogs(id!, from, to),
    enabled: !!id,
  })
}

export function useDeleteHabit() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: habitsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.all })
    },
  })
}

export function useCheckIn() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: { id: string } & CheckInInput & { date?: string }) =>
      habitsApi.checkIn(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.list() })
      queryClient.invalidateQueries({ queryKey: habitKeys.stats() })
      queryClient.invalidateQueries({ queryKey: habitKeys.heatmap() })
    },
  })
}

export function useUndoCheckIn() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, date }: { id: string; date?: string }) =>
      habitsApi.undoCheckIn(id, date),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: habitKeys.list() })
      queryClient.invalidateQueries({ queryKey: habitKeys.stats() })
    },
  })
}
