<script setup lang="ts">
import { switchLocale, getCurrentLocale } from '@/plugins/i18n'
import SelectButton from 'primevue/selectbutton'

const options = [
  { label: 'EN', value: 'en' as const },
  { label: 'RU', value: 'ru' as const },
]
</script>

<template>
  <SelectButton
    :model-value="getCurrentLocale()"
    :options="options"
    option-label="label"
    option-value="value"
    :allow-empty="false"
    size="small"
    @update:model-value="switchLocale"
  />
</template>
