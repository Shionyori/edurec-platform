import { del, get, post } from './client'
import type { Behavior, BehaviorAction, Page, Resource } from '@/types'

export interface BehaviorQuery {
  action?: BehaviorAction
  page?: number
  page_size?: number
}

export function recordBehavior(resourceId: number, action: BehaviorAction) {
  return post<null>(`/resources/${resourceId}/behaviors`, { action })
}

export function listMyBehaviors(params: BehaviorQuery = {}) {
  return get<Page<Behavior>>('/users/me/behaviors', { params })
}

// 收藏 / 取消收藏（幂等）。后端以 favorite 行为为收藏的依据。
export function setFavorite(resourceId: number, favorite: boolean) {
  return favorite
    ? post<{ favorited: boolean }>(`/resources/${resourceId}/favorite`)
    : del<{ favorited: boolean }>(`/resources/${resourceId}/favorite`)
}

// 我的收藏：分页返回收藏的资源（按最近收藏倒序）
export function listFavorites(params: { page?: number; page_size?: number } = {}) {
  return get<Page<Resource>>('/users/me/favorites', { params })
}
