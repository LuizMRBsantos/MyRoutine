import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { habitKeys } from '@/hooks/useHabits'
import { financeKeys } from '@/hooks/useFinance'
import { healthKeys } from '@/hooks/useHealth'
import type { Journal, RegisterJournalItem } from '@/types/journal'

export const journalKeys = {
  day: (date: string) => ['journal', date] as const,
}

const journalApi = {
  get: async (date: string): Promise<Journal> => (await api.get(`/journal/${date}`)).data,
  save: async (date: string, content: string): Promise<Journal> =>
    (await api.put(`/journal/${date}`, { content })).data,
  register: async (date: string, item: RegisterJournalItem): Promise<Journal> =>
    (await api.post(`/journal/${date}/items`, item)).data,
  unregister: async (date: string, sourceId: string): Promise<Journal> =>
    (await api.delete(`/journal/${date}/items/${sourceId}`)).data,
}

export function useJournal(date: string) {
  return useQuery({ queryKey: journalKeys.day(date), queryFn: () => journalApi.get(date) })
}

export function useSaveJournal(date: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (content: string) => journalApi.save(date, content),
    meta: { handlesError: true },
    onSuccess: (journal) => qc.setQueryData(journalKeys.day(date), journal),
  })
}

// Registrar/desfazer mexe em Finanças, Hábitos e Saúde: essas telas refazem a
// busca para mostrar (ou tirar) o que veio do diário.
function useJournalItemMutation<V>(date: string, fn: (v: V) => Promise<Journal>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    meta: { handlesError: true },
    onSuccess: (journal) => {
      qc.setQueryData(journalKeys.day(date), journal)
      for (const key of [habitKeys.all, financeKeys.all, healthKeys.all]) {
        void qc.invalidateQueries({ queryKey: key })
      }
    },
  })
}

export function useRegisterJournalItem(date: string) {
  return useJournalItemMutation(date, (item: RegisterJournalItem) => journalApi.register(date, item))
}

export function useUnregisterJournalItem(date: string) {
  return useJournalItemMutation(date, (sourceId: string) => journalApi.unregister(date, sourceId))
}
