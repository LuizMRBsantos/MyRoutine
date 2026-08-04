import api from '@/services/api'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type { StudySession, CreateStudySessionInput, StudySummary } from '@/types/study'
import { habitKeys } from '@/hooks/useHabits'

export const studyKeys = {
  all: ['study'] as const,
  sessions: (from: string, to: string) => [...studyKeys.all, 'sessions', from, to] as const,
  summary: () => [...studyKeys.all, 'summary'] as const,
}

const studyApi = {
  listSessions: async (from: string, to: string): Promise<StudySession[]> => {
    const { data } = await api.get('/study/sessions', { params: { from, to } })
    return data.sessions
  },

  createSession: async (input: CreateStudySessionInput): Promise<StudySession> => {
    const { data } = await api.post('/study/sessions', input)
    return data
  },

  deleteSession: async (id: string): Promise<void> => {
    await api.delete(`/study/sessions/${id}`)
  },

  getSummary: async (): Promise<StudySummary> => {
    const { data } = await api.get('/study/summary')
    return data
  },
}

export function useStudySessions(from: string, to: string) {
  return useQuery({
    queryKey: studyKeys.sessions(from, to),
    queryFn: () => studyApi.listSessions(from, to),
  })
}

export function useStudySummary() {
  return useQuery({
    queryKey: studyKeys.summary(),
    queryFn: studyApi.getSummary,
  })
}

export function useCreateStudySession() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: studyApi.createSession,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: studyKeys.all })
      // Uma sessão vinculada a hábito gera check-in — atualiza o dashboard
      qc.invalidateQueries({ queryKey: habitKeys.all })
    },
  })
}

export function useDeleteStudySession() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: studyApi.deleteSession,
    onSuccess: () => qc.invalidateQueries({ queryKey: studyKeys.all }),
  })
}
