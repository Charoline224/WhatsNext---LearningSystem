import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', redirect: '/spaces' },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/auth/LoginView.vue'),
      meta: { guestOnly: true },
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/auth/RegisterView.vue'),
      meta: { guestOnly: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/AppLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: 'spaces',
          name: 'spaces',
          component: () => import('@/views/spaces/SpaceListView.vue'),
        },
        {
          path: 'spaces/new',
          name: 'space-create',
          component: () => import('@/views/spaces/SpaceCreateView.vue'),
        },
        {
          path: 'spaces/:spaceId',
          name: 'space-overview',
          component: () => import('@/views/spaces/SpaceOverviewView.vue'),
        },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/spaces' },
  ],
})

router.beforeEach(async (to) => {
  const { useAuthStore } = await import('@/stores/auth')
  const auth = useAuthStore()
  await auth.initialize()

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.guestOnly && auth.isAuthenticated) return { name: 'spaces' }
})

export default router
