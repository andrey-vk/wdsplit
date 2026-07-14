<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/admin/stores/auth'
import Button from 'primevue/button'

const { t } = useI18n()
const authStore = useAuthStore()
</script>

<template>
  <div class="p-8">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold m-0">{{ t('nav.dashboard') }}</h1>
      <Button
        label="Logout"
        icon="pi pi-sign-out"
        severity="secondary"
        outlined
        @click="authStore.logout().then(() => $router.push({ name: 'login' }))"
      />
    </div>
  </div>
</template>
