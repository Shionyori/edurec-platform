import { afterAll, afterEach, beforeAll, describe, it, expect } from 'vitest'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'
import { unwrap, client } from '../client'
import { login } from '@/api/auth'
import { tokenStorage } from '@/utils/token'

let refreshCalled = false

const server = setupServer(
  http.post('*/api/v1/auth/login', () => {
    return HttpResponse.json({ code: 10002, message: '用户名或密码错误', data: null }, { status: 401 })
  }),
  http.post('*/api/v1/auth/refresh', () => {
    refreshCalled = true
    return HttpResponse.json({ code: 10002, message: '刷新失败', data: null }, { status: 401 })
  }),
)

beforeAll(() => {
  server.listen()
  client.defaults.baseURL = 'http://localhost/api/v1'
  client.defaults.adapter = 'fetch'
})

afterEach(() => {
  server.resetHandlers()
  refreshCalled = false
})

afterAll(() => server.close())

describe('client 统一响应解包', () => {
  it('code === 0 时返回 data', () => {
    expect(unwrap({ code: 0, message: 'ok', data: { id: 1 } })).toEqual({ id: 1 })
  })

  it('code !== 0 时抛出业务错误', () => {
    expect(() => unwrap({ code: 10001, message: '参数错误', data: null })).toThrow('参数错误')
  })
})

describe('client 401 拦截器', () => {
  it('登录接口 401 直接抛业务错误，不触发 token 续期', async () => {
    // 存在 refresh token 时，bug 会尝试调用 /auth/refresh 续期；修复后应直接抛错
    tokenStorage.setRefresh('mock-rf-1')
    refreshCalled = false
    await expect(login({ username: 'admin', password: 'wrong' })).rejects.toThrow('用户名或密码错误')
    expect(refreshCalled).toBe(false)
  })
})
