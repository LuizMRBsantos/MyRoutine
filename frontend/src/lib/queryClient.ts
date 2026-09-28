import { MutationCache, QueryClient } from '@tanstack/react-query'
import { toast, apiErrorMessage } from '@/lib/toast'
import { useAuthStore } from '@/store/authStore'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60, // 1 min
      retry: 1,
    },
  },
  // Toda mutation que falhar sem tratamento local vira um toast —
  // nenhuma ação do usuário falha em silêncio.
  mutationCache: new MutationCache({
    onError: (error) => {
      toast.error(apiErrorMessage(error, 'Algo deu errado. Tente novamente.'))
    },
  }),
})

// No logout (manual ou por renovação que falhou), descarta todo o cache:
// outro usuário na mesma aba não pode ver dados do anterior.
useAuthStore.subscribe((state, prev) => {
  if (prev.isAuthenticated && !state.isAuthenticated) {
    queryClient.clear()
  }
})
