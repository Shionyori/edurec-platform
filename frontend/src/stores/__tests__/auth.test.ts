import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '../auth'
import * as authApi from '@/api/auth'
import * as userApi from '@/api/user'
import type { User } from '@/types'

vi.mock('@/api/auth', () => ({ login: vi.fn(), register: vi.fn() }))
vi.mock('@/api/user', () => ({ getMe: vi.fn() }))

const mockedAuthApi = vi.mocked(authApi)
const mockedUserApi = vi.mocked(userApi)

const baseUser: User = {
  id: 2,
  username: 'user',
  email: 'user@edurec.dev',
  display_name: '张三',
  avatar_url: null,
  is_admin: false,
  created_at: '2026-07-05T08:00:00Z',
}

const loginResult = {
  access_token: 'at',
  refresh_token: 'rt',
  expires_in: 900,
  user: baseUser,
}

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('login 保存 token 与用户信息', async () => {
    mockedAuthApi.login.mockResolvedValue(loginResult)
    const store = useAuthStore()
    await store.login('user', 'user123')
    expect(store.isLoggedIn).toBe(true)
    expect(store.user?.username).toBe('user')
    expect(localStorage.getItem('edurec_access_token')).toBe('at')
    expect(localStorage.getItem('edurec_refresh_token')).toBe('rt')
  })

  it('logout 清除 token 与用户信息', async () => {
    mockedAuthApi.login.mockResolvedValue(loginResult)
    const store = useAuthStore()
    await store.login('user', 'user123')
    store.logout()
    expect(store.isLoggedIn).toBe(false)
    expect(store.user).toBeNull()
    expect(localStorage.getItem('edurec_access_token')).toBeNull()
  })

  it('isAdmin 反映用户管理员身份', () => {
    const store = useAuthStore()
    store.user = { ...baseUser, is_admin: true }
    expect(store.isAdmin).toBe(true)
    store.user = baseUser
    expect(store.isAdmin).toBe(false)
  })

  it('fetchMe 拉取用户信息', async () => {
    mockedUserApi.getMe.mockResolvedValue(baseUser)
    const store = useAuthStore()
    await store.fetchMe()
    expect(store.user).toEqual(baseUser)
  })
})
