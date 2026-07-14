import { createI18n } from 'vue-i18n'
import en from '@/locales/en.json'
import ru from '@/locales/ru.json'

type Locale = 'en' | 'ru'

const LANGUAGE_KEY = 'wdsplit_language'

function detectLocale(): Locale {
  const stored = window.localStorage.getItem(LANGUAGE_KEY)
  if (stored === 'en' || stored === 'ru') return stored

  if (navigator.language?.split('-')[0] === 'ru') return 'ru'

  return 'en'
}

const i18n = createI18n({
  legacy: false,
  locale: detectLocale(),
  fallbackLocale: 'en',
  messages: { en, ru },
})

export function switchLocale(locale: Locale): void {
  i18n.global.locale.value = locale
  window.localStorage.setItem(LANGUAGE_KEY, locale)
}

export function getCurrentLocale(): Locale {
  return i18n.global.locale.value as Locale
}

export default i18n
