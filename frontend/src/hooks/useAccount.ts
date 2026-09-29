import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useAuthStore } from '@/store/authStore'
import type { Profile } from '@/types/account'

export const accountKeys = {
  profile: ['me'] as const,
}

export function useProfile() {
  return useQuery({
    queryKey: accountKeys.profile,
    queryFn: async (): Promise<Profile> => (await api.get('/me')).data,
  })
}

export function useUpdateProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (input: { name?: string; timezone?: string }): Promise<Profile> =>
      (await api.patch('/me', input)).data,
    meta: { handlesError: true },
    onSuccess: (profile) => {
      qc.setQueryData(accountKeys.profile, profile)
      // O nome aparece no menu lateral, que lê do store de auth.
      useAuthStore.setState(s => (s.user ? { user: { ...s.user, name: profile.name } } : {}))
    },
  })
}

export function useChangePassword() {
  return useMutation({
    mutationFn: async (input: { current_password: string; new_password: string }) => {
      await api.put('/me/password', input)
    },
    meta: { handlesError: true },
  })
}

export function useDeleteAccount() {
  return useMutation({
    mutationFn: async (password: string) => {
      await api.delete('/me', { data: { password } })
    },
    meta: { handlesError: true },
  })
}

// Baixa o arquivo com todos os dados (LGPD). O servidor manda o nome do
// arquivo no Content-Disposition; se não vier, usa um padrão com a data.
export async function downloadMyData(): Promise<void> {
  const res = await api.get('/me/export', { responseType: 'blob' })
  const disposition = String(res.headers['content-disposition'] ?? '')
  const match = /filename="([^"]+)"/.exec(disposition)
  const filename = match?.[1] ?? `myroutine-dados-${new Date().toISOString().slice(0, 10)}.json`

  const url = URL.createObjectURL(res.data as Blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
