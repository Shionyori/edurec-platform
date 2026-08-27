import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { adminListResources, adminListUsers } from '@/api/admin'
import AdminDashboard from '../AdminDashboard.vue'

vi.mock('@/api/admin', () => ({
  adminListResources: vi.fn(),
  adminListUsers: vi.fn(),
}))

const mockedAdminListResources = vi.mocked(adminListResources)
const mockedAdminListUsers = vi.mocked(adminListUsers)

function page(total: number) {
  return { list: [], total, page: 1, page_size: 1 }
}

async function mountPage() {
  const wrapper = mount(AdminDashboard, {
    global: {
      plugins: [ElementPlus],
      stubs: { RouterLink: RouterLinkStub },
    },
  })
  await flushPromises()
  return wrapper
}

describe('AdminDashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedAdminListResources.mockResolvedValue(page(42))
    mockedAdminListUsers.mockResolvedValue(page(7))
  })

  it('拉取资源与用户总数并渲染统计', async () => {
    const wrapper = await mountPage()
    expect(mockedAdminListResources).toHaveBeenCalledWith({ page: 1, page_size: 1 })
    expect(mockedAdminListUsers).toHaveBeenCalledWith({ page: 1, page_size: 1 })
    expect(wrapper.text()).toContain('资源总数')
    expect(wrapper.text()).toContain('42')
    expect(wrapper.text()).toContain('用户总数')
    expect(wrapper.text()).toContain('7')
  })

  it('渲染三个管理入口链接', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('资源管理')
    expect(wrapper.text()).toContain('用户管理')
    expect(wrapper.text()).toContain('分类管理')
    const links = wrapper.findAllComponents(RouterLinkStub)
    expect(links.length).toBeGreaterThanOrEqual(5)
  })
})
