import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

export interface WeeklyGoal {
  id: string
  title: string
  week: string // segunda-feira, "YYYY-MM-DD"
  done: boolean
  created_at: string
}

// Sem data: o servidor usa a semana de hoje no fuso da pessoa.
const weeklyGoalKeys = { current: ['weekly-goals', 'current'] as const }

export function useWeeklyGoals() {
  return useQuery({
    queryKey: weeklyGoalKeys.current,
    queryFn: async (): Promise<WeeklyGoal[]> => (await api.get('/weekly-goals')).data,
  })
}

export function useCreateWeeklyGoal() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (title: string): Promise<WeeklyGoal> =>
      (await api.post('/weekly-goals', { title })).data,
    onSuccess: (goal) =>
      qc.setQueryData<WeeklyGoal[]>(weeklyGoalKeys.current, (old = []) => [...old, goal]),
  })
}

export function useToggleWeeklyGoal() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, done }: { id: string; done: boolean }): Promise<WeeklyGoal> =>
      (await api.patch(`/weekly-goals/${id}`, { done })).data,
    onSuccess: (goal) =>
      qc.setQueryData<WeeklyGoal[]>(weeklyGoalKeys.current, (old = []) =>
        old.map(g => (g.id === goal.id ? goal : g))),
  })
}

export function useDeleteWeeklyGoal() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/weekly-goals/${id}`)
      return id
    },
    onSuccess: (id) =>
      qc.setQueryData<WeeklyGoal[]>(weeklyGoalKeys.current, (old = []) => old.filter(g => g.id !== id)),
  })
}
