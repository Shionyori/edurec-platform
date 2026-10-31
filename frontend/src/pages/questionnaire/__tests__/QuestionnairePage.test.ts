import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { LearningProfileItem } from '@/types'
import { getLearningProfile, submitQuestionnaire } from '@/api/learning'
import QuestionnairePage from '../index.vue'

const { pushMock } = vi.hoisted(() => ({ pushMock: vi.fn() }))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
  RouterLink: { template: '<a><slot /></a>' },
}))
vi.mock('@/api/learning', () => ({
  getLearningProfile: vi.fn(),
  submitQuestionnaire: vi.fn(),
}))

const mockedProfile = vi.mocked(getLearningProfile)
const mockedSubmit = vi.mocked(submitQuestionnaire)

const profile: LearningProfileItem[] = [
  { point_code: 'calculus', name: '高等数学', level: 1, mastery: 1, source: 'questionnaire' },
  { point_code: 'linear-algebra', name: '线性代数', level: 1, mastery: 0.33, source: 'questionnaire' },
  { point_code: 'analysis', name: '数学分析', level: 2, mastery: 0, source: '' },
]

async function mountPage() {
  const wrapper = mount(QuestionnairePage, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

describe('QuestionnairePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedProfile.mockResolvedValue(profile)
    mockedSubmit.mockResolvedValue(profile)
  })

  it('按层级展示知识点并把已有掌握度回填为自评等级', async () => {
    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('初始学习问卷')
    expect(wrapper.text()).toContain('高等数学')
    expect(wrapper.text()).toContain('第 2 层')
    // mastery 1 → 等级 3；0.33 → 1；0 → 0
    const checked = wrapper.findAll('[role="radio"][aria-checked="true"]')
    expect(checked.length).toBeGreaterThanOrEqual(3)
  })

  it('默认只展开第一层，可切换全部展开/收起', async () => {
    const wrapper = await mountPage()

    const headers = wrapper.findAll('section > button[aria-expanded]')
    expect(headers.length).toBe(2)
    expect(headers[0].attributes('aria-expanded')).toBe('true')
    expect(headers[1].attributes('aria-expanded')).toBe('false')

    const expand = wrapper.findAll('button').find((b) => b.text() === '全部展开')
    await expand!.trigger('click')
    expect(headers[1].attributes('aria-expanded')).toBe('true')
  })

  it('「全部按没接触过」把答案重置为 0 并标记为已修改', async () => {
    const wrapper = await mountPage()

    const bulk = wrapper.findAll('button').find((b) => b.text().includes('全部按'))
    await bulk!.trigger('click')

    expect(wrapper.text()).toContain('已修改 2 项')
  })

  it('保存时把自评等级提交给接口并返回知识点页', async () => {
    const wrapper = await mountPage()
    const saveBtn = wrapper.findAll('button').find((b) => b.text().includes('保存并返回'))
    expect(saveBtn).toBeTruthy()
    await saveBtn!.trigger('click')
    await flushPromises()

    expect(mockedSubmit).toHaveBeenCalledTimes(1)
    expect(mockedSubmit.mock.calls[0][0]).toMatchObject({
      calculus: 3,
      'linear-algebra': 1,
      analysis: 0,
    })
    expect(pushMock).toHaveBeenCalledWith({ name: 'knowledge' })
  })

  it('接口失败时展示错误', async () => {
    mockedProfile.mockRejectedValue(new Error('网络错误'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('网络错误')
  })
})
