import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { Category } from '@/types'
import { createResource } from '@/api/resource'
import ResourceUploadForm from '../ResourceUploadForm.vue'

vi.mock('@/api/resource', () => ({ createResource: vi.fn() }))

const mockedCreateResource = vi.mocked(createResource)

const categories: Category[] = [
  { id: 1, name: '人工智能', description: '' },
  { id: 2, name: '后端开发', description: '' },
]

function mountForm() {
  return mount(ResourceUploadForm, {
    props: { categories },
    global: { plugins: [ElementPlus] },
  })
}

async function submitButton(wrapper: ReturnType<typeof mountForm>) {
  const btn = wrapper.findAll('button').find((b) => b.text().includes('提交'))
  expect(btn, '提交 button').toBeTruthy()
  await btn!.trigger('click')
}

describe('ResourceUploadForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('空表单提交时校验失败且不调用接口', async () => {
    const wrapper = mountForm()
    await submitButton(wrapper)
    expect(mockedCreateResource).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('请填写标题')
  })

  it('填写必填项后提交，tags 解析为数组并触发 created', async () => {
    mockedCreateResource.mockResolvedValue({
      id: 3,
      title: '深度学习实战',
      type: 'course',
      created_at: '2026-08-01T00:00:00Z',
    } as never)
    const wrapper = mountForm()

    await wrapper.find('input[placeholder="请填写标题"]').setValue('深度学习实战')
    await wrapper.find('textarea[placeholder="请填写描述"]').setValue('从入门到实战')

    const selects = wrapper.findAllComponents({ name: 'ElSelect' })
    await selects[0].vm.$emit('update:modelValue', 'course') // 类型
    await selects[1].vm.$emit('update:modelValue', 1) // 分类

    await wrapper.find('input[placeholder="例如：Python, 数据分析"]').setValue('Python, 深度学习')
    await wrapper.find('input[placeholder="选填"]').setValue('吴老师')

    await submitButton(wrapper)
    await flushPromises()

    expect(mockedCreateResource).toHaveBeenCalledWith(
      expect.objectContaining({
        title: '深度学习实战',
        description: '从入门到实战',
        type: 'course',
        category_id: 1,
        tags: ['Python', '深度学习'],
      }),
    )
    expect(wrapper.emitted('created')).toBeTruthy()
    expect(wrapper.emitted('created')![0][0]).toEqual({ id: 3 })
  })

  it('提交失败时展示错误态', async () => {
    mockedCreateResource.mockRejectedValue(new Error('创建失败'))
    const wrapper = mountForm()

    await wrapper.find('input[placeholder="请填写标题"]').setValue('深度学习实战')
    await wrapper.find('textarea[placeholder="请填写描述"]').setValue('描述')
    const selects = wrapper.findAllComponents({ name: 'ElSelect' })
    await selects[0].vm.$emit('update:modelValue', 'course')
    await selects[1].vm.$emit('update:modelValue', 1)

    await submitButton(wrapper)
    await flushPromises()

    expect(wrapper.text()).toContain('创建失败')
    expect(wrapper.emitted('created')).toBeFalsy()
  })
})
