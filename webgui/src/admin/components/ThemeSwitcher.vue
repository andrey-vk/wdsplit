<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useThemeStore, type ThemeMode } from '@/admin/stores/theme'
import SelectButton from 'primevue/selectbutton'

const { t } = useI18n()
const themeStore = useThemeStore()

// Icons, not emoji: emoji glyph coverage/rendering is inconsistent across
// platforms and fonts (e.g. renders as a broken tofu box in a browser
// with no color-emoji font installed) — primeicons is already used
// everywhere else in the app and renders reliably via its own webfont.
const options: { icon: string; value: ThemeMode; label: string }[] = [
  { icon: 'pi pi-sun', value: 'light', label: t('theme.light') },
  { icon: 'pi pi-moon', value: 'dark', label: t('theme.dark') },
]
</script>

<template>
  <SelectButton
    :model-value="themeStore.mode"
    :options="options"
    option-label="label"
    option-value="value"
    :allow-empty="false"
    size="small"
    @update:model-value="themeStore.setMode"
  >
    <template #option="{ option }">
      <i :class="option.icon" :aria-label="option.label" />
    </template>
  </SelectButton>
</template>
