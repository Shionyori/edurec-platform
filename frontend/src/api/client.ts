import axios, { AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios'
import { tokenStorage } from '@/utils/token'
import { pageLoading } from '@/composables/pageLoading'

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

// 触发实时爬 B 站的接口，后端单次调用上限 30s（bilibili_online.go 的 pythonTimeout）。
// 客户端超时必须更宽：否则后端已爬完并落库，浏览器却先超时，前端还会把列表标记成已耗尽
export const CRAWL_TIMEOUT = 35000

/** Per-request switches that should not become query parameters. */
export interface ApiRequestConfig extends AxiosRequestConfig {
  /** Background work (comments, crawl polling, impressions) must not hold the page loader. */
  skipPageLoading?: boolean
}

type TrackedConfig = InternalAxiosRequestConfig & {
  skipPageLoading?: boolean
  __pageLoadingTracked?: boolean
}

/* ---------------------------------------------------------------------------
   Navigation prefetch cache
   The router asks for the next page's primary GETs before it commits the
   navigation. This keeps the old page on screen until the new page has data.
   Entries are one-shot and short-lived; they are only written while a
   prefetch session is active.
   --------------------------------------------------------------------------- */
const PREFETCH_TTL_MS = 10000
const prefetchCache = new Map<string, { data: unknown; at: number }>()
let prefetchSession = 0

function stableValue(value: unknown): string {
  if (value === null || typeof value !== 'object') return JSON.stringify(value) ?? 'null'
  if (Array.isArray(value)) return `[${value.map(stableValue).join(',')}]`
  const record = value as Record<string, unknown>
  return `{${Object.keys(record)
    .sort()
    .filter((key) => record[key] !== undefined)
    .map((key) => `${JSON.stringify(key)}:${stableValue(record[key])}`)
    .join(',')}}`
}

function cacheKey(url: string, params?: unknown): string {
  return `${url}|${stableValue(params)}`
}

function prunePrefetchCache() {
  if (prefetchCache.size < 100) return
  const now = Date.now()
  for (const [key, entry] of prefetchCache) {
    if (now - entry.at > PREFETCH_TTL_MS) prefetchCache.delete(key)
  }
}

function putPrefetch(key: string, data: unknown) {
  prunePrefetchCache()
  prefetchCache.set(key, { data, at: Date.now() })
}

function takePrefetch(key: string): { hit: true; data: unknown } | null {
  const entry = prefetchCache.get(key)
  if (!entry) return null
  prefetchCache.delete(key)
  if (Date.now() - entry.at > PREFETCH_TTL_MS) return null
  return { hit: true, data: entry.data }
}

export function clearPrefetchCache(): void {
  prefetchCache.clear()
}

export function startPrefetchSession(): number {
  prefetchSession += 1
  return prefetchSession
}

export function stopPrefetchSession(session: number): void {
  if (prefetchSession === session) prefetchSession = 0
}

client.interceptors.request.use((config) => {
  const tracked = config as TrackedConfig
  if (!tracked.skipPageLoading) {
    pageLoading.beginRequest()
    tracked.__pageLoadingTracked = true
  }
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
      pageLoading.beginRequest()
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
        pageLoading.endRequest()
        refreshing = null
      }
    })()
  }
  return refreshing
}

client.interceptors.response.use(
  (resp) => {
    const tracked = resp.config as TrackedConfig
    if (tracked.__pageLoadingTracked) pageLoading.endRequest()
    return resp
  },
  async (error) => {
    const tracked = error.config as TrackedConfig | undefined
    if (tracked?.__pageLoadingTracked) pageLoading.endRequest()

    const original = error.config as (InternalAxiosRequestConfig & { _retried?: boolean }) | undefined
    // 认证类接口的 401 是凭证错误（而非 access token 过期），不应触发 token 续期与整页跳转
    const url = original?.url ?? ''
    const isAuthRequest = url.includes('/auth/login') || url.includes('/auth/register') || url.includes('/auth/refresh')
    if (error.response?.status === 401 && original && !original._retried && !isAuthRequest) {
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

export async function get<T>(url: string, config: ApiRequestConfig = {}): Promise<T> {
  const key = cacheKey(url, config.params)
  const sessionAtStart = prefetchSession

  if (!sessionAtStart) {
    const cached = takePrefetch(key)
    if (cached) return cached.data as T
  }

  const resp = await client.get<ApiResponse<T>>(url, config)
  const data = unwrap(resp.data)

  if (sessionAtStart && sessionAtStart === prefetchSession) {
    putPrefetch(key, data)
  }
  return data
}

export async function post<T>(url: string, data?: unknown, config: ApiRequestConfig = {}): Promise<T> {
  const resp = await client.post<ApiResponse<T>>(url, data, config)
  return unwrap(resp.data)
}

export async function put<T>(url: string, data?: unknown, config: ApiRequestConfig = {}): Promise<T> {
  const resp = await client.put<ApiResponse<T>>(url, data, config)
  return unwrap(resp.data)
}

export async function del<T>(url: string, config: ApiRequestConfig = {}): Promise<T> {
  const resp = await client.delete<ApiResponse<T>>(url, config)
  return unwrap(resp.data)
}
