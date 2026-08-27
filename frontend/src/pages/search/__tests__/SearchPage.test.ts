import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { Category, Resource } from '@/types'
import { listCategories } from '@/api/category'
import { listResources } from '@/api/resource'
import SearchPage from '../SearchPage.vue'

vi.mock('@/api/category', () => ({ listCategories: vi.fn() }))
vi.mock('@/api/resource', () => ({ listResources: vi.fn() }))

const mockedListCategories = vi.mocked(listCategories)
const mockedListResources = vi.mocked(listResources)

const categories: Category[] = [{ id: 1, name: '人工智能', description: '' }]

const resource: Resource = {
  id: 1, title: '机器学习入门', description: '面向零基础学习者',
  cover_url: null, type: 'course',
  category: { id: 1, name: '人工智能' },
  tags: ['AI'], metadata: {},
  author: '吴恩达', source_url: null,
  avg_rating: 4.5, view_count: 1024,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-15T10:00:00Z',
}

function pageResult(total = 1) {
  return { list: [resource], total, page: 1, page_size: 12 }
}

async function mountPage() {
  const wrapper = mount(SearchPage, {
    global: { plugins: [ElementPlus], stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  return wrapper
}

describe('SearchPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedListCategories.mockResolvedValue(categories)
    mockedListResources.mockResolvedValue(pageResult())
  })

  it('加载分类并拉取默认资源列表', async () => {
    const wrapper = await mountPage()
    expect(mockedListCategories).toHaveBeenCalled()
    expect(mockedListResources).toHaveBeenCalledWith(expect.objectContaining({ page: 1, sort: 'latest' }))
    expect(wrapper.text()).toContain('机器学习入门')
  })

  it('输入关键词搜索会携带筛选条件并重置页码', async () => {
    const wrapper = await mountPage()
    mockedListResources.mockClear()
    const input = wrapper.find('input[placeholder="输入关键词…"]')
    await input.setValue('Python')
    await input.trigger('keyup.enter')
    await flushPromises()
    expect(mockedListResources).toHaveBeenLastCalledWith(expect.objectContaining({ keyword: 'Python', page: 1 }))
  })

  it('没有结果时展示空态', async () => {
    mockedListResources.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 12 })
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('没有符合条件的资源')
  })

  it('切页重新拉取对应页码', async () => {
    mockedListResources.mockResolvedValue(pageResult(13))
    const wrapper = await mountPage()
    mockedListResources.mockClear()
    const pagination = wrapper.findComponent({ name: 'ElPagination' })
    pagination.vm.$emit('current-change', 2)
    await flushPromises()
    expect(mockedListResources).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }))
  })

  it('接口失败时展示错误态', async () => {
    mockedListResources.mockRejectedValue(new Error('加载失败'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('加载失败')
  })
})
