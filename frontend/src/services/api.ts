import axios from 'axios'
import { useAuthStore } from '@/store/authStore'
import { refreshSession } from './refreshSession'

// API client com interceptors para JWT automático
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Endpoints em que um 401 é resposta de negócio (credencial errada), não
// sessão expirada — não faz sentido tentar renovar.
const NO_REFRESH_URLS = ['/auth/login', '/auth/register', '/auth/refresh']

// Request interceptor — injeta Bearer token automaticamente
api.interceptors.request.use((config) => {
  const { accessToken } = useAuthStore.getState()
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`
  }
  return config
})

// Response interceptor — em 401, espera a renovação compartilhada e repete a
// request uma única vez com o novo access token.
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config

    if (
      error.response?.status !== 401 ||
      !original ||
      original._retry ||
      NO_REFRESH_URLS.some((url) => original.url?.endsWith(url))
    ) {
      return Promise.reject(error)
    }
    original._retry = true

    // A request pode ter saído com um token antigo e voltado depois que outra
    // renovação já terminou: basta repetir com o token atual, sem renovar de novo.
    const { accessToken } = useAuthStore.getState()
    const sentWith = original.headers?.Authorization
    if (!accessToken || sentWith === `Bearer ${accessToken}`) {
      try {
        await refreshSession()
      } catch {
        // refreshSession já fez o logout (uma vez só para todas as requests).
        return Promise.reject(error)
      }
    }

    // O request interceptor injeta o access token atual na repetição.
    return api(original)
  }
)

export default api
