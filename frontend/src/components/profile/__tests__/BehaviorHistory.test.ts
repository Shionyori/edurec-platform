import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { Behavior } from '@/types'
import { listMyBehaviors } from '@/api/behavior'
import BehaviorHistory from '../BehaviorHistory.vue'

vi.mock('@/api/behavior', () => ({ listMyBehaviors: vi.fn() }))

const mockedListMyBehaviors = vi.mocked(listMyBehaviors)

const behavior: Behavior = {
  id: 1,
  resource: { id: 5, title: 'Python 数据分析', cover_url: null, type: 'course' },
  action: 'view',
  created_at: '2026-07-27T15:30:00Z',
}

function pageResult(list: Behavior[] = [behavior], total = 1) {
  return { list, total, page: 1, page_size: 10 }
}

async function mountHistory() {
  const wrapper = mount(BehaviorHistory, {
    global: { plugins: [ElementPlus], stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  return wrapper
}

describe('BehaviorHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedListMyBehaviors.mockResolvedValue(pageResult())
  })

  it('加载并渲染行为列表', async () => {
    const wrapper = await mountHistory()
    expect(mockedListMyBehaviors).toHaveBeenCalledWith(expect.objectContaining({ page: 1, action: undefined }))
    expect(wrapper.text()).toContain('Python 数据分析')
    expect(wrapper.text()).toContain('浏览') // 行为动作已汉化（view → 浏览）
  })

  it('首次加载显示骨架；切换筛选时保留旧列表，不闪空态', async () => {
    let resolveFirst!: (v: ReturnType<typeof pageResult>) => void
    mockedListMyBehaviors.mockReturnValueOnce(new Promise((r) => { resolveFirst = r }))
    const wrapper = mount(BehaviorHistory, {
      global: { plugins: [ElementPlus], stubs: { RouterLink: RouterLinkStub } },
    })
    await flushPromises()
    expect(wrapper.find('.el-skeleton').exists()).toBe(true)

    resolveFirst(pageResult())
    await flushPromises()
    expect(wrapper.text()).toContain('Python 数据分析')

    // 切换筛选：新请求未返回时旧列表仍在，且不再回到骨架屏
    mockedListMyBehaviors.mockReturnValueOnce(new Promise(() => {}))
    const group = wrapper.findComponent({ name: 'ElRadioGroup' })
    group.vm.$emit('update:modelValue', 'favorite')
    group.vm.$emit('change', 'favorite')
    await flushPromises()

    expect(wrapper.find('.el-skeleton').exists()).toBe(false)
    expect(wrapper.text()).toContain('Python 数据分析')
  })

  it('按行为类型筛选会重置页码', async () => {
    const wrapper = await mountHistory()
    mockedListMyBehaviors.mockClear()
    const group = wrapper.findComponent({ name: 'ElRadioGroup' })
    group.vm.$emit('update:modelValue', 'favorite')
    group.vm.$emit('change', 'favorite')
    await flushPromises()
    expect(mockedListMyBehaviors).toHaveBeenLastCalledWith(
      expect.objectContaining({ action: 'favorite', page: 1 }),
    )
  })

  it('没有记录时展示空态', async () => {
    mockedListMyBehaviors.mockResolvedValue(pageResult([], 0))
    const wrapper = await mountHistory()
    expect(wrapper.text()).toContain('暂无行为记录')
  })

  it('内容区预留高度、分页器容器始终存在（避免切换时页面高度变化导致滚动位移）', async () => {
    // total 小于一页：不渲染分页器，但占位容器仍在
    mockedListMyBehaviors.mockResolvedValue(pageResult([behavior], 1))
    const wrapper = await mountHistory()

    expect(wrapper.find('[data-testid="history-content"]').classes()).toContain('min-h-[18rem]')
    expect(wrapper.findComponent({ name: 'ElPagination' }).exists()).toBe(false)
    expect(wrapper.find('[data-testid="history-pagination"]').exists()).toBe(true)
  })

  it('切页重新拉取对应页码', async () => {
    mockedListMyBehaviors.mockResolvedValue(pageResult([behavior], 13))
    const wrapper = await mountHistory()
    mockedListMyBehaviors.mockClear()
    const pagination = wrapper.findComponent({ name: 'ElPagination' })
    pagination.vm.$emit('current-change', 2)
    await flushPromises()
    expect(mockedListMyBehaviors).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }))
  })

  it('接口失败时展示错误态', async () => {
    mockedListMyBehaviors.mockRejectedValue(new Error('加载失败'))
    const wrapper = await mountHistory()
    expect(wrapper.text()).toContain('加载失败')
  })
})
