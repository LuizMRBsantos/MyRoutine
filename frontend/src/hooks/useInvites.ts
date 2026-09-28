import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { Invite, CreatedInvite, CreatedReset } from '@/types/invite'

export const inviteKeys = {
  all: ['invites'] as const,
}

const invitesApi = {
  list: async (): Promise<Invite[]> => {
    const { data } = await api.get('/admin/invites')
    return data.invites
  },
  create: async (email: string): Promise<CreatedInvite> => {
    const { data } = await api.post('/admin/invites', { email })
    return data
  },
  revoke: async (id: string): Promise<void> => {
    await api.delete(`/admin/invites/${id}`)
  },
  createReset: async (email: string): Promise<CreatedReset> => {
    const { data } = await api.post('/admin/password-resets', { email })
    return data
  },
}

export function useInvites() {
  return useQuery({ queryKey: inviteKeys.all, queryFn: invitesApi.list })
}

export function useCreateInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: invitesApi.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: inviteKeys.all }),
  })
}

export function useRevokeInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: invitesApi.revoke,
    onSuccess: () => qc.invalidateQueries({ queryKey: inviteKeys.all }),
  })
}

export function useCreatePasswordReset() {
  return useMutation({ mutationFn: invitesApi.createReset })
}
