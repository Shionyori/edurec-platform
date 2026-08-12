import axios, { AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios'
import { tokenStorage } from '@/utils/token'

export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export function unwrap<T>(body: ApiResponse<T>): T {
  if (body.code !== 0) {
    throw new Error(body.message || '请求失败')
  }
  return body.data
}

export const client = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

client.interceptors.request.use((config) => {
  const token = tokenStorage.getAccess()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let refreshing: Promise<boolean> | null = null

async function refreshToken(): Promise<boolean> {
  if (!refreshing) {
    refreshing = (async () => {
      const rt = tokenStorage.getRefresh()
      if (!rt) return false
      try {
        const resp = await axios.post<ApiResponse<{ access_token: string; refresh_token: string }>>(
          '/api/v1/auth/refresh',
          { refresh_token: rt },
        )
        tokenStorage.setAccess(resp.data.data.access_token)
        tokenStorage.setRefresh(resp.data.data.refresh_token)
        return true
      } catch {
        tokenStorage.clear()
        return false
      } finally {
        refreshing = null
      }
    })()
  }
  return refreshing
}

client.interceptors.response.use(
  (resp) => resp,
  async (error) => {
    const original = error.config as (InternalAxiosRequestConfig & { _retried?: boolean }) | undefined
    if (error.response?.status === 401 && original && !original._retried) {
      const ok = await refreshToken()
      if (ok) {
        // 请求拦截器会在重试时自动附带最新 access token
        original._retried = true
        return client(original)
      }
      window.location.href = '/login'
    }
    const message = error.response?.data?.message ?? error.message ?? '网络错误，请稍后重试'
    return Promise.reject(new Error(message))
  },
)

export async function get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
  const resp = await client.get<ApiResponse<T>>(url, config)
  return unwrap(resp.data)
}

export async function post<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  const resp = await client.post<ApiResponse<T>>(url, data, config)
  return unwrap(resp.data)
}

export async function put<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  const resp = await client.put<ApiResponse<T>>(url, data, config)
  return unwrap(resp.data)
}

export async function del<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
  const resp = await client.delete<ApiResponse<T>>(url, config)
  return unwrap(resp.data)
}
