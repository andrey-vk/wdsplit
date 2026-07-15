<script setup lang="ts">
import { ref } from 'vue'
import Toast from 'primevue/toast'
import AppSidebar from './AppSidebar.vue'
import AppTopbar from './AppTopbar.vue'

const sidebarOpen = ref(false)
</script>

<template>
  <div class="min-h-screen bg-surface-50 dark:bg-surface-950">
    <Toast />
    <AppSidebar :open="sidebarOpen" @close="sidebarOpen = false" />

    <div class="lg:pl-64">
      <AppTopbar @toggle-sidebar="sidebarOpen = !sidebarOpen" />
      <main>
        <router-view />
      </main>
    </div>
  </div>
</template>
