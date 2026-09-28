import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createBrowserRouter } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { DashboardPage } from '@/pages/DashboardPage'
import { HabitsPage } from '@/pages/HabitsPage'
import { PlannerPage } from '@/pages/PlannerPage'
import { FinancePage } from '@/pages/FinancePage'
import { ImportPage } from '@/pages/ImportPage'
import { HealthPage } from '@/pages/HealthPage'
import { StudyPage } from '@/pages/StudyPage'
import { InvitesPage } from '@/pages/InvitesPage'
import { LoginPage } from '@/pages/LoginPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { ProtectedRoute } from '@/components/auth/ProtectedRoute'
import { Toasts } from '@/components/ui/Toasts'
import { queryClient } from '@/lib/queryClient'

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
        path: '/finance/import',
        element: <ImportPage />,
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
        path: '/convites',
        element: <InvitesPage />,
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
