import { describe, expect, it, vi } from 'vitest'
import { resolveGuard } from '../guards'
import type { AuthLike } from '../guards'

function makeTo(overrides: Record<string, unknown> = {}) {
  return {
    fullPath: '/resources/1',
    name: 'resource-detail',
    meta: { requiresAuth: true },
    ...overrides,
  } as unknown as Parameters<typeof resolveGuard>[0]
}

function makeAuth(overrides: Partial<AuthLike> = {}): AuthLike {
  return {
    isLoggedIn: false,
    isAdmin: false,
    user: null,
    fetchMe: vi.fn(),
    logout: vi.fn(),
    ...overrides,
  }
}

describe('路由守卫', () => {
  it('未登录访问受保护页面重定向到登录页', async () => {
    const result = await resolveGuard(makeTo(), makeAuth())
    expect(result).toEqual({ name: 'login', query: { redirect: '/resources/1' } })
  })

  it('已登录但未加载用户时先拉取用户信息', async () => {
    const auth = makeAuth({ isLoggedIn: true })
    const result = await resolveGuard(makeTo(), auth)
    expect(auth.fetchMe).toHaveBeenCalled()
    expect(result).toBe(true)
  })

  it('拉取用户信息失败则登出并重定向登录', async () => {
    const auth = makeAuth({
      isLoggedIn: true,
      fetchMe: vi.fn().mockRejectedValue(new Error('fail')),
    })
    const result = await resolveGuard(makeTo(), auth)
    expect(auth.logout).toHaveBeenCalled()
    expect(result).toEqual({ name: 'login' })
  })

  it('非管理员访问管理页面重定向首页', async () => {
    const result = await resolveGuard(
      makeTo({ meta: { requiresAuth: true, requiresAdmin: true } }),
      makeAuth({ isLoggedIn: true, isAdmin: false, user: {} }),
    )
    expect(result).toEqual({ name: 'home' })
  })

  it('已登录访问登录页重定向首页', async () => {
    const result = await resolveGuard(
      makeTo({ meta: { guestOnly: true } }),
      makeAuth({ isLoggedIn: true, user: {} }),
    )
    expect(result).toEqual({ name: 'home' })
  })

  it('无约束路由放行', async () => {
    const result = await resolveGuard(makeTo({ meta: {} }), makeAuth())
    expect(result).toBe(true)
  })
})
