import { defineStore } from 'pinia'

const STORAGE_KEY = 'bets-theme'
type ThemeMode = 'system' | 'light' | 'dark'

export const useThemeStore = defineStore('theme', {
  state: () => ({
    currentTheme: (localStorage.getItem(STORAGE_KEY) || 'system') as ThemeMode
  }),

  getters: {
    isDark: (state): boolean => {
      if (state.currentTheme === 'dark') return true
      if (state.currentTheme === 'light') return false
      return window.matchMedia('(prefers-color-scheme: dark)').matches
    }
  },

  actions: {
    setTheme(theme: ThemeMode) {
      this.currentTheme = theme
      localStorage.setItem(STORAGE_KEY, theme)
      this.applyTheme()
    },

    cycleTheme() {
      const themes: ThemeMode[] = ['system', 'light', 'dark']
      const nextIndex = (themes.indexOf(this.currentTheme) + 1) % themes.length
      this.setTheme(themes[nextIndex]!)
    },

    applyTheme() {
      if (this.isDark) {
        document.documentElement.classList.add('dark')
      } else {
        document.documentElement.classList.remove('dark')
      }
    },

    initTheme() {
      this.applyTheme()
      window.matchMedia('(prefers-color-scheme: dark)')
        .addEventListener('change', () => {
          if (this.currentTheme === 'system') this.applyTheme()
        })
    }
  }
})
