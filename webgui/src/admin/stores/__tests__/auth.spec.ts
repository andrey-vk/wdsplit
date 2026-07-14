import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../auth'
import apiClient from '@/api/client'

vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), post: vi.fn() },
}))

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(apiClient.get).mockReset()
    vi.mocked(apiClient.post).mockReset()
  })

  it('login() succeeds and sets isAuthenticated', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({ data: { ok: true } })

    const store = useAuthStore()
    const err = await store.login('correct-password')

    expect(err).toBeNull()
    expect(store.isAuthenticated).toBe(true)
    expect(apiClient.post).toHaveBeenCalledWith(
      '/admin/login',
      { password: 'correct-password' },
      { skipAuthRedirect: true },
    )
  })

  it('login() returns the server error and leaves isAuthenticated false', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({ data: { ok: false, error: 'Invalid password' } })

    const store = useAuthStore()
    const err = await store.login('wrong-password')

    expect(err).toBe('Invalid password')
    expect(store.isAuthenticated).toBe(false)
  })

  it('login() translates a network failure', async () => {
    vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('boom'))

    const store = useAuthStore()
    const err = await store.login('anything')

    expect(err).toBe('Network error. Please try again.')
    expect(store.isAuthenticated).toBe(false)
  })

  it('checkAuth() reflects an authenticated session', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: { authenticated: true } })

    const store = useAuthStore()
    const result = await store.checkAuth()

    expect(result).toBe(true)
    expect(store.isAuthenticated).toBe(true)
    expect(store.isChecking).toBe(false)
  })

  it('checkAuth() reflects no session without throwing', async () => {
    vi.mocked(apiClient.get).mockRejectedValueOnce(new Error('401'))

    const store = useAuthStore()
    const result = await store.checkAuth()

    expect(result).toBe(false)
    expect(store.isAuthenticated).toBe(false)
  })

  it('checkAuth() deduplicates concurrent calls into one request', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: { authenticated: true } })

    const store = useAuthStore()
    const [a, b] = await Promise.all([store.checkAuth(), store.checkAuth()])

    expect(a).toBe(true)
    expect(b).toBe(true)
    expect(apiClient.get).toHaveBeenCalledTimes(1)
  })

  it('logout() clears isAuthenticated even if the request fails', async () => {
    vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('boom'))

    const store = useAuthStore()
    store.isAuthenticated = true
    await store.logout()

    expect(store.isAuthenticated).toBe(false)
  })
})
