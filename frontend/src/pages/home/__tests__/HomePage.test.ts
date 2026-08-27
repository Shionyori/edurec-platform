import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { RecommendationResult, Resource } from '@/types'
import { getRecommendations } from '@/api/recommendation'
import HomePage from '../index.vue'

vi.mock('vue-router', () => ({
  RouterLink: { template: '<a><slot /></a>' },
}))
vi.mock('@/api/recommendation', () => ({ getRecommendations: vi.fn() }))

const mockedGetRecommendations = vi.mocked(getRecommendations)

const resource: Resource = {
  id: 1, title: '机器学习入门', description: '面向零基础学习者',
  cover_url: null, type: 'course',
  category: { id: 1, name: '人工智能' },
  tags: ['AI'], metadata: {},
  author: '吴恩达', source_url: null,
  avg_rating: 4.5, view_count: 1024,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-15T10:00:00Z',
}

const recResult: RecommendationResult = { list: [resource], updated_at: '2026-08-01T00:00:00Z' }

async function mountPage() {
  const wrapper = mount(HomePage, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

describe('HomePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedGetRecommendations.mockResolvedValue(recResult)
  })

  it('调用推荐接口并渲染推荐列表', async () => {
    const wrapper = await mountPage()
    expect(mockedGetRecommendations).toHaveBeenCalledWith(12)
    expect(wrapper.text()).toContain('机器学习入门')
  })

  it('推荐接口失败时展示错误态', async () => {
    mockedGetRecommendations.mockRejectedValue(new Error('加载失败'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('加载失败')
  })
})
