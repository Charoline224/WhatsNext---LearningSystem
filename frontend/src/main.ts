import './assets/main.scss'
import 'element-plus/dist/index.css'

import ElementPlus from 'element-plus'
import { createPinia } from 'pinia'
import { createApp } from 'vue'

import App from './App.vue'
import { setUnauthorizedHandler } from './api/client'
import router from './router'
import { useAuthStore } from './stores/auth'

async function bootstrap() {
  if (import.meta.env.VITE_USE_MOCKS === 'true') {
    const { worker } = await import('./mocks/browser')
    await worker.start({ onUnhandledRequest: 'bypass' })
  }

  const app = createApp(App)
  const pinia = createPinia()
  app.use(pinia)
  app.use(router)
  app.use(ElementPlus)

  const auth = useAuthStore(pinia)
  setUnauthorizedHandler(() => {
    auth.clearSession()
    const currentRoute = router.currentRoute.value
    if (currentRoute.name !== 'login') {
      void router.push({ name: 'login', query: { redirect: currentRoute.fullPath } })
    }
  })
  app.mount('#app')
}

void bootstrap()
