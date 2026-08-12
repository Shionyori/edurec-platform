import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import * as authApi from '@/api/auth'
import * as userApi from '@/api/user'
import { tokenStorage } from '@/utils/token'
import type { RegisterPayload, User } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref(tokenStorage.getAccess())
  const refreshToken = ref(tokenStorage.getRefresh())
  const user = ref<User | null>(null)
  const loading = ref(false)

  const isLoggedIn = computed(() => accessToken.value !== '')
  const isAdmin = computed(() => user.value?.is_admin === true)

  async function login(username: string, password: string): Promise<User> {
    loading.value = true
    try {
      const result = await authApi.login({ username, password })
      accessToken.value = result.access_token
      refreshToken.value = result.refresh_token
      tokenStorage.setAccess(result.access_token)
      tokenStorage.setRefresh(result.refresh_token)
      user.value = result.user
      return result.user
    } finally {
      loading.value = false
    }
  }

  async function register(payload: RegisterPayload): Promise<User> {
    const result = await authApi.register(payload)
    return result.user
  }

  async function fetchMe(): Promise<User> {
    user.value = await userApi.getMe()
    return user.value
  }

  function logout(): void {
    accessToken.value = ''
    refreshToken.value = ''
    user.value = null
    tokenStorage.clear()
  }

  return {
    accessToken,
    refreshToken,
    user,
    loading,
    isLoggedIn,
    isAdmin,
    login,
    register,
    fetchMe,
    logout,
  }
})
