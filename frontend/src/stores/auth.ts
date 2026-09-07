import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { authApi } from '@/api/auth'
import { setAccessToken } from '@/api/client'
import type { LoginInput, RegisterInput, User } from '@/types/auth'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)
  const isAuthenticated = computed(() => Boolean(user.value))

  async function initialize() {
    if (initialized.value) return
    try {
      const result = await authApi.refresh()
      setAccessToken(result.access_token)
      user.value = result.user
    } catch {
      setAccessToken(null)
      user.value = null
    } finally {
      initialized.value = true
    }
  }

  async function login(input: LoginInput) {
    const result = await authApi.login(input)
    setAccessToken(result.access_token)
    user.value = result.user
    initialized.value = true
  }

  async function register(input: RegisterInput) {
    const result = await authApi.register(input)
    setAccessToken(result.access_token)
    user.value = result.user
    initialized.value = true
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      setAccessToken(null)
      user.value = null
    }
  }

  function clearSession() {
    setAccessToken(null)
    user.value = null
    initialized.value = true
  }

  return { user, initialized, isAuthenticated, initialize, login, register, logout, clearSession }
})
