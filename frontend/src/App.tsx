import { MutationCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createBrowserRouter } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { DashboardPage } from '@/pages/DashboardPage'
import { HabitsPage } from '@/pages/HabitsPage'
import { PlannerPage } from '@/pages/PlannerPage'
import { FinancePage } from '@/pages/FinancePage'
import { HealthPage } from '@/pages/HealthPage'
import { StudyPage } from '@/pages/StudyPage'
import { LoginPage } from '@/pages/LoginPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { ProtectedRoute } from '@/components/auth/ProtectedRoute'
import { Toasts } from '@/components/ui/Toasts'
import { toast, apiErrorMessage } from '@/lib/toast'

const queryClient = new QueryClient({
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

const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/register',
    element: <RegisterPage />,
  },
  {
    element: (
      <ProtectedRoute>
        <AppLayout />
      </ProtectedRoute>
    ),
    children: [
      {
        path: '/',
        element: <DashboardPage />,
      },
      {
        path: '/habits',
        element: <HabitsPage />,
      },
      {
        path: '/planner',
        element: <PlannerPage />,
      },
      {
        path: '/finance',
        element: <FinancePage />,
      },
      {
        path: '/health',
        element: <HealthPage />,
      },
      {
        path: '/studies',
        element: <StudyPage />,
      },
      {
        path: '*',
        element: <NotFoundPage />,
      },
    ],
  },
])

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      <Toasts />
    </QueryClientProvider>
  )
}
