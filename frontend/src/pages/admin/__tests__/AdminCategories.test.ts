import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { Category } from '@/types'
import { createCategory, listCategories } from '@/api/category'
import AdminCategories from '../AdminCategories.vue'

vi.mock('@/api/category', () => ({ listCategories: vi.fn(), createCategory: vi.fn() }))

const mockedListCategories = vi.mocked(listCategories)
const mockedCreateCategory = vi.mocked(createCategory)

const categories: Category[] = [
  { id: 1, name: '人工智能', description: 'AI、机器学习、深度学习' },
  { id: 2, name: '前端开发', description: 'HTML、CSS、JavaScript' },
]

async function mountPage() {
  const wrapper = mount(AdminCategories, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

describe('AdminCategories', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedListCategories.mockResolvedValue(categories)
  })

  it('加载并渲染分类列表', async () => {
    const wrapper = await mountPage()
    expect(mockedListCategories).toHaveBeenCalled()
    expect(wrapper.text()).toContain('人工智能')
    expect(wrapper.text()).toContain('前端开发')
  })

  it('新建分类成功后清空表单并刷新列表', async () => {
    mockedCreateCategory.mockResolvedValue({ id: 3, name: '数据科学', description: '' })
    const wrapper = await mountPage()
    mockedListCategories.mockClear()
    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('数据科学')
    await inputs[1].setValue('统计、数据')
    const btn = wrapper.findAll('button').find((b) => b.text().includes('创建'))
    await btn!.trigger('click')
    await flushPromises()

    expect(mockedCreateCategory).toHaveBeenCalledWith({ name: '数据科学', description: '统计、数据' })
    expect(mockedListCategories).toHaveBeenCalled()
    expect((inputs[0].element as HTMLInputElement).value).toBe('')
  })

  it('名称为空时不调用接口并提示错误', async () => {
    const wrapper = await mountPage()
    const btn = wrapper.findAll('button').find((b) => b.text().includes('创建'))
    await btn!.trigger('click')
    await flushPromises()
    expect(mockedCreateCategory).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('分类名称不能为空')
  })

  it('列表加载失败时展示错误态', async () => {
    mockedListCategories.mockRejectedValue(new Error('加载失败'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('加载失败')
  })
})
