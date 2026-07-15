<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Button from 'primevue/button'
import { useAuthStore } from '@/admin/stores/auth'
import ThemeSwitcher from '@/admin/components/ThemeSwitcher.vue'
import LanguageSwitcher from '@/admin/components/LanguageSwitcher.vue'

defineEmits<{ 'toggle-sidebar': [] }>()

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

async function handleLogout() {
  await authStore.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <header
    class="sticky top-0 z-10 flex h-16 items-center justify-between gap-4 border-b border-surface-200 bg-surface-0 px-4 dark:border-surface-800 dark:bg-surface-900"
  >
    <Button
      icon="pi pi-bars"
      severity="secondary"
      text
      class="lg:hidden"
      :aria-label="t('nav.dashboard')"
      @click="$emit('toggle-sidebar')"
    />
    <div class="flex-1" />
    <div class="flex items-center gap-3">
      <ThemeSwitcher />
      <LanguageSwitcher />
      <Button
        :label="t('nav.logout')"
        icon="pi pi-sign-out"
        severity="secondary"
        outlined
        size="small"
        @click="handleLogout"
      />
    </div>
  </header>
</template>
