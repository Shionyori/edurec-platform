import { get } from './client'
import type { RecommendationResult } from '@/types'

export function getRecommendations(limit = 20) {
  return get<RecommendationResult>('/recommendations', { params: { limit } })
}
