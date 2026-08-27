import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { Category } from '@/types'
import { listCategories } from '@/api/category'
import ResourceUploadForm from '@/components/resource/ResourceUploadForm.vue'
import UploadPage from '../UploadPage.vue'

const { push } = vi.hoisted(() => ({ push: vi.fn() }))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}))
vi.mock('@/api/category', () => ({ listCategories: vi.fn() }))

const mockedListCategories = vi.mocked(listCategories)

const categories: Category[] = [{ id: 1, name: '人工智能', description: '' }]

async function mountPage() {
  const wrapper = mount(UploadPage, {
    global: {
      plugins: [ElementPlus],
      stubs: { ResourceUploadForm: true },
    },
  })
  await flushPromises()
  return wrapper
}

describe('UploadPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedListCategories.mockResolvedValue(categories)
  })

  it('加载分类并渲染上传表单', async () => {
    const wrapper = await mountPage()
    expect(mockedListCategories).toHaveBeenCalled()
    expect(wrapper.findComponent(ResourceUploadForm).exists()).toBe(true)
  })

  it('表单创建成功后跳转资源详情', async () => {
    const wrapper = await mountPage()
    const form = wrapper.findComponent(ResourceUploadForm)
    form.vm.$emit('created', { id: 7 })
    await flushPromises()
    expect(push).toHaveBeenCalledWith({ name: 'resource-detail', params: { id: 7 } })
  })
})
