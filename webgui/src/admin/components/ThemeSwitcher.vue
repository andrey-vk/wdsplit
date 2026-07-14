<script setup lang="ts">
import { useThemeStore, type ThemeMode } from '@/admin/stores/theme'
import SelectButton from 'primevue/selectbutton'

const themeStore = useThemeStore()

const options: { label: string; value: ThemeMode }[] = [
  { label: '☀️', value: 'light' },
  { label: '🌙', value: 'dark' },
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
  />
</template>
