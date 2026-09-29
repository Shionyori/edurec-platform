import { get, put } from './client'
import type { User } from '@/types'

export interface UpdateMePayload {
  display_name?: string
  avatar_url?: string
}

export function getMe() {
  return get<User>('/users/me')
}

export function updateMe(payload: UpdateMePayload) {
  return put<Partial<User>>('/users/me', payload)
}

// 保存冷启动兴趣分类（覆盖式）。新用户无行为时，平台兜底推荐会优先取这些分类。
export function updateInterests(categoryIds: number[]) {
  return put<User>('/users/me/interests', { category_ids: categoryIds })
}
