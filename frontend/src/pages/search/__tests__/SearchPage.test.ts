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

type ObserverCallback = (entries: IntersectionObserverEntry[], observer: IntersectionObserver) => void

let observerCallback: ObserverCallback | null = null

class MockIntersectionObserver {
  constructor(cb: ObserverCallback) {
    observerCallback = cb
  }
  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords() {
    return []
  }
}

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

// B 站爬回来的新资源，id 与本地结果不同
const onlineResource: Resource = {
  ...resource,
  id: 2,
  title: '机器学习实战（B 站视频）',
  source_url: 'https://www.bilibili.com/video/BV1xx411c7mD',
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

async function searchKeyword(wrapper: ReturnType<typeof mount>, keyword: string) {
  const input = wrapper.find('input[placeholder="输入关键词…"]')
  await input.setValue(keyword)
  await input.trigger('keyup.enter')
  await flushPromises()
}

function triggerLoadMore() {
  observerCallback!([{ isIntersecting: true } as IntersectionObserverEntry], {} as IntersectionObserver)
}

describe('SearchPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    observerCallback = null
    vi.stubGlobal('IntersectionObserver', MockIntersectionObserver)
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
    await searchKeyword(wrapper, 'Python')
    expect(mockedListResources).toHaveBeenLastCalledWith(expect.objectContaining({ keyword: 'Python', page: 1 }))
  })

  it('没有结果时展示空态', async () => {
    mockedListResources.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 12 })
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('没有符合条件的资源')
  })

  it('纯关键词搜索本地翻完后，滚到底自动爬取 B 站', async () => {
    mockedListResources.mockImplementation(async (params = {}) => {
      if (params.online_page) {
        return { list: [onlineResource], total: 0, page: params.online_page, page_size: 12, has_more: true }
      }
      return { list: [resource], total: 1, page: 1, page_size: 12 }
    })
    const wrapper = await mountPage()
    await searchKeyword(wrapper, '机器学习')
    mockedListResources.mockClear()

    triggerLoadMore()
    await flushPromises()

    expect(mockedListResources).toHaveBeenLastCalledWith(
      expect.objectContaining({ online_page: 1, keyword: '机器学习' }),
    )
    // 追加不覆盖：本地 1 条 + B 站 1 条
    expect(wrapper.findAllComponents({ name: 'ResourceCard' }).length).toBe(2)
  })

  it('B 站判重命中回传本地已有资源时不重复渲染', async () => {
    mockedListResources.mockImplementation(async (params = {}) => {
      if (params.online_page) {
        // 后端 online 路径会把判重命中的行一并回传，这里回传的正是本地已展示的那条
        return { list: [resource], total: 0, page: params.online_page, page_size: 12, has_more: true }
      }
      return { list: [resource], total: 1, page: 1, page_size: 12 }
    })
    const wrapper = await mountPage()
    await searchKeyword(wrapper, '机器学习')

    triggerLoadMore()
    await flushPromises()

    expect(wrapper.findAllComponents({ name: 'ResourceCard' }).length).toBe(1)
  })

  it('首屏本地无结果时自动续拉 B 站，无需用户滚动', async () => {
    // 真实 IntersectionObserver 会在 observe() 之后投递一次初始通知，
    // 此处同步投递以模拟该时机（早于本地接口返回）
    class AutoNotifyObserver {
      constructor(private cb: ObserverCallback) {}
      observe() {
        this.cb([{ isIntersecting: true } as IntersectionObserverEntry], {} as IntersectionObserver)
      }
      unobserve() {}
      disconnect() {}
      takeRecords() {
        return []
      }
    }
    vi.stubGlobal('IntersectionObserver', AutoNotifyObserver)

    mockedListResources.mockImplementation(async (params = {}) => {
      if (params.online_page) {
        return { list: [onlineResource], total: 0, page: params.online_page, page_size: 12, has_more: false }
      }
      return { list: [], total: 0, page: 1, page_size: 12 }
    })

    const wrapper = await mountPage()
    await searchKeyword(wrapper, '机器学习')

    expect(mockedListResources).toHaveBeenCalledWith(expect.objectContaining({ online_page: 1 }))
    expect(wrapper.text()).toContain('机器学习实战（B 站视频）')
  })

  it('B 站无更多时停止拉取', async () => {
    let onlineCalls = 0
    mockedListResources.mockImplementation(async (params = {}) => {
      if (params.online_page) {
        onlineCalls += 1
        return { list: [], total: 0, page: params.online_page, page_size: 12, has_more: false }
      }
      return { list: [resource], total: 1, page: 1, page_size: 12 }
    })
    const wrapper = await mountPage()
    await searchKeyword(wrapper, '机器学习')

    triggerLoadMore()
    await flushPromises()
    expect(onlineCalls).toBe(1)
    expect(wrapper.text()).toContain('没有更多内容')

    triggerLoadMore()
    await flushPromises()
    expect(onlineCalls).toBe(1)
  })

  it('带筛选时本地翻完即停，不爬 B 站', async () => {
    mockedListResources.mockImplementation(async (params) => {
      return { list: [resource], total: 1, page: 1, page_size: 12 }
    })
    const wrapper = await mountPage()
    await searchKeyword(wrapper, '机器学习')
    // 选择类型筛选，触发带筛选的搜索
    const typeSelect = wrapper.findAllComponents({ name: 'ElSelect' })[1]
    typeSelect.vm.$emit('update:modelValue', 'video')
    typeSelect.vm.$emit('change', 'video')
    await flushPromises()
    mockedListResources.mockClear()

    triggerLoadMore()
    await flushPromises()

    // 不会再请求 online_page
    expect(mockedListResources).not.toHaveBeenCalledWith(expect.objectContaining({ online_page: expect.anything() }))
  })

  it('接口失败时展示错误态', async () => {
    mockedListResources.mockRejectedValue(new Error('加载失败'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('加载失败')
  })
})
