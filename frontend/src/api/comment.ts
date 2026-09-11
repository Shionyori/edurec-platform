import { CRAWL_TIMEOUT, get } from './client'
import type { BilibiliComment } from '@/types'

export interface CommentListResult {
  list: BilibiliComment[]
}

export function listComments(resourceId: number) {
  // 无缓存时后端会实时爬 B 站评论，需要放宽超时
  return get<CommentListResult>(`/resources/${resourceId}/comments`, { timeout: CRAWL_TIMEOUT })
}
