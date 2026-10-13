import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus, { ElMessage } from 'element-plus'
import GoalInput from '../GoalInput.vue'

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<typeof import('element-plus')>('element-plus')
  return { ...actual, ElMessage: { warning: vi.fn(), error: vi.fn(), success: vi.fn() } }
})

function mountInput(props: Record<string, unknown> = {}) {
  return mount(GoalInput, { props, global: { plugins: [ElementPlus] } })
}

describe('GoalInput', () => {
  beforeEach(() => {
    vi.mocked(ElMessage.warning).mockClear()
  })

  it('合法目标提交时 emit 去除首尾空白的目标', async () => {
    const wrapper = mountInput()
    await wrapper.find('input').setValue('  我想入门机器学习  ')
    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('submit')).toEqual([['我想入门机器学习']])
  })

  it('空目标不提交并给出提示', async () => {
    const wrapper = mountInput()
    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('submit')).toBeUndefined()
    expect(ElMessage.warning).toHaveBeenCalled()
  })

  it('超长目标不提交并给出提示', async () => {
    const wrapper = mountInput()
    await wrapper.find('input').setValue('学'.repeat(101))
    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('submit')).toBeUndefined()
    expect(ElMessage.warning).toHaveBeenCalled()
  })

  it('initialGoal 变化时回填输入框（URL 同步场景）', async () => {
    const wrapper = mountInput({ initialGoal: '机器学习' })
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('机器学习')

    await wrapper.setProps({ initialGoal: '线性代数' })
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('线性代数')
  })
})
