import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { adminGetRecommendationStats, adminListRecommendationRuns } from '@/api/admin'
import AdminRecommendations from '../AdminRecommendations.vue'

vi.mock('@/api/admin', () => ({
  adminGetRecommendationStats: vi.fn(),
  adminListRecommendationRuns: vi.fn(),
}))

const mockedStats = vi.mocked(adminGetRecommendationStats)
const mockedRuns = vi.mocked(adminListRecommendationRuns)

async function mountPage() {
  const wrapper = mount(AdminRecommendations, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

describe('AdminRecommendations', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedStats.mockResolvedValue({
      impressions: 200, clicks: 10, favorites: 3, views: 50,
      recommendation_users: 4, click_through_rate: 0.05,
    })
    mockedRuns.mockResolvedValue({
      list: [{
        id: 1, run_id: '20260101_000000', snapshot_run_id: '20251231_000000',
        model_name: 'semantic_deterministic_two_tower', model_version: 'v1', encoder: 'tfidf',
        generated_at: 1700000000, top_n: 20, users_count: 4,
        imported_users: 4, skipped_users: 0, imported_resources: 80, skipped_resources: 0,
        created_at: 1700000001,
      }],
    })
  })

  it('渲染推荐效果统计与 CTR', async () => {
    const wrapper = await mountPage()
    expect(mockedStats).toHaveBeenCalled()
    expect(wrapper.text()).toContain('曝光')
    expect(wrapper.text()).toContain('点击率 CTR')
    expect(wrapper.text()).toContain('5.00%')
    expect(wrapper.text()).toContain('200')
  })

  it('渲染运行记录（可追溯）', async () => {
    const wrapper = await mountPage()
    expect(mockedRuns).toHaveBeenCalledWith(20)
    expect(wrapper.text()).toContain('20260101_000000')
    expect(wrapper.text()).toContain('semantic_deterministic_two_tower')
  })
})
