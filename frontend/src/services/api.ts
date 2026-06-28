import axios from 'axios'
import { useAuthStore } from '@/store/authStore'

// API client com interceptors para JWT automático
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor — injeta Bearer token automaticamente
api.interceptors.request.use((config) => {
  const { accessToken } = useAuthStore.getState()
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`
  }
  return config
})

// Response interceptor — auto refresh em 401
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config

    if (error.response?.status === 401 && !original._retry) {
      original._retry = true
      const { refreshToken, setAuth, logout } = useAuthStore.getState()

      if (refreshToken) {
        try {
          const { data } = await axios.post('/api/v1/auth/refresh', {
            refresh_token: refreshToken,
          })
          setAuth(
            { accessToken: data.access_token, refreshToken: data.refresh_token },
            data.user
          )
          original.headers.Authorization = `Bearer ${data.access_token}`
          return api(original)
        } catch {
          logout()
        }
      } else {
        logout()
      }
    }

    return Promise.reject(error)
  }
)

export default api
