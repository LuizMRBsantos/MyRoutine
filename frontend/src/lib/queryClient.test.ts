import { describe, it, expect, beforeEach } from 'vitest'
import { MutationObserver } from '@tanstack/react-query'
import { queryClient } from './queryClient'
import { useToastStore } from './toast'
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

describe('queryClient — aviso global de erro', () => {
  const failing = () => Promise.reject({ response: { data: { error: 'email already registered' } } })

  beforeEach(() => useToastStore.setState({ toasts: [] }))

  it('mostra o erro do servidor quando a tela não trata', async () => {
    await new MutationObserver(queryClient, { mutationFn: failing }).mutate().catch(() => {})
    expect(useToastStore.getState().toasts.map(t => t.message)).toEqual(['email already registered'])
  })

  it('fica em silêncio quando a tela trata o próprio erro', async () => {
    await new MutationObserver(queryClient, { mutationFn: failing, meta: { handlesError: true } })
      .mutate().catch(() => {})
    expect(useToastStore.getState().toasts).toEqual([])
  })
})
