import { get } from './client'
import type { BilibiliComment } from '@/types'

export interface CommentListResult {
  list: BilibiliComment[]
}

export function listComments(resourceId: number) {
  return get<CommentListResult>(`/resources/${resourceId}/comments`)
}
