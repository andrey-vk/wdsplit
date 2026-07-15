import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import i18n from '@/plugins/i18n'
import SettingField from '../SettingField.vue'
import type { Setting } from '@/types/settings'

function baseSetting(overrides: Partial<Setting> = {}): Setting {
  return {
    key: 'test_key',
    value: '',
    default: '',
    env_var: '',
    env_locked: false,
    editable: true,
    secret: false,
    type: 'string',
    options: [],
    ...overrides,
  }
}

function mountField(setting: Setting, modelValue = '') {
  return mount(SettingField, {
    props: {
      setting,
      labelKey: 'settings.fields.admin_password.label',
      hintKey: 'settings.fields.admin_password.hint',
      modelValue,
    },
    global: { plugins: [i18n, PrimeVue] },
  })
}

describe('SettingField', () => {
  it('renders an editable string setting as a text input', () => {
    const wrapper = mountField(baseSetting({ type: 'string' }), 'hello')
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    expect(wrapper.find('.p-tag').exists()).toBe(false)
  })

  it('renders an int setting as a number input', () => {
    const wrapper = mountField(baseSetting({ type: 'int' }), '3600')
    expect(wrapper.find('input[inputmode]').exists() || wrapper.find('.p-inputnumber').exists()).toBe(true)
  })

  it('renders a select setting with the given options', () => {
    const wrapper = mountField(
      baseSetting({ type: 'select', options: ['auto', 'true', 'false'] }),
      'auto',
    )
    expect(wrapper.find('.p-select').exists()).toBe(true)
  })

  it('renders a password setting as a masked input', () => {
    const wrapper = mountField(baseSetting({ type: 'password', secret: true, editable: true }))
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
  })

  it('renders env-only settings as read-only with an ENV ONLY tag, no input', () => {
    const wrapper = mountField(baseSetting({ editable: false, env_var: 'WDSPLIT_ADMIN_PASSWORD' }))
    expect(wrapper.find('.p-tag').text()).toContain('ENV ONLY')
    expect(wrapper.find('input').exists()).toBe(false)
  })

  it('renders env-locked (but DB-editable) settings as read-only with an ENV tag', () => {
    const wrapper = mountField(
      baseSetting({ editable: true, env_locked: true, env_var: 'WDSPLIT_SESSION_MAX_AGE', value: '3600' }),
    )
    expect(wrapper.find('.p-tag').text()).toBe('ENV')
    expect(wrapper.find('input').exists()).toBe(false)
    expect(wrapper.text()).toContain('3600')
  })

  it('shows "Configured" for a read-only secret, never the raw value', () => {
    const wrapper = mountField(baseSetting({ editable: false, secret: true, value: '' }))
    expect(wrapper.text()).toContain('Configured')
  })

  it('shows the current value for a read-only non-secret setting', () => {
    const wrapper = mountField(baseSetting({ editable: false, secret: false, value: 'some-value' }))
    expect(wrapper.text()).toContain('some-value')
  })

  it('falls back to the default when value is empty for a read-only non-secret setting', () => {
    const wrapper = mountField(baseSetting({ editable: false, secret: false, value: '', default: 'auto' }))
    expect(wrapper.text()).toContain('auto')
  })

  it('emits update:modelValue when editing a string field', async () => {
    const wrapper = mountField(baseSetting({ type: 'string' }), 'old')
    await wrapper.find('input[type="text"]').setValue('new')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['new'])
  })
})
