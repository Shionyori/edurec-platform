import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { User } from '@/types'
import ProfileCard from '@/components/profile/ProfileCard.vue'
import BehaviorHistory from '@/components/profile/BehaviorHistory.vue'
import ProfilePage from '../ProfilePage.vue'

const { fetchMe } = vi.hoisted(() => ({ fetchMe: vi.fn() }))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: null, fetchMe }),
}))

const user: User = {
  id: 1, username: 'zhangsan', email: 'zhangsan@example.com',
  display_name: '张三', avatar_url: null, is_admin: false,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-01T08:00:00Z',
}

async function mountPage() {
  const wrapper = mount(ProfilePage, {
    global: {
      plugins: [ElementPlus],
      stubs: { ProfileCard: true, BehaviorHistory: true },
    },
  })
  await flushPromises()
  return wrapper
}

describe('ProfilePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    fetchMe.mockResolvedValue(user)
  })

  it('拉取用户信息并渲染资料卡片与行为历史', async () => {
    const wrapper = await mountPage()
    expect(fetchMe).toHaveBeenCalled()
    expect(wrapper.findComponent(ProfileCard).exists()).toBe(true)
    expect(wrapper.findComponent(BehaviorHistory).exists()).toBe(true)
  })
})
