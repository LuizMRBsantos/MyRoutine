import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { Invite, CreatedInvite } from '@/types/invite'

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
