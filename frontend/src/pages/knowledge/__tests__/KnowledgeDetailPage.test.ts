import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { KnowledgeDetail, Resource } from '@/types'
import { getKnowledgePoint, startConceptGeneration, getConceptGenerationStatus } from '@/api/knowledge'
import { createLearningPath } from '@/api/learning'
import KnowledgeDetailPage from '../DetailPage.vue'

const { pushMock } = vi.hoisted(() => ({ pushMock: vi.fn() }))
vi.mock('vue-router', async () => {
  const { reactive } = await import('vue')
  const route = reactive({ params: { code: 'mathematical-analysis' } })
  return {
    route,
    useRoute: () => route,
    useRouter: () => ({ push: pushMock }),
    RouterLink: { template: '<a><slot /></a>' },
  }
})
vi.mock('@/api/knowledge', () => ({
  getKnowledgePoint: vi.fn(),
  startConceptGeneration: vi.fn(),
  getConceptGenerationStatus: vi.fn(),
}))
vi.mock('@/api/learning', () => ({ createLearningPath: vi.fn() }))

const mockedDetail = vi.mocked(getKnowledgePoint)
const mockedStartGeneration = vi.mocked(startConceptGeneration)
const mockedGenerationStatus = vi.mocked(getConceptGenerationStatus)
const mockedCreatePath = vi.mocked(createLearningPath)

const resource: Resource = {
  id: 63, title: '高等数学全程教学', description: '宋浩老师', cover_url: null,
  type: 'course', category: { id: 2, name: 'B站视频' }, tags: ['高数'], metadata: {},
  author: '宋浩', source_url: 'https://www.bilibili.com/video/BV1xx',
  avg_rating: 4.8, view_count: 1000,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-15T10:00:00Z',
}

const detail: KnowledgeDetail = {
  point: {
    id: 2, code: 'mathematical-analysis', name: '数学分析', description: '极限与积分的严格理论',
    level: 2, discipline_code: 'math', resource_count: 26, concept_count: 0,
  },
  nodes: [
    { id: 1, code: 'calculus', name: '高等数学', description: '', level: 1, discipline_code: 'math', resource_count: 43, concept_count: 91 },
    { id: 2, code: 'mathematical-analysis', name: '数学分析', description: '', level: 2, discipline_code: 'math', resource_count: 26, concept_count: 0 },
    { id: 3, code: 'linear-algebra', name: '线性代数', description: '', level: 1, discipline_code: 'math', resource_count: 15, concept_count: 19 },
  ],
  edges: [{ from: 1, to: 2 }],
  resources: [resource],
  resource_total: 1,
  concepts: [],
  concept_edges: [],
}

async function mountPage() {
  const wrapper = mount(KnowledgeDetailPage, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

describe('KnowledgeDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedDetail.mockResolvedValue(detail)
  })

  it('展示学科信息、先修图与课程资源', async () => {
    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('数学分析')
    expect(wrapper.text()).toContain('第 2 层')
    expect(wrapper.text()).toContain('课程资源 26')
    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.text()).toContain('高等数学')
    expect(wrapper.text()).toContain('高等数学全程教学')
    expect(mockedDetail).toHaveBeenCalledWith('mathematical-analysis', 12)
  })

  it('概念层为空时给出说明', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('暂未匹配到概念层')
  })

  it('后端把空列表返回为 null 时仍能渲染', async () => {
    mockedDetail.mockResolvedValue({
      ...detail,
      nodes: null,
      edges: null,
      resources: null,
      concepts: null,
      concept_edges: null,
    } as unknown as KnowledgeDetail)

    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('数学分析')
    expect(wrapper.text()).toContain('暂未匹配到概念层')
  })

  it('有概念时展示概念标签与数量', async () => {
    mockedDetail.mockResolvedValue({
      ...detail,
      point: { ...detail.point, concept_count: 2 },
      concepts: [
        { id: 10, code: 'K_极限_数学', name: '极限', field: '数学', point_code: 'mathematical-analysis', level: 1, explanation: '数列极限' },
        { id: 11, code: 'K_级数_数学', name: '级数', field: '数学', point_code: 'mathematical-analysis', level: 2, explanation: '级数收敛' },
      ],
      concept_edges: [{ from: 10, to: 11 }],
    })

    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('极限')
    expect(wrapper.text()).toContain('级数')
    expect(wrapper.text()).toContain('共 2 个概念')
    expect(wrapper.text()).toContain('概念先修边 1 条')
  })

  it('点击「生成我的学习路径」调用接口并跳转路径页', async () => {
    mockedCreatePath.mockResolvedValue({
      id: 7,
      target_code: 'mathematical-analysis',
      target_name: '数学分析',
      steps: [],
      current_index: 0,
      status: 'active',
    } as unknown as Awaited<ReturnType<typeof createLearningPath>>)

    const wrapper = await mountPage()
    const button = wrapper.findAll('button').find((item) => item.text().includes('生成我的学习路径'))
    expect(button).toBeTruthy()
    await button!.trigger('click')
    await flushPromises()

    expect(mockedCreatePath).toHaveBeenCalledWith('mathematical-analysis')
    expect(pushMock).toHaveBeenCalledWith({ name: 'learning-path', params: { id: 7 } })
  })

  it('默认只展示目标先修子图，旁支学科不出现；可切换显示全部', async () => {
    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('高等数学')
    expect(wrapper.text()).toContain('数学分析')
    // 线性代数与数学分析无先修关系，默认不进入目标子图
    expect(wrapper.text()).not.toContain('线性代数')

    await wrapper.find('.el-switch').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('线性代数')
  })

  it('点击「重新生成概念」调用 engine 并刷新详情', async () => {
    mockedStartGeneration.mockResolvedValue({
      point_code: 'mathematical-analysis',
      status: 'done',
      generated: 11,
      total: 11,
    })

    const wrapper = await mountPage()
    const button = wrapper.findAll('button').find((item) => item.text().includes('重新生成概念'))
    expect(button).toBeTruthy()
    await button!.trigger('click')
    await flushPromises()

    expect(mockedStartGeneration).toHaveBeenCalledWith('mathematical-analysis')
    expect(wrapper.text()).toContain('已生成 11 个概念')
    // 完成后重新拉取详情
    expect(mockedDetail).toHaveBeenCalledTimes(2)
    expect(mockedGenerationStatus).not.toHaveBeenCalled()
  })
})

