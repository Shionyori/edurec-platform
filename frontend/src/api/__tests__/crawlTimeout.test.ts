import { beforeEach, describe, expect, it, vi } from 'vitest'

// 只替换请求方法，保留真实的 CRAWL_TIMEOUT，断言才有意义
vi.mock('../client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../client')>()
  return { ...actual, get: vi.fn(), post: vi.fn(), put: vi.fn(), del: vi.fn() }
})

import { get } from '../client'
import { listResources } from '../resource'
import { listComments } from '../comment'

const mockedGet = vi.mocked(get)

// 后端单次调用 python 的硬上限（bilibili_online.go 的 pythonTimeout）
const BACKEND_PYTHON_TIMEOUT = 30000

function timeoutOfCall(index = 0) {
  const config = mockedGet.mock.calls[index][1] as { timeout?: number } | undefined
  return config?.timeout
}

describe('触发实时爬取的接口超时', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedGet.mockResolvedValue({} as never)
  })

  it('在线翻页请求的超时晚于后端，避免后端已落库而浏览器先超时', async () => {
    await listResources({ online_page: 1, keyword: '机器学习' })

    expect(mockedGet).toHaveBeenCalledWith(
      '/resources',
      expect.objectContaining({ params: expect.objectContaining({ online_page: 1 }) }),
    )
    expect(timeoutOfCall()).toBeGreaterThan(BACKEND_PYTHON_TIMEOUT)
  })

  it('普通列表请求不传超时，沿用默认值', async () => {
    await listResources({ page: 1 })

    expect(mockedGet).toHaveBeenCalledWith('/resources', { params: { page: 1 } })
    expect(timeoutOfCall()).toBeUndefined()
  })

  it('评论列表无缓存时会实时爬取，超时同样晚于后端', async () => {
    await listComments(3)

    expect(mockedGet).toHaveBeenCalledWith('/resources/3/comments', expect.anything())
    expect(timeoutOfCall()).toBeGreaterThan(BACKEND_PYTHON_TIMEOUT)
  })
})
