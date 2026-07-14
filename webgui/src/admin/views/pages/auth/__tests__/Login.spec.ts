import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import PrimeVue from 'primevue/config'
import i18n from '@/plugins/i18n'
import Login from '../Login.vue'
import apiClient from '@/api/client'

vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), post: vi.fn() },
}))

function mountLogin() {
  setActivePinia(createPinia())

  const router = createRouter({
    history: createWebHistory('/'),
    routes: [
      { path: '/', name: 'dashboard', component: { template: '<div />' } },
      { path: '/login', name: 'login', component: Login },
    ],
  })

  return mount(Login, {
    global: {
      plugins: [router, i18n, PrimeVue],
    },
  })
}

describe('Login', () => {
  beforeEach(() => {
    vi.mocked(apiClient.post).mockReset()
  })

  it('renders a password field and submit button', () => {
    const wrapper = mountLogin()
    expect(wrapper.find('input#password').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').exists()).toBe(true)
  })

  it('shows a translated error message on invalid password', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({ data: { ok: false, error: 'Invalid password' } })

    const wrapper = mountLogin()
    await wrapper.find('input#password').setValue('wrong')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.text()).toContain('Invalid password')
  })

  it('navigates to dashboard on successful login', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({ data: { ok: true } })

    const wrapper = mountLogin()
    const router = wrapper.vm.$router
    const pushSpy = vi.spyOn(router, 'push')

    await wrapper.find('input#password').setValue('correct')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(pushSpy).toHaveBeenCalledWith({ name: 'dashboard' })
  })
})

function flushPromises() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}
