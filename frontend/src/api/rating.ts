import { get, post } from './client'
import type { Page, Rating } from '@/types'

export interface RatingQuery {
  page?: number
  page_size?: number
}

export interface UpsertRatingPayload {
  score: number
  comment?: string
}

export function listRatings(resourceId: number, params: RatingQuery = {}) {
  return get<Page<Rating>>(`/resources/${resourceId}/ratings`, { params })
}

export function upsertRating(resourceId: number, payload: UpsertRatingPayload) {
  return post<Rating>(`/resources/${resourceId}/ratings`, payload)
}
