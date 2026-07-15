import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import i18n from '@/plugins/i18n'
import SettingsPage from '../SettingsPage.vue'
import apiClient from '@/api/client'

vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), put: vi.fn() },
}))

const sampleSettings = [
  {
    key: 'admin_password',
    value: '',
    default: '',
    env_var: 'WDSPLIT_ADMIN_PASSWORD',
    env_locked: false,
    editable: false,
    secret: true,
    type: 'password',
    options: [],
  },
  {
    key: 'session_max_age',
    value: '28800',
    default: '28800',
    env_var: 'WDSPLIT_SESSION_MAX_AGE',
    env_locked: false,
    editable: true,
    secret: false,
    type: 'int',
    options: [],
  },
  {
    key: 'admin_cookie_secure',
    value: 'auto',
    default: 'auto',
    env_var: 'WDSPLIT_ADMIN_COOKIE_SECURE',
    env_locked: false,
    editable: true,
    secret: false,
    type: 'select',
    options: ['auto', 'true', 'false'],
  },
]

function mountPage() {
  const router = createRouter({
    history: createWebHistory('/'),
    routes: [
      { path: '/settings', name: 'settings', component: SettingsPage },
      { path: '/', name: 'dashboard', component: { template: '<div />' } },
    ],
  })
  return mount(SettingsPage, {
    global: { plugins: [router, i18n, PrimeVue, ToastService] },
  })
}

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0))
}

describe('SettingsPage', () => {
  beforeEach(() => {
    vi.mocked(apiClient.get).mockReset()
    vi.mocked(apiClient.put).mockReset()
  })

  it('loads and renders every setting', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: sampleSettings })
    const wrapper = mountPage()
    await flushPromises()
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Admin password')
    expect(wrapper.text()).toContain('Session max age')
    expect(wrapper.text()).toContain('Admin cookie Secure flag')
  })

  it('shows a load error on failure', async () => {
    vi.mocked(apiClient.get).mockRejectedValueOnce(new Error('network down'))
    const wrapper = mountPage()
    await flushPromises()
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Failed to load settings')
  })

  it('Save button is disabled until something changes', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: sampleSettings })
    const wrapper = mountPage()
    await flushPromises()
    await wrapper.vm.$nextTick()

    const buttons = wrapper.findAll('button')
    const save = buttons.find((b) => b.text().includes('Save'))
    expect(save).toBeDefined()
    expect(save!.attributes('disabled')).toBeDefined()
  })

  it('save() sends only the changed, editable, non-env-locked fields', async () => {
    // Uses a "string"-typed field (plain InputText) rather than
    // session_max_age's real "int" type (PrimeVue's InputNumber does
    // internal formatting/parsing that setValue() doesn't cleanly drive
    // in jsdom — a widget-testing friction point, not a bug; the
    // SettingField -> emit wiring itself is already covered directly in
    // SettingField.spec.ts). This test's job is SettingsPage's save-
    // filtering logic, not re-exercising every widget type.
    const settingsWithAString = sampleSettings.map((s) =>
      s.key === 'session_max_age' ? { ...s, type: 'string' } : s,
    )
    vi.mocked(apiClient.get)
      .mockResolvedValueOnce({ data: settingsWithAString })
      .mockResolvedValueOnce({ data: settingsWithAString })
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: { ok: true } })

    const wrapper = mountPage()
    await flushPromises()
    await wrapper.vm.$nextTick()

    await wrapper.find('input[type="text"]').setValue('3600')
    await wrapper.vm.$nextTick()

    const buttons = wrapper.findAll('button')
    const save = buttons.find((b) => b.text().includes('Save'))
    await save!.trigger('click')
    await flushPromises()

    expect(apiClient.put).toHaveBeenCalledTimes(1)
    const [url, body] = vi.mocked(apiClient.put).mock.calls[0]
    expect(url).toBe('/admin/settings')
    expect(body).toEqual({ session_max_age: '3600' })
  })
})
