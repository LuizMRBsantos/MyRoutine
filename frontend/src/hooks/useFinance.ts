import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type {
  Transaction, CreateTransactionInput, Budget, FinanceSummary,
} from '@/types/finance'

// ─── Query Keys ──────────────────────────────────────────────────────────────
export const financeKeys = {
  all: ['finance'] as const,
  transactions: (from: string, to: string, category?: string) =>
    [...financeKeys.all, 'transactions', from, to, category ?? ''] as const,
  summary: (month: string) => [...financeKeys.all, 'summary', month] as const,
  budgets: (month: string) => [...financeKeys.all, 'budgets', month] as const,
}

// ─── API ─────────────────────────────────────────────────────────────────────
const financeApi = {
  listTransactions: async (from: string, to: string, category?: string): Promise<Transaction[]> => {
    const { data } = await api.get('/finance/transactions', { params: { from, to, category } })
    return data.transactions
  },

  createTransaction: async (input: CreateTransactionInput): Promise<Transaction> => {
    const { data } = await api.post('/finance/transactions', input)
    return data
  },

  deleteTransaction: async (id: string): Promise<void> => {
    await api.delete(`/finance/transactions/${id}`)
  },

  getSummary: async (month: string): Promise<FinanceSummary> => {
    const { data } = await api.get('/finance/summary', { params: { month } })
    return data
  },

  listBudgets: async (month: string): Promise<Budget[]> => {
    const { data } = await api.get('/finance/budgets', { params: { month } })
    return data.budgets
  },

  upsertBudget: async (input: { category: string; month: string; amount_cents: number }): Promise<Budget> => {
    const { data } = await api.put('/finance/budgets', input)
    return data
  },

  deleteBudget: async (id: string): Promise<void> => {
    await api.delete(`/finance/budgets/${id}`)
  },
}

// ─── Hooks ────────────────────────────────────────────────────────────────────

export function useTransactions(from: string, to: string, category?: string) {
  return useQuery({
    queryKey: financeKeys.transactions(from, to, category),
    queryFn: () => financeApi.listTransactions(from, to, category),
    enabled: !!from && !!to,
  })
}

export function useFinanceSummary(month: string) {
  return useQuery({
    queryKey: financeKeys.summary(month),
    queryFn: () => financeApi.getSummary(month),
    enabled: !!month,
  })
}

export function useBudgets(month: string) {
  return useQuery({
    queryKey: financeKeys.budgets(month),
    queryFn: () => financeApi.listBudgets(month),
    enabled: !!month,
  })
}

export function useCreateTransaction() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: financeApi.createTransaction,
    onSuccess: () => qc.invalidateQueries({ queryKey: financeKeys.all }),
  })
}

export function useDeleteTransaction() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: financeApi.deleteTransaction,
    onSuccess: () => qc.invalidateQueries({ queryKey: financeKeys.all }),
  })
}

export function useUpsertBudget() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: financeApi.upsertBudget,
    onSuccess: () => qc.invalidateQueries({ queryKey: financeKeys.all }),
  })
}

export function useDeleteBudget() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: financeApi.deleteBudget,
    onSuccess: () => qc.invalidateQueries({ queryKey: financeKeys.all }),
  })
}
