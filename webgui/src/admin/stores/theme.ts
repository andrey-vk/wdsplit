import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeMode = 'light' | 'dark'

const THEME_KEY = 'wdsplit_theme'

function detectInitialMode(): ThemeMode {
  const stored = window.localStorage.getItem(THEME_KEY)
  if (stored === 'light' || stored === 'dark') return stored
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export const useThemeStore = defineStore('theme', () => {
  const mode = ref<ThemeMode>(detectInitialMode())

  function apply(m: ThemeMode) {
    document.documentElement.classList.toggle('app-dark', m === 'dark')
  }

  function setMode(m: ThemeMode) {
    mode.value = m
    window.localStorage.setItem(THEME_KEY, m)
  }

  watch(mode, apply, { immediate: true, flush: 'sync' })

  return { mode, setMode }
})
