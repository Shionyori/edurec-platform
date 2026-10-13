import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { LearningPlan } from '@/types'
import { getLearningPlan } from '@/api/learningPlan'
import LearningPlanView from '../LearningPlanView.vue'

vi.mock('vue-router', () => ({ RouterLink: { template: '<a><slot /></a>' } }))
vi.mock('@/api/learningPlan', () => ({ getLearningPlan: vi.fn() }))
vi.mock('@/api/behavior', () => ({ recordBehavior: vi.fn(() => Promise.resolve(null)) }))

const mockedGetLearningPlan = vi.mocked(getLearningPlan)

const plan: LearningPlan = {
  goal: '我想入门机器学习',
  matched: true,
  groups: [
    {
      knowledge_point: { id: 1, name: '监督学习', description: '', category: null },
      total: 3,
      resources: [],
    },
  ],
}

const fallbackPlan: LearningPlan = {
  goal: 'zzz',
  matched: false,
  groups: [
    {
      knowledge_point: { id: 0, name: '为你找到的相关资源', description: '', category: null },
      total: 1,
      resources: [],
    },
  ],
}

async function mountView(goal = '我想入门机器学习') {
  const wrapper = mount(LearningPlanView, {
    props: { goal },
    global: { plugins: [ElementPlus] },
  })
  await flushPromises()
  return wrapper
}

describe('LearningPlanView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedGetLearningPlan.mockResolvedValue(plan)
  })

  it('按目标拉取计划并渲染知识点分组', async () => {
    const wrapper = await mountView()
    expect(mockedGetLearningPlan).toHaveBeenCalledWith('我想入门机器学习')
    expect(wrapper.text()).toContain('监督学习')
  })

  it('接口失败时展示错误态与重试入口', async () => {
    mockedGetLearningPlan.mockRejectedValue(new Error('加载失败'))
    const wrapper = await mountView()
    expect(wrapper.text()).toContain('加载失败')
    expect(wrapper.text()).toContain('重试')
  })

  it('未命中知识点时展示兜底说明而不是空白', async () => {
    mockedGetLearningPlan.mockResolvedValue(fallbackPlan)
    const wrapper = await mountView('zzz')
    expect(wrapper.text()).toContain('没有找到对应知识点')
    expect(wrapper.text()).toContain('为你找到的相关资源')
  })

  // 竞态：先发的慢响应不能覆盖后发的结果
  it('目标快速变更时只采用最后一次请求的结果', async () => {
    let resolveSlow: (v: LearningPlan) => void = () => {}
    mockedGetLearningPlan
      .mockImplementationOnce(() => new Promise<LearningPlan>((r) => { resolveSlow = r }))
      .mockResolvedValueOnce({ ...plan, groups: [
        { knowledge_point: { id: 2, name: '线性代数', description: '', category: null }, total: 1, resources: [] },
      ] })

    const wrapper = mount(LearningPlanView, { props: { goal: '慢目标' }, global: { plugins: [ElementPlus] } })
    await wrapper.setProps({ goal: '快目标' })
    await flushPromises()

    resolveSlow(plan) // 慢请求此刻才返回
    await flushPromises()

    expect(wrapper.text()).toContain('线性代数')
    expect(wrapper.text()).not.toContain('监督学习')
  })
})
