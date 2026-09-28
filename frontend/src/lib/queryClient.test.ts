import { describe, it, expect } from 'vitest'
import { queryClient } from './queryClient'
import { useAuthStore } from '@/store/authStore'

describe('queryClient', () => {
  it('logout limpa o cache do Query', () => {
    useAuthStore.setState({
      accessToken: 'a', refreshToken: 'r',
      user: { id: '1', name: 'Luiz', email: 'l@x.com', createdAt: '' },
      isAuthenticated: true,
    })
    queryClient.setQueryData(['habits'], [{ id: 'h1' }])
    expect(queryClient.getQueryData(['habits'])).toBeDefined()

    useAuthStore.getState().logout()

    expect(queryClient.getQueryData(['habits'])).toBeUndefined()
    expect(queryClient.getQueryCache().getAll()).toHaveLength(0)
  })
})
