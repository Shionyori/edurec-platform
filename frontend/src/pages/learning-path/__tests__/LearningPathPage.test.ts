import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { LearningPath, Resource } from '@/types'
import { completePathStep, getLearningPath, getPathResources } from '@/api/learning'
import LearningPathPage from '../DetailPage.vue'

const { pushMock } = vi.hoisted(() => ({ pushMock: vi.fn() }))
vi.mock('vue-router', async () => {
  const { reactive } = await import('vue')
  const route = reactive({ params: { id: '1' } })
  return {
    route,
    useRoute: () => route,
    useRouter: () => ({ push: pushMock }),
    RouterLink: { template: '<a><slot /></a>' },
  }
})
vi.mock('@/api/learning', () => ({
  getLearningPath: vi.fn(),
  getPathResources: vi.fn(),
  completePathStep: vi.fn(),
}))

const mockedPath = vi.mocked(getLearningPath)
const mockedResources = vi.mocked(getPathResources)
const mockedComplete = vi.mocked(completePathStep)

const resource: Resource = {
  id: 41, title: '线性代数精讲', description: '', cover_url: null, type: 'course',
  category: { id: 2, name: 'B站视频' }, tags: [], metadata: {}, author: '某老师',
  source_url: 'https://www.bilibili.com/video/BV1xx', avg_rating: 4.5, view_count: 100,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-02T08:00:00Z',
}

const activePath: LearningPath = {
  id: 1, target_code: 'machine-learning', target_name: '机器学习',
  steps: [
    { code: 'calculus', name: '高等数学', level: 1, mastery: 1, status: 'done', resource_count: 43 },
    { code: 'linear-algebra', name: '线性代数', level: 1, mastery: 0.67, status: 'current', resource_count: 15 },
    { code: 'data-structures', name: '数据结构与算法', level: 1, mastery: 0, status: 'locked', resource_count: 5 },
  ],
  current_index: 1,
  status: 'active',
}

const resourcesPage = {
  knowledge_point: { id: 2, code: 'linear-algebra', name: '线性代数' },
  list: [resource],
  total: 1,
  page: 1,
  page_size: 9,
}

async function mountPage() {
  const wrapper = mount(LearningPathPage, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

describe('LearningPathPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedPath.mockResolvedValue(activePath)
    mockedResources.mockResolvedValue(resourcesPage)
    mockedComplete.mockResolvedValue({
      ...activePath,
      current_index: 2,
      steps: activePath.steps.map((step, index) => ({
        ...step,
        status: index === 1 ? 'done' : index === 2 ? 'current' : step.status,
      })),
    })
  })

  it('展示步骤状态、进度与当前一节资源', async () => {
    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('机器学习 · 学习路径')
    expect(wrapper.text()).toContain('高等数学')
    expect(wrapper.text()).toContain('线性代数')
    expect(wrapper.text()).toContain('数据结构与算法')
    expect(wrapper.text()).toContain('已完成 1 / 3 节')
    expect(wrapper.text()).toContain('当前一节')
    expect(wrapper.text()).toContain('线性代数精讲')
    expect(wrapper.text()).toContain('未解锁')
    expect(mockedResources).toHaveBeenCalledWith(1, 9)
  })

  it('学完当前节后重新拉取路径并解锁下一节', async () => {
    const wrapper = await mountPage()
    const btn = wrapper.findAll('button').find((b) => b.text().includes('学完本节'))
    expect(btn).toBeTruthy()
    await btn!.trigger('click')
    await flushPromises()

    expect(mockedComplete).toHaveBeenCalledWith(1, 1)
    // 路径直接用 complete 接口返回的新状态，不再额外拉一次路径
    expect(mockedPath).toHaveBeenCalledTimes(1)
    expect(mockedResources).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('已完成 2 / 3 节')
  })

  it('路径全部完成时不再显示完成按钮', async () => {
    mockedPath.mockResolvedValue({
      ...activePath,
      status: 'completed',
      current_index: 2,
      steps: activePath.steps.map((step) => ({ ...step, status: 'done' })),
    })
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('这条学习路径已经学完')
    expect(wrapper.findAll('button').some((b) => b.text().includes('学完本节'))).toBe(false)
  })
})
