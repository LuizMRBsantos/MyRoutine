import api from '@/services/api'
import { useQuery } from '@tanstack/react-query'
import { habitKeys } from '@/hooks/useHabits'

// Constância por hábito nos últimos 90 dias (GET /habits/consistency).
// Um caractere por dia, do mais antigo até hoje:
//   x feito · o programado sem registro · - não é dia do hábito
//   . hoje, ainda em aberto · ' ' antes de o hábito existir
export interface HabitConsistency {
  id: string
  name: string
  icon: string
  color: string
  days: string
  done: number
  scheduled: number
  previous_done: number
  previous_scheduled: number
}

export interface Consistency {
  from: string
  to: string
  habits: HabitConsistency[]
}

export function useConsistency() {
  return useQuery({
    queryKey: habitKeys.consistency(),
    queryFn: async (): Promise<Consistency> => (await api.get('/habits/consistency')).data,
  })
}
