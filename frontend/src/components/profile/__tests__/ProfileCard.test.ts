import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { User } from '@/types'
import { updateMe } from '@/api/user'
import ProfileCard from '../ProfileCard.vue'

vi.mock('@/api/user', () => ({ updateMe: vi.fn() }))

const mockedUpdateMe = vi.mocked(updateMe)

const user: User = {
  id: 1, username: 'zhangsan', email: 'zhangsan@example.com',
  display_name: '张三', avatar_url: null, is_admin: false,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-01T08:00:00Z',
}

function mountCard() {
  return mount(ProfileCard, {
    props: { user },
    global: { plugins: [ElementPlus] },
  })
}

async function clickButton(wrapper: ReturnType<typeof mountCard>, text: string) {
  const btn = wrapper.findAll('button').find((b) => b.text().includes(text))
  expect(btn, `button "${text}"`).toBeTruthy()
  await btn!.trigger('click')
}

describe('ProfileCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('渲染用户资料', () => {
    const wrapper = mountCard()
    expect(wrapper.text()).toContain('张三')
    expect(wrapper.text()).toContain('@zhangsan · zhangsan@example.com')
    expect(wrapper.text()).toContain('编辑资料')
  })

  it('编辑资料调用 updateMe 并触发 updated', async () => {
    mockedUpdateMe.mockResolvedValue({
      id: 1,
      display_name: '新昵称',
      avatar_url: 'https://example.com/a.png',
      updated_at: '2026-08-01T00:00:00Z',
    })
    const wrapper = mountCard()

    await clickButton(wrapper, '编辑资料')
    await wrapper.findAll('input')[0].setValue('新昵称')
    await clickButton(wrapper, '保存')
    await flushPromises()

    expect(mockedUpdateMe).toHaveBeenCalledWith(
      expect.objectContaining({ display_name: '新昵称', avatar_url: undefined }),
    )
    const emitted = wrapper.emitted('updated')
    expect(emitted).toBeTruthy()
    expect(emitted![0][0]).toEqual(expect.objectContaining({ display_name: '新昵称' }))
  })

  it('保存失败展示错误并保留编辑态', async () => {
    mockedUpdateMe.mockRejectedValue(new Error('保存失败'))
    const wrapper = mountCard()

    await clickButton(wrapper, '编辑资料')
    await clickButton(wrapper, '保存')
    await flushPromises()

    expect(wrapper.text()).toContain('保存失败')
    expect(wrapper.text()).toContain('取消') // 仍在编辑态
  })
})
