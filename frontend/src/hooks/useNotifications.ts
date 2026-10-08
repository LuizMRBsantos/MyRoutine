import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

export interface NotificationSettings {
  task_reminders: boolean
  task_lead_minutes: number
  morning_digest: boolean
  morning_time: string // "HH:MM"
  evening_digest: boolean
  evening_time: string
}

export interface NotificationView {
  settings: NotificationSettings
  devices: number
}

export interface PushConfig {
  enabled: boolean
  vapid_public_key?: string
}

export const notificationKeys = {
  config: ['notifications', 'config'] as const,
  settings: ['notifications', 'settings'] as const,
}

export function usePushConfig() {
  return useQuery({
    queryKey: notificationKeys.config,
    queryFn: async (): Promise<PushConfig> => (await api.get('/notifications/config')).data,
    staleTime: Infinity,
  })
}

export function useNotificationSettings() {
  return useQuery({
    queryKey: notificationKeys.settings,
    queryFn: async (): Promise<NotificationView> => (await api.get('/notifications/settings')).data,
  })
}

export function useUpdateNotificationSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (input: NotificationSettings): Promise<NotificationView> =>
      (await api.put('/notifications/settings', input)).data,
    meta: { handlesError: true },
    onSuccess: (view) => qc.setQueryData(notificationKeys.settings, view),
  })
}
