import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { financeKeys } from '@/hooks/useFinance'
import type {
  ImportPreview, ImportBatch, ImportEntry, CSVMapping, CategoryRule,
} from '@/types/import'

export const importKeys = {
  all: ['imports'] as const,
  pending: () => [...importKeys.all, 'pending'] as const,
  batches: () => [...importKeys.all, 'batches'] as const,
  rules: () => [...importKeys.all, 'rules'] as const,
}

const importApi = {
  preview: async (content: string): Promise<ImportPreview> => {
    const { data } = await api.post('/finance/imports/preview', { content })
    return data
  },

  create: async (input: {
    filename: string
    content: string
    mapping: CSVMapping
    credit_card_id?: string
  }): Promise<ImportBatch> => {
    const { data } = await api.post('/finance/imports', input)
    return data
  },

  listPending: async (): Promise<ImportEntry[]> => {
    const { data } = await api.get('/finance/imports/pending')
    return data.entries
  },

  approve: async (id: string, category: string): Promise<void> => {
    await api.post(`/finance/imports/entries/${id}/approve`, { category })
  },

  decide: async (id: string, status: 'ignored' | 'matched'): Promise<void> => {
    await api.post(`/finance/imports/entries/${id}/decide`, { status })
  },

  listRules: async (): Promise<CategoryRule[]> => {
    const { data } = await api.get('/finance/rules')
    return data.rules
  },

  deleteRule: async (id: string): Promise<void> => {
    await api.delete(`/finance/rules/${id}`)
  },
}

export function usePendingEntries() {
  return useQuery({
    queryKey: importKeys.pending(),
    queryFn: importApi.listPending,
  })
}

export function useCategoryRules() {
  return useQuery({
    queryKey: importKeys.rules(),
    queryFn: importApi.listRules,
  })
}

export function usePreviewImport() {
  return useMutation({ mutationFn: importApi.preview })
}

export function useCreateImport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: importApi.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: importKeys.all }),
  })
}

export function useApproveEntry() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, category }: { id: string; category: string }) =>
      importApi.approve(id, category),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: importKeys.all })
      // Aprovar cria transação — o resumo e os gráficos mudam.
      qc.invalidateQueries({ queryKey: financeKeys.all })
    },
  })
}

export function useDecideEntry() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: 'ignored' | 'matched' }) =>
      importApi.decide(id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: importKeys.all }),
  })
}

export function useDeleteRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: importApi.deleteRule,
    onSuccess: () => qc.invalidateQueries({ queryKey: importKeys.rules() }),
  })
}
