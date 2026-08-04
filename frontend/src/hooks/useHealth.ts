import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { Activity, HealthSummary, BodyMetric } from '@/types/health'

export const healthKeys = {
  all: ['health'] as const,
  activities: (from: string, to: string) => [...healthKeys.all, 'activities', from, to] as const,
  summary: (weeks: number) => [...healthKeys.all, 'summary', weeks] as const,
  bodyMetrics: () => [...healthKeys.all, 'body-metrics'] as const,
}

const healthApi = {
  listActivities: async (from: string, to: string): Promise<Activity[]> => {
    const { data } = await api.get('/health-module/activities', { params: { from, to } })
    return data.activities
  },

  getSummary: async (weeks: number): Promise<HealthSummary> => {
    const { data } = await api.get('/health-module/summary', { params: { weeks } })
    return data
  },

  listBodyMetrics: async (): Promise<BodyMetric[]> => {
    const { data } = await api.get('/health-module/body-metrics')
    return data.body_metrics
  },

  upsertBodyMetric: async (input: { measured_on?: string; weight_kg?: number; notes?: string }): Promise<BodyMetric> => {
    const { data } = await api.post('/health-module/body-metrics', input)
    return data
  },
}

export function useActivities(from: string, to: string) {
  return useQuery({
    queryKey: healthKeys.activities(from, to),
    queryFn: () => healthApi.listActivities(from, to),
  })
}

export function useHealthSummary(weeks = 4) {
  return useQuery({
    queryKey: healthKeys.summary(weeks),
    queryFn: () => healthApi.getSummary(weeks),
  })
}

export function useBodyMetrics() {
  return useQuery({
    queryKey: healthKeys.bodyMetrics(),
    queryFn: healthApi.listBodyMetrics,
  })
}

export function useUpsertBodyMetric() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: healthApi.upsertBodyMetric,
    onSuccess: () => qc.invalidateQueries({ queryKey: healthKeys.bodyMetrics() }),
  })
}
