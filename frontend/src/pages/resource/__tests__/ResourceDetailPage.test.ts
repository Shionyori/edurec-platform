import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { Rating, Resource } from '@/types'
import { getResource } from '@/api/resource'
import { listRatings, upsertRating } from '@/api/rating'
import { recordBehavior } from '@/api/behavior'
import * as vueRouterMock from 'vue-router'
import ResourceDetailPage from '../ResourceDetailPage.vue'

// route 用 reactive 包装并在 mock 里暴露出来，便于模拟「同一路由记录下切换 :id」——
// 这时 Vue Router 复用组件实例而不重新挂载，是本次修复的核心场景
vi.mock('vue-router', async () => {
  const { reactive } = await import('vue')
  const route = reactive({ params: { id: '1' } })
  return {
    route,
    useRoute: () => route,
    RouterLink: { template: '<a><slot /></a>' },
  }
})

const route = (vueRouterMock as unknown as { route: { params: { id: string } } }).route
vi.mock('@/api/resource', () => ({ getResource: vi.fn() }))
vi.mock('@/api/rating', () => ({ listRatings: vi.fn(), upsertRating: vi.fn() }))
vi.mock('@/api/behavior', () => ({ recordBehavior: vi.fn() }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { id: 2, username: 'user' } }) }))

const mockedGetResource = vi.mocked(getResource)
const mockedListRatings = vi.mocked(listRatings)
const mockedUpsertRating = vi.mocked(upsertRating)
const mockedRecordBehavior = vi.mocked(recordBehavior)

const resource: Resource = {
  id: 1, title: '机器学习入门', description: '面向零基础学习者',
  cover_url: null, type: 'course',
  category: { id: 1, name: '人工智能' },
  tags: ['AI'], metadata: { duration: '12 小时', chapters: 24, language: '中文' },
  author: '吴恩达', source_url: 'https://example.com',
  avg_rating: 4.5, view_count: 1024,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-15T10:00:00Z',
}

const emptyPage = { list: [] as Rating[], total: 0, page: 1, page_size: 10 }

async function mountPage() {
  const wrapper = mount(ResourceDetailPage, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

describe('ResourceDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    route.params.id = '1'
    mockedGetResource.mockResolvedValue(resource)
    mockedListRatings.mockResolvedValue(emptyPage)
    mockedRecordBehavior.mockResolvedValue(null)
  })

  it('渲染资源信息并自动上报 view 行为', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('机器学习入门')
    expect(wrapper.text()).toContain('面向零基础学习者')
    expect(mockedRecordBehavior).toHaveBeenCalledWith(1, 'view')
    expect(mockedListRatings).toHaveBeenCalledWith(1, { page: 1, page_size: 10 })
  })

  it('有封面时渲染封面，并禁用 Referer 以绕过外链防盗链', async () => {
    const coverUrl = 'https://i1.hdslb.com/bfs/archive/cover.jpg'
    mockedGetResource.mockResolvedValue({ ...resource, cover_url: coverUrl })

    const wrapper = await mountPage()

    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    expect(img.attributes('src')).toBe(coverUrl)
    expect(img.attributes('referrerpolicy')).toBe('no-referrer')
  })

  it('无封面时不渲染图片', async () => {
    const wrapper = await mountPage()
    expect(wrapper.find('img').exists()).toBe(false)
  })

  it('资源 404 时展示错误态', async () => {
    mockedGetResource.mockRejectedValue(new Error('资源不存在'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('资源不存在')
    expect(mockedRecordBehavior).not.toHaveBeenCalled()
  })

  it('提交评分后重拉资源与评分列表', async () => {
    mockedUpsertRating.mockResolvedValue({
      id: 99,
      user: { id: 2, username: 'user', display_name: '张三', avatar_url: null },
      score: 4,
      comment: '好',
      created_at: '2026-08-01T00:00:00Z',
    })
    const wrapper = await mountPage()
    mockedGetResource.mockClear()
    mockedListRatings.mockClear()
    const form = wrapper.findComponent({ name: 'RatingForm' })
    form.vm.$emit('submit', { score: 4, comment: '好' })
    await flushPromises()
    expect(mockedUpsertRating).toHaveBeenCalledWith(1, { score: 4, comment: '好' })
    expect(mockedGetResource).toHaveBeenCalledWith(1)
    expect(mockedListRatings).toHaveBeenCalledWith(1, { page: 1, page_size: 10 })
  })

  it('切换 :id 时重新拉取资源与评分，不残留上一个资源的数据', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('机器学习入门')

    mockedGetResource.mockResolvedValue({ ...resource, id: 2, title: 'Go 微服务实战' })
    route.params.id = '2'
    await flushPromises()

    expect(mockedGetResource).toHaveBeenLastCalledWith(2)
    expect(mockedListRatings).toHaveBeenLastCalledWith(2, { page: 1, page_size: 10 })
    expect(wrapper.text()).toContain('Go 微服务实战')
    expect(wrapper.text()).not.toContain('机器学习入门')
  })

  it('metadata 里的 Unix 时间戳渲染为可读日期', async () => {
    mockedGetResource.mockResolvedValue({
      ...resource,
      metadata: { pubdate: 1759988744, duration: '16:08' },
    })

    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('2025-10-09')
    expect(wrapper.text()).not.toContain('1759988744')
    // 其余字段原样渲染
    expect(wrapper.text()).toContain('16:08')
  })

  it('切页时重新拉取评分', async () => {
    const wrapper = await mountPage()
    mockedListRatings.mockClear()
    const list = wrapper.findComponent({ name: 'RatingList' })
    list.vm.$emit('page-change', 2)
    await flushPromises()
    expect(mockedListRatings).toHaveBeenCalledWith(1, { page: 2, page_size: 10 })
  })
})
