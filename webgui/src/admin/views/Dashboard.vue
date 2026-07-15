<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>

<template>
  <div class="p-8">
    <h1 class="text-2xl font-bold m-0">{{ t('nav.dashboard') }}</h1>
  </div>
</template>
