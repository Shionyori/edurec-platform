import { get, post } from './client'
import type { Behavior, BehaviorAction, Page } from '@/types'

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
