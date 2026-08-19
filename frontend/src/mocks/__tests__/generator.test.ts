import { describe, expect, it } from 'vitest'
import { DATA_CONFIG, generateMockData } from '../generator'
import type { DbResource, DbUser } from '../db'

const baseUsers: DbUser[] = [
  { id: 1, username: 'admin', email: 'admin@edurec.dev', password: 'admin123', display_name: null, avatar_url: null, is_admin: true, created_at: '2026-07-01T08:00:00Z' },
  { id: 2, username: 'user', email: 'user@edurec.dev', password: 'user123', display_name: '张三', avatar_url: null, is_admin: false, created_at: '2026-07-05T08:00:00Z' },
]

const baseResources: DbResource[] = [
  {
    id: 1, title: '机器学习入门', description: '零基础入门', cover_url: null, type: 'course',
    category_id: 1, tags: ['AI'], metadata: {}, author: '吴恩达', source_url: null,
    avg_rating: 4.5, view_count: 100, created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-01T08:00:00Z',
  },
]

describe('generateMockData', () => {
  it('同一 seed 两次生成结果一致（确定性）', () => {
    const a = generateMockData({ users: baseUsers, resources: baseResources })
    const b = generateMockData({ users: baseUsers, resources: baseResources })
    expect(a).toEqual(b)
  })

  it('补充资源数量与 id 顺延符合配置', () => {
    const result = generateMockData(
      { users: baseUsers, resources: baseResources },
      { extraResourceCount: 2, minRatingsPerResource: 1, maxRatingsPerResource: 1, hotResourceIds: [], hotResourceRatingTarget: 0 },
    )
    expect(result.resources).toHaveLength(2)
    expect(result.resources[0].id).toBe(2)
    expect(result.resources[1].id).toBe(3)
  })

  it('热门资源堆评分达到目标数（分页演示）', () => {
    const result = generateMockData(
      { users: baseUsers, resources: baseResources },
      { extraResourceCount: 0, hotResourceIds: [1], hotResourceRatingTarget: 14 },
    )
    expect(result.ratings.filter((r) => r.resource_id === 1)).toHaveLength(14)
  })

  it('生成的资源具有重算后的 avg_rating 与正 view_count', () => {
    const result = generateMockData(
      { users: baseUsers, resources: baseResources },
      { extraResourceCount: 1, minRatingsPerResource: 3, maxRatingsPerResource: 3, hotResourceIds: [], hotResourceRatingTarget: 0 },
    )
    const gen = result.resources[0]
    const rs = result.ratings.filter((r) => r.resource_id === gen.id)
    expect(rs).toHaveLength(3)
    expect(gen.avg_rating).toBeGreaterThan(0)
    expect(gen.avg_rating).toBeLessThanOrEqual(5)
    expect(gen.view_count).toBeGreaterThanOrEqual(0)
  })
})
