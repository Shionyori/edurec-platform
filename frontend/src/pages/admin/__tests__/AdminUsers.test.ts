import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { User } from '@/types'
import { adminListUsers } from '@/api/admin'
import AdminUsers from '../AdminUsers.vue'

vi.mock('@/api/admin', () => ({ adminListUsers: vi.fn() }))

const mockedAdminListUsers = vi.mocked(adminListUsers)

const user: User = {
  id: 1, username: 'zhangsan', email: 'zhangsan@example.com',
  display_name: '张三', avatar_url: null, is_admin: true,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-01T08:00:00Z',
}

function pageResult(list: User[] = [user], total = 1) {
  return { list, total, page: 1, page_size: 20 }
}

async function mountPage() {
  const wrapper = mount(AdminUsers, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

describe('AdminUsers', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedAdminListUsers.mockResolvedValue(pageResult())
  })

  it('加载并渲染用户表格', async () => {
    const wrapper = await mountPage()
    expect(mockedAdminListUsers).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, keyword: undefined }),
    )
    expect(wrapper.text()).toContain('zhangsan')
    expect(wrapper.text()).toContain('zhangsan@example.com')
    expect(wrapper.text()).toContain('管理员')
  })

  it('输入关键词搜索并重置页码', async () => {
    const wrapper = await mountPage()
    mockedAdminListUsers.mockClear()
    const input = wrapper.find('input[placeholder="按用户名 / 邮箱搜索"]')
    await input.setValue('lisi')
    await input.trigger('keyup.enter')
    await flushPromises()
    expect(mockedAdminListUsers).toHaveBeenLastCalledWith(
      expect.objectContaining({ keyword: 'lisi', page: 1 }),
    )
  })

  it('切页重新拉取对应页码', async () => {
    mockedAdminListUsers.mockResolvedValue(pageResult([user], 25))
    const wrapper = await mountPage()
    mockedAdminListUsers.mockClear()
    const pagination = wrapper.findComponent({ name: 'ElPagination' })
    pagination.vm.$emit('current-change', 2)
    await flushPromises()
    expect(mockedAdminListUsers).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }))
  })

  it('接口失败时展示错误态', async () => {
    mockedAdminListUsers.mockRejectedValue(new Error('加载失败'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('加载失败')
  })
})
