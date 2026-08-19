import { beforeAll, afterAll, afterEach, describe, it, expect } from 'vitest'
import { setupServer } from 'msw/node'
import { handlers } from '../handlers'
import { client } from '@/api/client'
import { tokenStorage } from '@/utils/token'
import { listResources, getResource } from '@/api/resource'
import { login } from '@/api/auth'
import { listCategories } from '@/api/category'
import { getMe } from '@/api/user'
import { upsertRating } from '@/api/rating'

const server = setupServer(...handlers)

beforeAll(() => {
  server.listen()
  client.defaults.baseURL = 'http://localhost/api/v1'
  // jsdom 中 axios 默认选 xhr adapter，不会被 MSW node server 拦截；强制 fetch adapter
  client.defaults.adapter = 'fetch'
})

afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('MSW handlers', () => {
  async function loginAsAdmin() {
    const result = await login({ username: 'admin', password: 'admin123' })
    tokenStorage.setAccess(result.access_token)
  }

  it('登录返回统一响应结构', async () => {
    const result = await login({ username: 'admin', password: 'admin123' })
    expect(result.access_token).toBeTruthy()
    expect(result.user.username).toBe('admin')
  })

  it('登录失败返回业务错误', async () => {
    await expect(login({ username: 'admin', password: 'wrong' })).rejects.toThrow('用户名或密码错误')
  })

  it('资源列表返回种子数据', async () => {
    await loginAsAdmin()
    const data = await listResources({ page: 1, page_size: 20 })
    expect(data.list.length).toBeGreaterThan(0)
    expect(data.list[0]).toHaveProperty('title')
  })

  it('未认证访问受保护接口返回 401', async () => {
    await expect(getMe()).rejects.toThrow('未认证')
  })

  it('分类列表返回数据', async () => {
    await loginAsAdmin()
    const categories = await listCategories()
    expect(categories.length).toBeGreaterThan(0)
  })

  it('评分 upsert 后重算 avg_rating', async () => {
    await loginAsAdmin()
    const before = await getResource(1)
    await upsertRating(1, { score: 1 })
    const after = await getResource(1)
    expect(after.avg_rating).not.toBe(before.avg_rating)
    expect(after.avg_rating).toBeGreaterThan(0)
    expect(after.avg_rating).toBeLessThanOrEqual(5)
  })
})
