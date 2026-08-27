import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { Resource } from '@/types'
import { adminListResources } from '@/api/admin'
import { deleteResource } from '@/api/resource'
import AdminResources from '../AdminResources.vue'

vi.mock('@/api/admin', () => ({ adminListResources: vi.fn() }))
vi.mock('@/api/resource', () => ({ deleteResource: vi.fn() }))

const mockedAdminListResources = vi.mocked(adminListResources)
const mockedDeleteResource = vi.mocked(deleteResource)

const resource: Resource = {
  id: 1, title: '机器学习入门', description: '面向零基础学习者',
  cover_url: null, type: 'course',
  category: { id: 1, name: '人工智能' },
  tags: ['AI'], metadata: {},
  author: '吴恩达', source_url: null,
  avg_rating: 4.5, view_count: 1024,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-01T08:00:00Z',
}

function pageResult(list: Resource[] = [resource], total = 1) {
  return { list, total, page: 1, page_size: 20 }
}

async function mountPage() {
  const wrapper = mount(AdminResources, {
    global: { plugins: [ElementPlus], stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  return wrapper
}

describe('AdminResources', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedAdminListResources.mockResolvedValue(pageResult())
  })

  it('加载并渲染资源表格', async () => {
    const wrapper = await mountPage()
    expect(mockedAdminListResources).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, keyword: undefined, type: undefined }),
    )
    expect(wrapper.text()).toContain('机器学习入门')
    expect(wrapper.text()).toContain('人工智能')
    expect(wrapper.text()).toContain('新增资源')
  })

  it('输入关键词搜索并重置页码', async () => {
    const wrapper = await mountPage()
    mockedAdminListResources.mockClear()
    const input = wrapper.find('input[placeholder="按标题搜索"]')
    await input.setValue('Python')
    await input.trigger('keyup.enter')
    await flushPromises()
    expect(mockedAdminListResources).toHaveBeenLastCalledWith(
      expect.objectContaining({ keyword: 'Python', page: 1 }),
    )
  })

  it('确认删除后调用接口并刷新列表', async () => {
    const wrapper = await mountPage()
    mockedAdminListResources.mockClear()
    const popconfirm = wrapper.findComponent({ name: 'ElPopconfirm' })
    popconfirm.vm.$emit('confirm')
    await flushPromises()
    expect(mockedDeleteResource).toHaveBeenCalledWith(1)
    expect(mockedAdminListResources).toHaveBeenCalled()
  })

  it('切页重新拉取对应页码', async () => {
    mockedAdminListResources.mockResolvedValue(pageResult([resource], 25))
    const wrapper = await mountPage()
    mockedAdminListResources.mockClear()
    const pagination = wrapper.findComponent({ name: 'ElPagination' })
    pagination.vm.$emit('current-change', 2)
    await flushPromises()
    expect(mockedAdminListResources).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }))
  })

  it('接口失败时展示错误态', async () => {
    mockedAdminListResources.mockRejectedValue(new Error('加载失败'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('加载失败')
  })
})
