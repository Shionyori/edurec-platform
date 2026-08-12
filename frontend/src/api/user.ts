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
