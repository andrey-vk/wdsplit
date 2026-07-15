// Declarative label/hint i18n keys per setting, keyed by the backend's
// stable setting.key. Adding a setting = adding one entry here; no
// SettingsPage template changes needed (see SettingField.vue).
export interface SettingMeta {
  labelKey: string
  hintKey: string
}

export const settingsMeta: Record<string, SettingMeta> = {
  admin_password: {
    labelKey: 'settings.fields.admin_password.label',
    hintKey: 'settings.fields.admin_password.hint',
  },
  session_secret: {
    labelKey: 'settings.fields.session_secret.label',
    hintKey: 'settings.fields.session_secret.hint',
  },
  session_max_age: {
    labelKey: 'settings.fields.session_max_age.label',
    hintKey: 'settings.fields.session_max_age.hint',
  },
  admin_cookie_secure: {
    labelKey: 'settings.fields.admin_cookie_secure.label',
    hintKey: 'settings.fields.admin_cookie_secure.hint',
  },
}
