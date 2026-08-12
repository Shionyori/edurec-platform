import { get, post } from './client'
import type { LoginResult, RefreshResult, RegisterPayload } from '@/types'

export interface LoginPayload {
  username: string
  password: string
}

export function login(payload: LoginPayload) {
  return post<LoginResult>('/auth/login', payload)
}

export function register(payload: RegisterPayload) {
  return post<{ user: LoginResult['user'] }>('/auth/register', payload)
}

export function refresh(refreshToken: string) {
  return post<RefreshResult>('/auth/refresh', { refresh_token: refreshToken })
}
