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
