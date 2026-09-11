import { describe, expect, it } from 'vitest'
import { db } from '../db'
import { DATA_CONFIG } from '../generator'

describe('mock db 种子数据', () => {
  it('核心实体 id 稳定（现有测试依赖）', () => {
    expect(db.users[0].username).toBe('admin')
    expect(db.users[1].username).toBe('user')
    expect(db.categories[0].name).toBe('人工智能')
    expect(db.resources[0].id).toBe(1)
    expect(db.resources[0].title).toBe('机器学习入门')
  })

  it('用户 4 个、分类 6 个', () => {
    expect(db.users).toHaveLength(4)
    expect(db.categories).toHaveLength(6)
  })

  it('资源总数 = 精选 + 生成数量', () => {
    expect(db.resources).toHaveLength(6 + DATA_CONFIG.extraResourceCount)
  })

  it('热门资源 1 评分数达到分页演示阈值', () => {
    const count = db.ratings.filter((r) => r.resource_id === 1).length
    expect(count).toBeGreaterThanOrEqual(DATA_CONFIG.hotResourceRatingTarget)
  })

  it('生成资源无重复 id，且 id 全部顺延', () => {
    const ids = db.resources.map((r) => r.id)
    expect(new Set(ids).size).toBe(ids.length)
    expect(ids).toEqual([...ids].sort((a, b) => a - b))
  })
})
