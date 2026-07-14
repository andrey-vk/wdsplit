import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/admin/stores/auth'

const router = createRouter({
  history: createWebHistory('/'),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: () => import('@/admin/views/Dashboard.vue'),
    },
    {
      path: '/login',
      name: 'login',
      meta: { public: true },
      component: () => import('@/admin/views/pages/auth/Login.vue'),
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true

  const authStore = useAuthStore()
  await authStore.checkAuth()

  return authStore.isAuthenticated || { name: 'login' }
})

export default router
