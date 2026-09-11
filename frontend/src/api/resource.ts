import { CRAWL_TIMEOUT, del, get, post, put } from './client'
import type { Page, Resource } from '@/types'

export interface ResourceQuery {
  page?: number
  page_size?: number
  keyword?: string
  category_id?: number
  type?: string
  sort?: 'latest' | 'popular' | 'rating'
  tags?: string
  online_page?: number
}

export function listResources(params: ResourceQuery = {}) {
  // 带 online_page 时会实时爬 B 站，需要放宽超时
  if (params.online_page) {
    return get<Page<Resource>>('/resources', { params, timeout: CRAWL_TIMEOUT })
  }
  return get<Page<Resource>>('/resources', { params })
}

export function getResource(id: number) {
  return get<Resource>(`/resources/${id}`)
}

export function createResource(data: Record<string, unknown>) {
  return post<Resource>('/resources', data)
}

export function updateResource(id: number, data: Record<string, unknown>) {
  return put<Resource>(`/resources/${id}`, data)
}

export function deleteResource(id: number) {
  return del<null>(`/resources/${id}`)
}
