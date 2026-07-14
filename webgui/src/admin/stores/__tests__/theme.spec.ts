import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useThemeStore } from '../theme'

describe('useThemeStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    window.localStorage.clear()
    document.documentElement.classList.remove('app-dark')
  })

  it('setMode("dark") toggles the app-dark class and persists the choice', () => {
    const store = useThemeStore()
    store.setMode('dark')

    expect(store.mode).toBe('dark')
    expect(document.documentElement.classList.contains('app-dark')).toBe(true)
    expect(window.localStorage.getItem('wdsplit_theme')).toBe('dark')
  })

  it('setMode("light") removes the app-dark class', () => {
    const store = useThemeStore()
    store.setMode('dark')
    store.setMode('light')

    expect(document.documentElement.classList.contains('app-dark')).toBe(false)
    expect(window.localStorage.getItem('wdsplit_theme')).toBe('light')
  })
})
