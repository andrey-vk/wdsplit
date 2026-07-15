<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const { t } = useI18n()

const items = [
  { to: { name: 'dashboard' }, icon: 'pi pi-home', label: () => t('nav.dashboard') },
  { to: { name: 'settings' }, icon: 'pi pi-cog', label: () => t('nav.settings') },
]
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-20 bg-black/50 lg:hidden"
    @click="emit('close')"
  />

  <aside
    class="fixed inset-y-0 left-0 z-30 w-64 shrink-0 border-r border-surface-200 bg-surface-0 transition-transform dark:border-surface-800 dark:bg-surface-900 lg:translate-x-0"
    :class="open ? 'translate-x-0' : '-translate-x-full'"
  >
    <div class="flex h-16 items-center gap-2 px-6">
      <span class="text-xl font-bold text-primary">{{ t('app.title') }}</span>
    </div>
    <nav class="flex flex-col gap-1 px-3">
      <router-link
        v-for="item in items"
        :key="item.icon"
        :to="item.to"
        class="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-surface-700 hover:bg-surface-100 dark:text-surface-300 dark:hover:bg-surface-800"
        active-class="!bg-primary/10 !text-primary"
        @click="emit('close')"
      >
        <i :class="item.icon" />
        {{ item.label() }}
      </router-link>
    </nav>
  </aside>
</template>
