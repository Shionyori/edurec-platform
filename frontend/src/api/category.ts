import { get, post } from './client'
import type { Category } from '@/types'

export interface CreateCategoryPayload {
  name: string
  description?: string
}

export function listCategories() {
  return get<Category[]>('/categories')
}

export function createCategory(payload: CreateCategoryPayload) {
  return post<Category>('/categories', payload)
}
