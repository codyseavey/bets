import axios from 'axios'
import { router } from '../router'

const api = axios.create({
  baseURL: '/api',
  withCredentials: true,
})

// Redirect to login on 401, but skip for the initial /auth/me check
// (that's handled by the auth store and router guards instead).
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401 && error.config?.url !== '/auth/me') {
      router.push('/login')
    }
    return Promise.reject(error)
  }
)

export default api
