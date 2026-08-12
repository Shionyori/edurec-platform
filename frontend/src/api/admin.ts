import { get } from './client'
import type { Page, Resource, User } from '@/types'

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
