import { defineStore } from 'pinia'
import { ref } from 'vue'
import apiClient from '@/api/client'

let checkPromise: Promise<boolean> | null = null

export const useAuthStore = defineStore('auth', () => {
  const isAuthenticated = ref(false)
  const isChecking = ref(true)

  async function login(password: string): Promise<string | null> {
    try {
      const response = await apiClient.post('/admin/login', { password }, { skipAuthRedirect: true })
      if (response.data.ok) {
        isAuthenticated.value = true
        return null
      }
      return response.data.error || 'Login failed'
    } catch (err: unknown) {
      const e = err as { response?: { status?: number; data?: { error?: string } } }
      if (e.response?.status === 401) {
        return e.response.data?.error || 'Invalid password'
      }
      if (e.response?.status === 429) {
        return 'Rate limit exceeded. Please try again later.'
      }
      return 'Network error. Please try again.'
    }
  }

  async function checkAuth(): Promise<boolean> {
    if (checkPromise) return checkPromise

    isChecking.value = true
    checkPromise = (async () => {
      try {
        const response = await apiClient.get('/admin/me', { skipAuthRedirect: true })
        isAuthenticated.value = response.data.authenticated === true
        return isAuthenticated.value
      } catch {
        isAuthenticated.value = false
        return false
      } finally {
        isChecking.value = false
        checkPromise = null
      }
    })()
    return checkPromise
  }

  async function logout(): Promise<void> {
    try {
      await apiClient.post('/admin/logout', undefined, { skipAuthRedirect: true })
    } catch {
      // ignore logout errors — the client-side state clears regardless
    }
    isAuthenticated.value = false
    isChecking.value = false
  }

  return { isAuthenticated, isChecking, login, checkAuth, logout }
})
