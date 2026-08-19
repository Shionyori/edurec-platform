import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import RatingForm from '../RatingForm.vue'

async function setScore(wrapper: ReturnType<typeof mount>, score: number) {
  await wrapper.findComponent({ name: 'ElRate' }).vm.$emit('update:modelValue', score)
}

describe('RatingForm', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('未选评分提交时提示且不 emit', async () => {
    vi.spyOn(ElMessage, 'warning').mockImplementation(() => ({}) as never)
    const wrapper = mount(RatingForm, { global: { plugins: [ElementPlus] } })
    await wrapper.find('button').trigger('click')
    expect(ElMessage.warning).toHaveBeenCalledWith('请先选择评分')
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('选评分 + 评论后 emit 载荷', async () => {
    const wrapper = mount(RatingForm, { global: { plugins: [ElementPlus] } })
    await setScore(wrapper, 4)
    await wrapper.find('textarea').setValue('  讲解清晰  ')
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({ score: 4, comment: '讲解清晰' })
  })

  it('initialScore / initialComment 回填', () => {
    const wrapper = mount(RatingForm, {
      props: { initialScore: 3, initialComment: '回填评论' },
      global: { plugins: [ElementPlus] },
    })
    expect(wrapper.findComponent({ name: 'ElRate' }).props('modelValue')).toBe(3)
    expect(wrapper.find('textarea').element.value).toBe('回填评论')
  })
})
