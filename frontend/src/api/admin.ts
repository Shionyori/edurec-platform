import { get } from './client'
import type { Page, RecommendationRun, RecommendationStats, Resource, User } from '@/types'

export interface AdminQuery {
  keyword?: string
  type?: string
  category_id?: number
  page?: number
  page_size?: number
}

export function adminListUsers(params: AdminQuery = {}) {
  return get<Page<User>>('/admin/users', { params })
}

export function adminListResources(params: AdminQuery = {}) {
  return get<Page<Resource>>('/admin/resources', { params })
}

// 推荐运行记录（可追溯：哪份快照、哪个模型、何时生成）
export function adminListRecommendationRuns(limit = 20) {
  return get<{ list: RecommendationRun[] }>('/admin/recommendation-runs', { params: { limit } })
}

// 推荐效果看板：曝光 / 点击 / 收藏 / 覆盖率与 CTR
export function adminGetRecommendationStats() {
  return get<RecommendationStats>('/admin/recommendation-stats')
}
