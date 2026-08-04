import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type {
  Task, MonthlyGoal,
  CreateTaskInput, UpdateTaskInput, CreateGoalInput,
} from '@/types/task'

// ─── Query Keys ──────────────────────────────────────────────────────────────
export const taskKeys = {
  all: ['tasks'] as const,
  byDate: (date: string) => [...taskKeys.all, 'date', date] as const,
  byWeek: (start: string, end: string) => [...taskKeys.all, 'week', start, end] as const,
  goals: (month: string) => ['goals', month] as const,
}

// ─── API ─────────────────────────────────────────────────────────────────────
const tasksApi = {
  listByDate: async (date: string): Promise<Task[]> => {
    const { data } = await api.get('/tasks', { params: { date } })
    return data
  },

  listByWeek: async (start: string, end: string): Promise<Task[]> => {
    const { data } = await api.get('/tasks/week', { params: { start, end } })
    return data
  },

  create: async (input: CreateTaskInput): Promise<Task> => {
    const { data } = await api.post('/tasks', input)
    return data
  },

  update: async (id: string, input: UpdateTaskInput): Promise<Task> => {
    const { data } = await api.patch(`/tasks/${id}`, input)
    return data
  },

  advanceStatus: async (id: string): Promise<Task> => {
    const { data } = await api.post(`/tasks/${id}/advance`)
    return data
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/tasks/${id}`)
  },
}

const goalsApi = {
  listByMonth: async (month: string): Promise<MonthlyGoal[]> => {
    const { data } = await api.get('/goals', { params: { month } })
    return data
  },

  create: async (input: CreateGoalInput): Promise<MonthlyGoal> => {
    const { data } = await api.post('/goals', input)
    return data
  },

  updateStatus: async (id: string, status: string): Promise<MonthlyGoal> => {
    const { data } = await api.patch(`/goals/${id}/status`, { status })
    return data
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/goals/${id}`)
  },
}

// ─── Task Hooks ───────────────────────────────────────────────────────────────

export function useTasks(date: string) {
  return useQuery({
    queryKey: taskKeys.byDate(date),
    queryFn: () => tasksApi.listByDate(date),
    enabled: !!date,
  })
}

export function useWeekTasks(start: string, end: string) {
  return useQuery({
    queryKey: taskKeys.byWeek(start, end),
    queryFn: () => tasksApi.listByWeek(start, end),
    enabled: !!start && !!end,
  })
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : `${n}`
}

export function useMonthTasks(date: Date) {
  const y = date.getFullYear()
  const m = date.getMonth()
  const start = `${y}-${pad(m + 1)}-01`
  const lastDay = new Date(y, m + 1, 0).getDate()
  const end = `${y}-${pad(m + 1)}-${pad(lastDay)}`

  return useQuery({
    queryKey: ['tasks', 'monthRange', start, end] as const,
    queryFn: () => tasksApi.listByWeek(start, end),
    enabled: !!start && !!end,
  })
}

export function useCreateTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: tasksApi.create,
    onSuccess: (task) => {
      qc.invalidateQueries({ queryKey: taskKeys.byDate(task.date) })
      qc.invalidateQueries({ queryKey: taskKeys.all })
    },
  })
}

export function useUpdateTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateTaskInput }) =>
      tasksApi.update(id, input),
    onSuccess: (task) => {
      qc.invalidateQueries({ queryKey: taskKeys.byDate(task.date) })
      qc.invalidateQueries({ queryKey: taskKeys.all })
    },
  })
}

export function useAdvanceTaskStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => tasksApi.advanceStatus(id),
    onSuccess: (task) => {
      // Atualiza o cache otimisticamente sem refetch completo
      qc.setQueryData<Task[]>(taskKeys.byDate(task.date), (old) =>
        old?.map((t) => (t.id === task.id ? task : t)) ?? []
      )
    },
  })
}

export function useDeleteTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: tasksApi.delete,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: taskKeys.all })
    },
  })
}

// ─── Goal Hooks ───────────────────────────────────────────────────────────────

export function useMonthlyGoals(month: string) {
  return useQuery({
    queryKey: taskKeys.goals(month),
    queryFn: () => goalsApi.listByMonth(month),
    enabled: !!month,
  })
}

export function useCreateGoal() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: goalsApi.create,
    onSuccess: (goal) => {
      qc.invalidateQueries({ queryKey: taskKeys.goals(goal.month) })
    },
  })
}

export function useUpdateGoalStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      goalsApi.updateStatus(id, status),
    onSuccess: (goal) => {
      qc.invalidateQueries({ queryKey: taskKeys.goals(goal.month) })
    },
  })
}

export function useDeleteGoal() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: goalsApi.delete,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['goals'] })
    },
  })
}
