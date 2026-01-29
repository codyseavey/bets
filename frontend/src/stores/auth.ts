import { defineStore } from 'pinia'
import api from '../services/api'

interface User {
  id: string
  email: string
  name: string
  avatar_url: string
}

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    loading: true,
  }),

  getters: {
    isAuthenticated: (state) => !!state.user,
  },

  actions: {
    async fetchUser() {
      try {
        const { data } = await api.get('/auth/me')
        this.user = data
      } catch {
        this.user = null
      } finally {
        this.loading = false
      }
    },

    async logout() {
      try {
        await api.post('/auth/logout')
      } finally {
        this.user = null
      }
    },

    loginWithGoogle() {
      window.location.href = '/api/auth/google'
    },
  },
})
