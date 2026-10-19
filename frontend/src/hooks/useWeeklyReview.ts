import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

export interface MissedDay {
  habit_id: string
  habit_name: string
  habit_icon: string
  habit_color: string
  date: string
  review?: {
    id: string
    status: 'migrated' | 'discarded'
    reviewed_at: string
  }
}

const reviewApi = {
  getMissedDays: async (): Promise<MissedDay[]> => {
    const { data } = await api.get('/reviews/missed')
    return data.missed_days
  },

  reviewDay: async (habitId: string, reviewDate: string, status: 'migrated' | 'discarded') => {
    const { data } = await api.post(`/habits/${habitId}/review`, {
      review_date: reviewDate,
      status,
    })
    return data
  },
}

export const reviewKeys = {
  missed: () => ['reviews', 'missed'] as const,
}

export function useMissedDays() {
  return useQuery({
    queryKey: reviewKeys.missed(),
    queryFn: reviewApi.getMissedDays,
  })
}

export function useReviewDay() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      habitId,
      reviewDate,
      status,
    }: {
      habitId: string
      reviewDate: string
      status: 'migrated' | 'discarded'
    }) => reviewApi.reviewDay(habitId, reviewDate, status),
    // A tela mostra a própria mensagem (em português) quando falha.
    meta: { handlesError: true },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: reviewKeys.missed() })
    },
  })
}

export type ReviewDecision = { habitId: string; reviewDate: string; status: 'migrated' | 'discarded' }

// Salva várias decisões de uma vez ("Salvar revisão"); os dias decididos
// saem da lista.
export function useSaveReview() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (decisions: ReviewDecision[]) =>
      Promise.all(decisions.map(d => reviewApi.reviewDay(d.habitId, d.reviewDate, d.status))),
    meta: { handlesError: true },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: reviewKeys.missed() })
    },
  })
}

