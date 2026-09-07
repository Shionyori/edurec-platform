# 前端脚手架 + 基础设施 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 搭建 edurec-platform 前端工程地基，使 `pnpm dev` 能启动、首页能通过 MSW mock 加载出符合后端接口结构的数据。

**Architecture:** Vite + Vue 3 + TypeScript 单页应用。基础设施包括 Axios 封装（拦截器链、统一响应解包、401 自动刷新）、MSW mock（覆盖全部 19 个接口）、Pinia auth store、路由守卫、FrontLayout/AdminLayout 两套布局骨架。

**Tech Stack:** Vue 3.5+ / TypeScript / Vite / Pinia / Vue Router 4 / Element Plus / Tailwind CSS v4 / Axios / MSW v2 / Vitest

## Global Constraints

- **提交规则（CLAUDE.md）**：每个 Task 结束时 commit 一次，commit 后**必须停下等待用户确认**，用户说"继续"后再执行下一个 Task。
- **分支**：所有工作都在 `feat/frontend-scaffolding` 分支上进行。
- **提交信息格式**：`<type>(frontend): <subject>`，如 `chore(frontend): 初始化工程、主题与类型定义`。
- **工作目录**：所有前端文件在 `frontend/` 下。
- **统一响应格式**：所有 API 请求/响应使用 `{ code, message, data }` 结构，与后端一致。
- **Element Plus 引入方式（对 spec 的明确偏离）**：设计文档写"按需导入"，但为保证 CSS 变量主题覆盖的确定性，本计划采用**全量引入**（`app.use(ElementPlus)` + `import 'element-plus/dist/index.css'`）。原因：按需导入的组件样式注入顺序不稳定，会导致 `:root` 上的 `--el-*` 主题覆盖失效。功能完成后可低成本切换为按需导入。
- **环境变量**：`VITE_MOCK=1`（默认开）启用 MSW mock；`VITE_API_BASE_URL` 预留真实后端地址（本阶段不使用）。
- **开发端口**：5173；Vite 代理 `/api` → `http://localhost:8080`（真实联调时使用）。

---

### Task 1: 工程初始化、主题与类型定义

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/vitest.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/eslint.config.js`
- Create: `frontend/.prettierrc.json`
- Create: `frontend/.gitignore`
- Create: `frontend/index.html`
- Create: `frontend/src/env.d.ts`
- Create: `frontend/src/main.ts`
- Create: `frontend/src/App.vue`
- Create: `frontend/src/styles/index.css`
- Create: `frontend/src/styles/theme.ts`
- Create: `frontend/src/types/index.ts`
- Create: `frontend/src/test/setup.ts`
- Create: `frontend/public/favicon.svg`

**Interfaces:**
- Consumes: 无
- Produces: `src/types/index.ts` 导出 `User`、`Category`、`Resource`、`Rating`、`Behavior`、`Page<T>`、`LoginResult`、`RegisterPayload` —— 后续所有 Task 使用的公共类型。`src/styles/theme.ts` 导出 `themeTokens`。

- [ ] **Step 1: 创建 package.json 并安装依赖**

`frontend/package.json`：

```json
{
  "name": "edurec-frontend",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc --noEmit && vite build",
    "preview": "vite preview",
    "type-check": "vue-tsc --noEmit",
    "lint": "eslint . --fix",
    "test": "vitest run",
    "test:watch": "vitest"
  }
}
```

运行：

```bash
cd frontend
pnpm add vue vue-router pinia element-plus @element-plus/icons-vue axios
pnpm add -D vite @vitejs/plugin-vue typescript vue-tsc @types/node \
  tailwindcss @tailwindcss/vite \
  eslint eslint-plugin-vue @vue/eslint-config-typescript @vue/eslint-config-prettier prettier \
  vitest @vue/test-utils jsdom msw
```

- [ ] **Step 2: 创建构建配置**

`frontend/vite.config.ts`：

```ts
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
```

`frontend/vitest.config.ts`：

```ts
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['src/test/setup.ts'],
  },
})
```

`frontend/tsconfig.json`：

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "isolatedModules": true,
    "resolveJsonModule": true,
    "esModuleInterop": true,
    "useDefineForClassFields": true,
    "jsx": "preserve",
    "types": ["vite/client", "vitest/globals", "node"],
    "baseUrl": ".",
    "paths": { "@/*": ["src/*"] }
  },
  "include": ["src/**/*.ts", "src/**/*.d.ts", "src/**/*.vue", "vite.config.ts", "vitest.config.ts"]
}
```

- [ ] **Step 3: 创建 lint / 格式 / gitignore**

`frontend/eslint.config.js`：

```js
import pluginVue from 'eslint-plugin-vue'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import skipFormatting from '@vue/eslint-config-prettier/skip-formatting'

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,tsx,vue}'],
  },
  {
    name: 'app/files-to-ignore',
    ignores: ['**/dist/**', '**/coverage/**', '**/public/mockServiceWorker.js'],
  },
  pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,
  skipFormatting,
)
```

`frontend/.prettierrc.json`：

```json
{
  "semi": false,
  "singleQuote": true,
  "printWidth": 100
}
```

`frontend/.gitignore`：

```
node_modules/
dist/
coverage/
*.local
.DS_Store
```

- [ ] **Step 4: 创建 HTML 入口、环境声明、应用骨架**

`frontend/index.html`：

```html
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <link rel="icon" type="image/svg+xml" href="/favicon.svg" />
    <title>edurec 教育资源推荐平台</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

`frontend/public/favicon.svg`：

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
  <rect width="32" height="32" rx="8" fill="#4f46e5"/>
  <text x="16" y="22" font-family="system-ui" font-size="16" font-weight="700" fill="#fff" text-anchor="middle">E</text>
</svg>
```

`frontend/src/env.d.ts`：

```ts
/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_MOCK: string
  readonly VITE_API_BASE_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
```

`frontend/src/main.ts`（本阶段最小版，Task 6 补全）：

```ts
import { createApp } from 'vue'
import App from './App.vue'
import './styles/index.css'

createApp(App).mount('#app')
```

`frontend/src/App.vue`：

```vue
<template>
  <div class="flex min-h-screen items-center justify-center">
    <h1 class="text-2xl font-bold">edurec 前端脚手架</h1>
  </div>
</template>
```

`frontend/src/test/setup.ts`：

```ts
import { beforeEach } from 'vitest'

beforeEach(() => {
  localStorage.clear()
})
```

- [ ] **Step 5: 创建主题 tokens 与全局样式**

`frontend/src/styles/theme.ts`：

```ts
export const themeTokens = {
  color: {
    primary: '#4f46e5',
    primaryHover: '#6366f1',
    primaryActive: '#4338ca',
    bg: '#f8fafc',
    surface: '#ffffff',
    border: '#e2e8f0',
    text: '#0f172a',
    textSecondary: '#64748b',
    textMuted: '#94a3b8',
    success: '#16a34a',
    warning: '#d97706',
    danger: '#dc2626',
  },
  radius: {
    sm: '0.375rem',
    md: '0.5rem',
    lg: '0.75rem',
    full: '9999px',
  },
  font: {
    sans: "system-ui, -apple-system, 'Segoe UI', Roboto, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif",
  },
}
```

`frontend/src/styles/index.css`：

```css
@import 'tailwindcss';

@theme {
  --color-primary: #4f46e5;
  --color-primary-hover: #6366f1;
  --color-primary-active: #4338ca;
  --color-bg: #f8fafc;
  --color-surface: #ffffff;
  --color-border: #e2e8f0;
  --color-ink: #0f172a;
  --color-ink-secondary: #64748b;
  --color-ink-muted: #94a3b8;
}

:root {
  --el-color-primary: #4f46e5;
  --el-color-primary-light-3: #7c7ff0;
  --el-color-primary-light-5: #a5a8f5;
  --el-color-primary-light-7: #cfd0fa;
  --el-color-primary-light-8: #e3e3fc;
  --el-color-primary-light-9: #f1f1fd;
  --el-color-primary-dark-2: #4338ca;
  --el-border-radius-base: 6px;
}

body {
  @apply bg-bg text-ink;
  font-family: system-ui, -apple-system, 'Segoe UI', Roboto, 'PingFang SC',
    'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
  -webkit-font-smoothing: antialiased;
}
```

- [ ] **Step 6: 创建公共类型定义**

`frontend/src/types/index.ts`：

```ts
export interface User {
  id: number
  username: string
  email: string
  display_name: string | null
  avatar_url: string | null
  is_admin?: boolean
  created_at: string
  updated_at?: string
}

export interface Category {
  id: number
  name: string
  description: string
}

export type ResourceType = 'course' | 'article' | 'video'

export interface Resource {
  id: number
  title: string
  description: string
  cover_url: string | null
  type: ResourceType
  category: Pick<Category, 'id' | 'name'> | null
  tags: string[]
  metadata: Record<string, unknown>
  author: string | null
  source_url: string | null
  avg_rating: number
  view_count: number
  created_at: string
  updated_at: string
}

export interface Rating {
  id: number
  user: Pick<User, 'id' | 'username' | 'display_name' | 'avatar_url'>
  score: number
  comment: string | null
  created_at: string
}

export type BehaviorAction = 'view' | 'click' | 'favorite'

export interface Behavior {
  id: number
  resource: Pick<Resource, 'id' | 'title' | 'cover_url' | 'type'>
  action: BehaviorAction
  created_at: string
}

export interface Page<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface LoginResult {
  access_token: string
  refresh_token: string
  expires_in: number
  user: User
}

export interface RefreshResult {
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface RegisterPayload {
  username: string
  email: string
  password: string
  display_name?: string
}

export interface RecommendationResult {
  list: Resource[]
  updated_at: string
}
```

- [ ] **Step 7: 验证构建**

```bash
cd frontend
pnpm build
```

Expected: `vue-tsc` 无类型错误，`vite build` 产出 `dist/`。

- [ ] **Step 8: 验证 dev 可启动**

```bash
cd frontend
pnpm dev
```

Expected: 终端显示 `Local: http://localhost:5173/`，浏览器访问显示 "edurec 前端脚手架"。按 Ctrl+C 停止。

- [ ] **Step 9: Commit**

```bash
git add frontend/
git commit -m "chore(frontend): 初始化工程、主题与类型定义"
```

---

### Task 2: API 客户端与接口封装

**Files:**
- Create: `frontend/src/utils/token.ts`
- Create: `frontend/src/api/client.ts`
- Create: `frontend/src/api/auth.ts`
- Create: `frontend/src/api/user.ts`
- Create: `frontend/src/api/category.ts`
- Create: `frontend/src/api/resource.ts`
- Create: `frontend/src/api/rating.ts`
- Create: `frontend/src/api/behavior.ts`
- Create: `frontend/src/api/admin.ts`
- Test: `frontend/src/api/__tests__/client.test.ts`

**Interfaces:**
- Consumes: `src/types/index.ts`（Task 1）
- Produces:
  - `src/utils/token.ts` 导出 `tokenStorage`：`getAccess()/setAccess(t)/getRefresh()/setRefresh(t)/clear()`
  - `src/api/client.ts` 导出 `client`（axios 实例）、`ApiResponse<T>`、泛型请求函数 `get<T>()/post<T>()/put<T>()/del<T>()`，统一返回解包后的 `data`
  - `src/api/auth.ts` 导出 `login(payload)/register(payload)/refresh(refreshToken)`
  - `src/api/user.ts` 导出 `getMe()/updateMe(payload)`
  - `src/api/category.ts` 导出 `listCategories()/createCategory(payload)`
  - `src/api/resource.ts` 导出 `listResources(params)/getResource(id)/createResource(data)/updateResource(id, data)/deleteResource(id)`
  - `src/api/rating.ts` 导出 `listRatings(resourceId, params)/upsertRating(resourceId, payload)`
  - `src/api/behavior.ts` 导出 `recordBehavior(resourceId, action)`
  - `src/api/admin.ts` 导出 `adminListUsers(params)/adminListResources(params)`

- [ ] **Step 1: 写失败测试**

`frontend/src/api/__tests__/client.test.ts`：

```ts
import { describe, it, expect } from 'vitest'
import { unwrap } from '../client'

describe('client 统一响应解包', () => {
  it('code === 0 时返回 data', () => {
    expect(unwrap({ code: 0, message: 'ok', data: { id: 1 } })).toEqual({ id: 1 })
  })

  it('code !== 0 时抛出业务错误', () => {
    expect(() => unwrap({ code: 10001, message: '参数错误', data: null })).toThrow('参数错误')
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd frontend
pnpm test src/api/__tests__/client.test.ts
```

Expected: FAIL，`Cannot find module '../client'`。

- [ ] **Step 3: 实现 token 存储与 axios 客户端**

`frontend/src/utils/token.ts`：

```ts
const ACCESS_KEY = 'edurec_access_token'
const REFRESH_KEY = 'edurec_refresh_token'

export const tokenStorage = {
  getAccess(): string {
    return localStorage.getItem(ACCESS_KEY) ?? ''
  },
  setAccess(token: string): void {
    localStorage.setItem(ACCESS_KEY, token)
  },
  getRefresh(): string {
    return localStorage.getItem(REFRESH_KEY) ?? ''
  },
  setRefresh(token: string): void {
    localStorage.setItem(REFRESH_KEY, token)
  },
  clear(): void {
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
}
```

`frontend/src/api/client.ts`：

```ts
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
        original._retried = true
        original.headers = { ...original.headers, Authorization: `Bearer ${tokenStorage.getAccess()}` }
        return client(original)
      }
      window.location.href = '/login'
    }
    const message =
      error.response?.data?.message ?? error.message ?? '网络错误，请稍后重试'
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
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd frontend
pnpm test src/api/__tests__/client.test.ts
```

Expected: PASS。

- [ ] **Step 5: 实现各模块 API 封装**

`frontend/src/api/auth.ts`：

```ts
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
```

`frontend/src/api/user.ts`：

```ts
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
```

`frontend/src/api/category.ts`：

```ts
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
```

`frontend/src/api/resource.ts`：

```ts
import { del, get, post, put } from './client'
import type { Page, Resource } from '@/types'

export interface ResourceQuery {
  page?: number
  page_size?: number
  keyword?: string
  category_id?: number
  type?: string
  sort?: 'latest' | 'popular' | 'rating'
  tags?: string
}

export function listResources(params: ResourceQuery = {}) {
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
```

`frontend/src/api/rating.ts`：

```ts
import { get, post } from './client'
import type { Page, Rating } from '@/types'

export interface RatingQuery {
  page?: number
  page_size?: number
}

export interface UpsertRatingPayload {
  score: number
  comment?: string
}

export function listRatings(resourceId: number, params: RatingQuery = {}) {
  return get<Page<Rating>>(`/resources/${resourceId}/ratings`, { params })
}

export function upsertRating(resourceId: number, payload: UpsertRatingPayload) {
  return post<Rating>(`/resources/${resourceId}/ratings`, payload)
}
```

`frontend/src/api/behavior.ts`：

```ts
import { get, post } from './client'
import type { Behavior, BehaviorAction, Page } from '@/types'

export interface BehaviorQuery {
  action?: BehaviorAction
  page?: number
  page_size?: number
}

export function recordBehavior(resourceId: number, action: BehaviorAction) {
  return post<null>(`/resources/${resourceId}/behaviors`, { action })
}

export function listMyBehaviors(params: BehaviorQuery = {}) {
  return get<Page<Behavior>>('/users/me/behaviors', { params })
}
```

`frontend/src/api/admin.ts`：

```ts
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
```

- [ ] **Step 6: 验证类型检查与测试**

```bash
cd frontend
pnpm type-check
pnpm test
```

Expected: 均无错误。

- [ ] **Step 7: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): 实现 API 客户端与接口封装"
```

---

### Task 3: MSW mock 数据

**Files:**
- Create: `frontend/src/mocks/db.ts`
- Create: `frontend/src/mocks/handlers.ts`
- Create: `frontend/src/mocks/browser.ts`
- Create: `frontend/public/mockServiceWorker.js`（由 `msw init` 生成）
- Test: `frontend/src/mocks/__tests__/handlers.test.ts`

**Interfaces:**
- Consumes: `src/api/client.ts` 的 `client`（Task 2）、`src/types/index.ts`（Task 1）
- Produces:
  - `src/mocks/db.ts` 导出 `db` 与 `DbUser`/`DbResource`/`DbCategory`/`DbRating`/`DbBehavior` 类型
  - `src/mocks/handlers.ts` 导出 `handlers`（MSW handler 数组）
  - `src/mocks/browser.ts` 导出 `worker`

- [ ] **Step 1: 初始化 MSW service worker**

```bash
cd frontend
npx msw init public/ --save
```

Expected: 生成 `frontend/public/mockServiceWorker.js`，并在 package.json 的 scripts 中加入 `"msw": "msw init public/ --save"`。

- [ ] **Step 2: 写失败测试**

`frontend/src/mocks/__tests__/handlers.test.ts`：

```ts
import { beforeAll, afterAll, afterEach, describe, it, expect } from 'vitest'
import { setupServer } from 'msw/node'
import { handlers } from '../handlers'
import { client } from '@/api/client'
import { listResources } from '@/api/resource'
import { login } from '@/api/auth'
import { listCategories } from '@/api/category'
import { getMe } from '@/api/user'

const server = setupServer(...handlers)

beforeAll(() => {
  server.listen()
  client.defaults.baseURL = 'http://localhost/api/v1'
})

afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('MSW handlers', () => {
  it('登录返回统一响应结构', async () => {
    const result = await login({ username: 'admin', password: 'admin123' })
    expect(result.access_token).toBeTruthy()
    expect(result.user.username).toBe('admin')
  })

  it('登录失败返回业务错误', async () => {
    await expect(login({ username: 'admin', password: 'wrong' })).rejects.toThrow('用户名或密码错误')
  })

  it('资源列表返回种子数据', async () => {
    const data = await listResources({ page: 1, page_size: 20 })
    expect(data.list.length).toBeGreaterThan(0)
    expect(data.list[0]).toHaveProperty('title')
  })

  it('未认证访问受保护接口返回 401', async () => {
    await expect(getMe()).rejects.toThrow('未认证')
  })

  it('分类列表返回数据', async () => {
    const categories = await listCategories()
    expect(categories.length).toBeGreaterThan(0)
  })
})
```

- [ ] **Step 3: 运行测试确认失败**

```bash
cd frontend
pnpm test src/mocks/__tests__/handlers.test.ts
```

Expected: FAIL，`Cannot find module '../handlers'`。

- [ ] **Step 4: 实现 mock 数据库**

`frontend/src/mocks/db.ts`：

```ts
export interface DbUser {
  id: number
  username: string
  email: string
  password: string
  display_name: string | null
  avatar_url: string | null
  is_admin: boolean
  created_at: string
}

export interface DbCategory {
  id: number
  name: string
  description: string
}

export type DbResourceType = 'course' | 'article' | 'video'

export interface DbResource {
  id: number
  title: string
  description: string
  cover_url: string | null
  type: DbResourceType
  category_id: number
  tags: string[]
  metadata: Record<string, unknown>
  author: string | null
  source_url: string | null
  avg_rating: number
  view_count: number
  created_at: string
  updated_at: string
}

export interface DbRating {
  id: number
  user_id: number
  resource_id: number
  score: number
  comment: string | null
  created_at: string
}

export type DbBehaviorAction = 'view' | 'click' | 'favorite'

export interface DbBehavior {
  id: number
  user_id: number
  resource_id: number
  action: DbBehaviorAction
  created_at: string
}

const now = () => new Date().toISOString()

export const db = {
  users: [
    {
      id: 1,
      username: 'admin',
      email: 'admin@edurec.dev',
      password: 'admin123',
      display_name: '平台管理员',
      avatar_url: null,
      is_admin: true,
      created_at: '2026-07-01T08:00:00Z',
    },
    {
      id: 2,
      username: 'user',
      email: 'user@edurec.dev',
      password: 'user123',
      display_name: '张三',
      avatar_url: null,
      is_admin: false,
      created_at: '2026-07-05T08:00:00Z',
    },
  ] as DbUser[],

  categories: [
    { id: 1, name: '人工智能', description: 'AI、机器学习、深度学习' },
    { id: 2, name: '前端开发', description: 'HTML、CSS、JavaScript' },
    { id: 3, name: '后端开发', description: 'Go、Python、Java' },
  ] as DbCategory[],

  resources: [
    {
      id: 1,
      title: '机器学习入门',
      description: '面向零基础学习者的机器学习课程，从线性回归到神经网络。',
      cover_url: null,
      type: 'course',
      category_id: 1,
      tags: ['AI', '入门', 'Python'],
      metadata: { duration: '12 小时', chapters: 24, language: '中文' },
      author: '吴恩达',
      source_url: 'https://example.com/course/ml-intro',
      avg_rating: 4.5,
      view_count: 1024,
      created_at: '2026-07-01T08:00:00Z',
      updated_at: '2026-07-15T10:00:00Z',
    },
    {
      id: 2,
      title: 'Vue 3 实战',
      description: '从零构建 Vue 3 + TypeScript 应用，覆盖组合式 API 与工程化。',
      cover_url: null,
      type: 'course',
      category_id: 2,
      tags: ['Vue', 'TypeScript', '前端'],
      metadata: { duration: '8 小时', chapters: 18, language: '中文' },
      author: '李老师',
      source_url: 'https://example.com/course/vue3',
      avg_rating: 4.8,
      view_count: 2048,
      created_at: '2026-06-15T08:00:00Z',
      updated_at: '2026-07-10T09:00:00Z',
    },
    {
      id: 3,
      title: 'Go 语言高并发编程',
      description: '深入理解 goroutine 与 channel，编写高并发服务。',
      cover_url: null,
      type: 'article',
      category_id: 3,
      tags: ['Go', '并发'],
      metadata: { word_count: 5000, reading_minutes: 20 },
      author: '王工',
      source_url: 'https://example.com/article/go-concurrency',
      avg_rating: 4.2,
      view_count: 860,
      created_at: '2026-05-20T08:00:00Z',
      updated_at: '2026-07-01T10:00:00Z',
    },
    {
      id: 4,
      title: 'Python 数据分析',
      description: '使用 pandas 与 matplotlib 完成数据清洗与可视化。',
      cover_url: null,
      type: 'video',
      category_id: 1,
      tags: ['Python', '数据分析'],
      metadata: { duration: '45 分钟', resolution: '1080p' },
      author: '陈老师',
      source_url: 'https://example.com/video/python-data',
      avg_rating: 4.7,
      view_count: 1532,
      created_at: '2026-06-01T08:00:00Z',
      updated_at: '2026-07-05T10:00:00Z',
    },
    {
      id: 5,
      title: 'TypeScript 类型体操',
      description: '进阶类型技巧：泛型、条件类型、模板字面量类型。',
      cover_url: null,
      type: 'article',
      category_id: 2,
      tags: ['TypeScript', '进阶'],
      metadata: { word_count: 3200, reading_minutes: 12 },
      author: '赵博',
      source_url: 'https://example.com/article/ts-types',
      avg_rating: 4.6,
      view_count: 640,
      created_at: '2026-07-10T08:00:00Z',
      updated_at: '2026-07-12T08:00:00Z',
    },
  ] as DbResource[],

  ratings: [
    { id: 1, user_id: 2, resource_id: 1, score: 5, comment: '讲解非常清晰，入门首选', created_at: '2026-07-20T10:00:00Z' },
    { id: 2, user_id: 2, resource_id: 2, score: 4, comment: '实战性强', created_at: '2026-07-22T10:00:00Z' },
    { id: 3, user_id: 2, resource_id: 3, score: 3, comment: '并发部分可以更深入', created_at: '2026-07-24T10:00:00Z' },
  ] as DbRating[],

  behaviors: [
    { id: 1, user_id: 2, resource_id: 1, action: 'view', created_at: now() },
    { id: 2, user_id: 2, resource_id: 2, action: 'click', created_at: now() },
    { id: 3, user_id: 2, resource_id: 4, action: 'favorite', created_at: now() },
  ] as DbBehavior[],
}
```

- [ ] **Step 5: 实现 mock handlers**

`frontend/src/mocks/handlers.ts`：

```ts
import { http, HttpResponse } from 'msw'
import { db, DbResource, DbUser } from './db'

const BASE = '/api/v1'

export function ok(data: unknown) {
  return HttpResponse.json({ code: 0, message: 'ok', data })
}

export function bizError(code: number, message: string, status = 400) {
  return HttpResponse.json({ code, message, data: null }, { status })
}

function currentUser(request: Request): DbUser | null {
  const auth = request.headers.get('Authorization') ?? ''
  const token = auth.replace(/^Bearer /, '')
  if (!token) return null
  const id = Number(token.replace('mock-', ''))
  return db.users.find((u) => u.id === id) ?? null
}

function requireUser(request: Request): DbUser | null {
  const user = currentUser(request)
  if (!user) {
    return null
  }
  return user
}

function toPublicUser(u: DbUser) {
  return {
    id: u.id,
    username: u.username,
    email: u.email,
    display_name: u.display_name,
    avatar_url: u.avatar_url,
    is_admin: u.is_admin,
    created_at: u.created_at,
  }
}

function toPublicResource(r: DbResource) {
  const cat = db.categories.find((c) => c.id === r.category_id)
  return {
    id: r.id,
    title: r.title,
    description: r.description,
    cover_url: r.cover_url,
    type: r.type,
    category: cat ? { id: cat.id, name: cat.name } : null,
    tags: r.tags,
    metadata: r.metadata,
    author: r.author,
    source_url: r.source_url,
    avg_rating: r.avg_rating,
    view_count: r.view_count,
    created_at: r.created_at,
    updated_at: r.updated_at,
  }
}

export const handlers = [
  // 认证
  http.post(`${BASE}/auth/register`, async ({ request }) => {
    const body = (await request.json()) as {
      username: string
      email: string
      password: string
      display_name?: string
    }
    if (db.users.some((u) => u.username === body.username)) {
      return bizError(10005, '用户名已存在', 409)
    }
    const user: DbUser = {
      id: db.users.length + 1,
      username: body.username,
      email: body.email,
      password: body.password,
      display_name: body.display_name ?? null,
      avatar_url: null,
      is_admin: false,
      created_at: new Date().toISOString(),
    }
    db.users.push(user)
    return ok({ user: toPublicUser(user) })
  }),

  http.post(`${BASE}/auth/login`, async ({ request }) => {
    const body = (await request.json()) as { username: string; password: string }
    const user = db.users.find(
      (u) => (u.username === body.username || u.email === body.username) && u.password === body.password,
    )
    if (!user) {
      return bizError(10002, '用户名或密码错误', 401)
    }
    return ok({
      access_token: `mock-${user.id}`,
      refresh_token: `mock-rf-${user.id}`,
      expires_in: 900,
      user: toPublicUser(user),
    })
  }),

  http.post(`${BASE}/auth/refresh`, async ({ request }) => {
    const body = (await request.json()) as { refresh_token: string }
    const id = Number((body.refresh_token ?? '').replace('mock-rf-', ''))
    const user = db.users.find((u) => u.id === id)
    if (!user) {
      return bizError(10002, '刷新失败', 401)
    }
    return ok({ access_token: `mock-${user.id}`, refresh_token: `mock-rf-${user.id}`, expires_in: 900 })
  }),

  // 用户
  http.get(`${BASE}/users/me`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    return ok(toPublicUser(user))
  }),

  http.put(`${BASE}/users/me`, async ({ request }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    const body = (await request.json()) as { display_name?: string; avatar_url?: string }
    user.display_name = body.display_name ?? user.display_name
    user.avatar_url = body.avatar_url ?? user.avatar_url
    return ok({ id: user.id, display_name: user.display_name, avatar_url: user.avatar_url })
  }),

  // 分类
  http.get(`${BASE}/categories`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    return ok(db.categories)
  }),

  http.post(`${BASE}/categories`, async ({ request }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    if (!user.is_admin) return bizError(10003, '无权限', 403)
    const body = (await request.json()) as { name: string; description?: string }
    if (db.categories.some((c) => c.name === body.name)) {
      return bizError(10005, '分类已存在', 409)
    }
    const category = { id: db.categories.length + 1, name: body.name, description: body.description ?? '' }
    db.categories.push(category)
    return ok(category)
  }),

  // 资源
  http.get(`${BASE}/resources`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? '1')
    const pageSize = Math.min(Number(url.searchParams.get('page_size') ?? '20'), 100)
    const keyword = url.searchParams.get('keyword') ?? ''
    const categoryId = url.searchParams.get('category_id')
    const type = url.searchParams.get('type')
    const sort = url.searchParams.get('sort') ?? 'latest'

    let list = db.resources.filter((r) => {
      if (keyword && !(r.title.includes(keyword) || r.description.includes(keyword))) return false
      if (categoryId && String(r.category_id) !== categoryId) return false
      if (type && r.type !== type) return false
      return true
    })
    if (sort === 'popular') list = [...list].sort((a, b) => b.view_count - a.view_count)
    else if (sort === 'rating') list = [...list].sort((a, b) => b.avg_rating - a.avg_rating)
    else list = [...list].sort((a, b) => b.created_at.localeCompare(a.created_at))

    const total = list.length
    const paged = list.slice((page - 1) * pageSize, page * pageSize)
    return ok({ list: paged.map(toPublicResource), total, page, page_size: pageSize })
  }),

  http.get(`${BASE}/resources/:id`, ({ request, params }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    const resource = db.resources.find((r) => r.id === Number(params.id))
    if (!resource) return bizError(10004, '资源不存在', 404)
    resource.view_count += 1
    return ok(toPublicResource(resource))
  }),

  http.post(`${BASE}/resources`, async ({ request }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    if (!user.is_admin) return bizError(10003, '无权限', 403)
    const body = (await request.json()) as Partial<DbResource> & { title: string; type: DbResource['type']; category_id: number }
    const resource: DbResource = {
      id: db.resources.length + 1,
      title: body.title,
      description: body.description ?? '',
      cover_url: body.cover_url ?? null,
      type: body.type,
      category_id: body.category_id,
      tags: body.tags ?? [],
      metadata: body.metadata ?? {},
      author: body.author ?? null,
      source_url: body.source_url ?? null,
      avg_rating: 0,
      view_count: 0,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    db.resources.push(resource)
    return ok(toPublicResource(resource))
  }),

  http.put(`${BASE}/resources/:id`, async ({ request, params }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    if (!user.is_admin) return bizError(10003, '无权限', 403)
    const resource = db.resources.find((r) => r.id === Number(params.id))
    if (!resource) return bizError(10004, '资源不存在', 404)
    const body = (await request.json()) as Partial<DbResource>
    Object.assign(resource, body, { id: resource.id, updated_at: new Date().toISOString() })
    return ok({ id: resource.id, updated_at: resource.updated_at })
  }),

  http.delete(`${BASE}/resources/:id`, ({ request, params }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    if (!user.is_admin) return bizError(10003, '无权限', 403)
    const index = db.resources.findIndex((r) => r.id === Number(params.id))
    if (index === -1) return bizError(10004, '资源不存在', 404)
    db.resources.splice(index, 1)
    return ok(null)
  }),

  // 评分
  http.get(`${BASE}/resources/:id/ratings`, ({ request, params }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? '1')
    const pageSize = Number(url.searchParams.get('page_size') ?? '20')
    const list = db.ratings.filter((r) => r.resource_id === Number(params.id))
    const total = list.length
    const paged = list.slice((page - 1) * pageSize, page * pageSize)
    return ok({
      list: paged.map((r) => {
        const u = db.users.find((x) => x.id === r.user_id)
        return {
          id: r.id,
          user: u
            ? { id: u.id, username: u.username, display_name: u.display_name, avatar_url: u.avatar_url }
            : null,
          score: r.score,
          comment: r.comment,
          created_at: r.created_at,
        }
      }),
      total,
      page,
      page_size: pageSize,
    })
  }),

  http.post(`${BASE}/resources/:id/ratings`, async ({ request, params }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    const body = (await request.json()) as { score: number; comment?: string }
    const resourceId = Number(params.id)
    const existing = db.ratings.find((r) => r.user_id === user.id && r.resource_id === resourceId)
    if (existing) {
      existing.score = body.score
      existing.comment = body.comment ?? existing.comment
      return ok({ id: existing.id, score: existing.score, comment: existing.comment, created_at: existing.created_at })
    }
    const rating = {
      id: db.ratings.length + 1,
      user_id: user.id,
      resource_id: resourceId,
      score: body.score,
      comment: body.comment ?? null,
      created_at: new Date().toISOString(),
    }
    db.ratings.push(rating)
    return ok({ id: rating.id, score: rating.score, comment: rating.comment, created_at: rating.created_at })
  }),

  // 推荐
  http.get(`${BASE}/recommendations`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    const url = new URL(request.url)
    const limit = Math.min(Number(url.searchParams.get('limit') ?? '20'), 50)
    const list = [...db.resources].sort((a, b) => b.avg_rating - a.avg_rating).slice(0, limit)
    return ok({ list: list.map(toPublicResource), updated_at: new Date().toISOString() })
  }),

  // 行为
  http.post(`${BASE}/resources/:id/behaviors`, async ({ request, params }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    const body = (await request.json()) as { action: DbBehavior['action'] }
    db.behaviors.push({
      id: db.behaviors.length + 1,
      user_id: user.id,
      resource_id: Number(params.id),
      action: body.action,
      created_at: new Date().toISOString(),
    })
    return ok(null)
  }),

  http.get(`${BASE}/users/me/behaviors`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? '1')
    const pageSize = Number(url.searchParams.get('page_size') ?? '20')
    const action = url.searchParams.get('action')
    let list = db.behaviors.filter((b) => b.user_id === user.id)
    if (action) list = list.filter((b) => b.action === action)
    list = [...list].sort((a, b) => b.created_at.localeCompare(a.created_at))
    const total = list.length
    const paged = list.slice((page - 1) * pageSize, page * pageSize)
    return ok({
      list: paged.map((b) => {
        const r = db.resources.find((x) => x.id === b.resource_id)
        return {
          id: b.id,
          resource: r
            ? { id: r.id, title: r.title, cover_url: r.cover_url, type: r.type }
            : null,
          action: b.action,
          created_at: b.created_at,
        }
      }),
      total,
      page,
      page_size: pageSize,
    })
  }),

  // 管理
  http.get(`${BASE}/admin/users`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    if (!user.is_admin) return bizError(10003, '无权限', 403)
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? '1')
    const pageSize = Number(url.searchParams.get('page_size') ?? '20')
    const keyword = url.searchParams.get('keyword') ?? ''
    let list = db.users.filter(
      (u) => !keyword || u.username.includes(keyword) || u.email.includes(keyword),
    )
    const total = list.length
    const paged = list.slice((page - 1) * pageSize, page * pageSize)
    return ok({ list: paged.map(toPublicUser), total, page, page_size: pageSize })
  }),

  http.get(`${BASE}/admin/resources`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    if (!user.is_admin) return bizError(10003, '无权限', 403)
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? '1')
    const pageSize = Number(url.searchParams.get('page_size') ?? '20')
    const keyword = url.searchParams.get('keyword') ?? ''
    let list = db.resources.filter((r) => !keyword || r.title.includes(keyword))
    const total = list.length
    const paged = list.slice((page - 1) * pageSize, page * pageSize)
    return ok({ list: paged.map(toPublicResource), total, page, page_size: pageSize })
  }),
]
```

- [ ] **Step 6: 实现 browser worker**

`frontend/src/mocks/browser.ts`：

```ts
import { setupWorker } from 'msw/browser'
import { handlers } from './handlers'

export const worker = setupWorker(...handlers)
```

- [ ] **Step 7: 运行测试确认通过**

```bash
cd frontend
pnpm test src/mocks/__tests__/handlers.test.ts
```

Expected: PASS。

- [ ] **Step 8: 类型检查**

```bash
cd frontend
pnpm type-check
```

Expected: 无错误。

- [ ] **Step 9: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): 实现 MSW mock 数据"
```

---

### Task 4: 认证 store 与测试

**Files:**
- Create: `frontend/src/stores/auth.ts`
- Test: `frontend/src/stores/__tests__/auth.test.ts`

**Interfaces:**
- Consumes: `src/api/auth.ts`、`src/api/user.ts`（Task 2）、`src/utils/token.ts`（Task 2）、`src/types/index.ts`（Task 1）
- Produces: `useAuthStore()`，返回 `{ accessToken, refreshToken, user, loading, isLoggedIn, isAdmin, login, register, fetchMe, logout }`

- [ ] **Step 1: 写失败测试**

`frontend/src/stores/__tests__/auth.test.ts`：

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '../auth'
import * as authApi from '@/api/auth'
import * as userApi from '@/api/user'
import type { User } from '@/types'

vi.mock('@/api/auth', () => ({ login: vi.fn(), register: vi.fn() }))
vi.mock('@/api/user', () => ({ getMe: vi.fn() }))

const mockedAuthApi = vi.mocked(authApi)
const mockedUserApi = vi.mocked(userApi)

const baseUser: User = {
  id: 2,
  username: 'user',
  email: 'user@edurec.dev',
  display_name: '张三',
  avatar_url: null,
  is_admin: false,
  created_at: '2026-07-05T08:00:00Z',
}

const loginResult = {
  access_token: 'at',
  refresh_token: 'rt',
  expires_in: 900,
  user: baseUser,
}

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('login 保存 token 与用户信息', async () => {
    mockedAuthApi.login.mockResolvedValue(loginResult)
    const store = useAuthStore()
    await store.login('user', 'user123')
    expect(store.isLoggedIn).toBe(true)
    expect(store.user?.username).toBe('user')
    expect(localStorage.getItem('edurec_access_token')).toBe('at')
    expect(localStorage.getItem('edurec_refresh_token')).toBe('rt')
  })

  it('logout 清除 token 与用户信息', async () => {
    mockedAuthApi.login.mockResolvedValue(loginResult)
    const store = useAuthStore()
    await store.login('user', 'user123')
    store.logout()
    expect(store.isLoggedIn).toBe(false)
    expect(store.user).toBeNull()
    expect(localStorage.getItem('edurec_access_token')).toBeNull()
  })

  it('isAdmin 反映用户管理员身份', () => {
    const store = useAuthStore()
    store.user = { ...baseUser, is_admin: true }
    expect(store.isAdmin).toBe(true)
    store.user = baseUser
    expect(store.isAdmin).toBe(false)
  })

  it('fetchMe 拉取用户信息', async () => {
    mockedUserApi.getMe.mockResolvedValue(baseUser)
    const store = useAuthStore()
    await store.fetchMe()
    expect(store.user).toEqual(baseUser)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd frontend
pnpm test src/stores/__tests__/auth.test.ts
```

Expected: FAIL，`Cannot find module '../auth'`。

- [ ] **Step 3: 实现 auth store**

`frontend/src/stores/auth.ts`：

```ts
import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import * as authApi from '@/api/auth'
import * as userApi from '@/api/user'
import { tokenStorage } from '@/utils/token'
import type { RegisterPayload, User } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref(tokenStorage.getAccess())
  const refreshToken = ref(tokenStorage.getRefresh())
  const user = ref<User | null>(null)
  const loading = ref(false)

  const isLoggedIn = computed(() => accessToken.value !== '')
  const isAdmin = computed(() => user.value?.is_admin === true)

  async function login(username: string, password: string): Promise<User> {
    loading.value = true
    try {
      const result = await authApi.login({ username, password })
      accessToken.value = result.access_token
      refreshToken.value = result.refresh_token
      tokenStorage.setAccess(result.access_token)
      tokenStorage.setRefresh(result.refresh_token)
      user.value = result.user
      return result.user
    } finally {
      loading.value = false
    }
  }

  async function register(payload: RegisterPayload): Promise<User> {
    const result = await authApi.register(payload)
    return result.user
  }

  async function fetchMe(): Promise<User> {
    user.value = await userApi.getMe()
    return user.value
  }

  function logout(): void {
    accessToken.value = ''
    refreshToken.value = ''
    user.value = null
    tokenStorage.clear()
  }

  return {
    accessToken,
    refreshToken,
    user,
    loading,
    isLoggedIn,
    isAdmin,
    login,
    register,
    fetchMe,
    logout,
  }
})
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd frontend
pnpm test src/stores/__tests__/auth.test.ts
```

Expected: PASS。

- [ ] **Step 5: 类型检查**

```bash
cd frontend
pnpm type-check
```

Expected: 无错误。

- [ ] **Step 6: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): 实现认证 store 与测试"
```

---

### Task 5: 布局骨架、路由与守卫

**Files:**
- Create: `frontend/src/components/layout/FrontLayout.vue`
- Create: `frontend/src/components/layout/AdminLayout.vue`
- Create: `frontend/src/pages/home/index.vue`
- Create: `frontend/src/pages/auth/LoginPage.vue`
- Create: `frontend/src/pages/auth/RegisterPage.vue`
- Create: `frontend/src/pages/placeholder/PlaceholderPage.vue`
- Create: `frontend/src/pages/not-found/NotFound.vue`
- Create: `frontend/src/router/guards.ts`
- Create: `frontend/src/router/index.ts`
- Test: `frontend/src/router/__tests__/guards.test.ts`

**Interfaces:**
- Consumes: `useAuthStore`（Task 4）、`@/pages/*`、`@/components/layout/*`
- Produces:
  - `src/router/guards.ts` 导出 `resolveGuard(to, auth)` 与 `AuthLike`
  - `src/router/index.ts` 默认导出 `router`
  - `FrontLayout`/`AdminLayout` 默认导出 Vue 组件

- [ ] **Step 1: 写失败测试**

`frontend/src/router/__tests__/guards.test.ts`：

```ts
import { describe, expect, it, vi } from 'vitest'
import { resolveGuard } from '../guards'
import type { AuthLike } from '../guards'

function makeTo(overrides: Record<string, unknown> = {}) {
  return {
    fullPath: '/resources/1',
    name: 'resource-detail',
    meta: { requiresAuth: true },
    ...overrides,
  } as unknown as Parameters<typeof resolveGuard>[0]
}

function makeAuth(overrides: Partial<AuthLike> = {}): AuthLike {
  return {
    isLoggedIn: false,
    isAdmin: false,
    user: null,
    fetchMe: vi.fn(),
    logout: vi.fn(),
    ...overrides,
  }
}

describe('路由守卫', () => {
  it('未登录访问受保护页面重定向到登录页', async () => {
    const result = await resolveGuard(makeTo(), makeAuth())
    expect(result).toEqual({ name: 'login', query: { redirect: '/resources/1' } })
  })

  it('已登录但未加载用户时先拉取用户信息', async () => {
    const auth = makeAuth({ isLoggedIn: true })
    const result = await resolveGuard(makeTo(), auth)
    expect(auth.fetchMe).toHaveBeenCalled()
    expect(result).toBe(true)
  })

  it('拉取用户信息失败则登出并重定向登录', async () => {
    const auth = makeAuth({
      isLoggedIn: true,
      fetchMe: vi.fn().mockRejectedValue(new Error('fail')),
    })
    const result = await resolveGuard(makeTo(), auth)
    expect(auth.logout).toHaveBeenCalled()
    expect(result).toEqual({ name: 'login' })
  })

  it('非管理员访问管理页面重定向首页', async () => {
    const result = await resolveGuard(
      makeTo({ meta: { requiresAuth: true, requiresAdmin: true } }),
      makeAuth({ isLoggedIn: true, isAdmin: false, user: {} }),
    )
    expect(result).toEqual({ name: 'home' })
  })

  it('已登录访问登录页重定向首页', async () => {
    const result = await resolveGuard(
      makeTo({ meta: { guestOnly: true } }),
      makeAuth({ isLoggedIn: true, user: {} }),
    )
    expect(result).toEqual({ name: 'home' })
  })

  it('无约束路由放行', async () => {
    const result = await resolveGuard(makeTo({ meta: {} }), makeAuth())
    expect(result).toBe(true)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd frontend
pnpm test src/router/__tests__/guards.test.ts
```

Expected: FAIL，`Cannot find module '../guards'`。

- [ ] **Step 3: 实现守卫**

`frontend/src/router/guards.ts`：

```ts
import type { RouteLocationNormalized } from 'vue-router'

export interface AuthLike {
  isLoggedIn: boolean
  isAdmin: boolean
  user: unknown
  fetchMe: () => Promise<unknown>
  logout: () => void
}

export async function resolveGuard(
  to: RouteLocationNormalized,
  auth: AuthLike,
): Promise<true | { name: string; query?: { redirect?: string } }> {
  if (to.meta.requiresAuth) {
    if (!auth.isLoggedIn) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }
    if (!auth.user) {
      try {
        await auth.fetchMe()
      } catch {
        auth.logout()
        return { name: 'login' }
      }
      if (!auth.isLoggedIn) {
        return { name: 'login' }
      }
    }
    if (to.meta.requiresAdmin && !auth.isAdmin) {
      return { name: 'home' }
    }
  }
  if (to.meta.guestOnly && auth.isLoggedIn) {
    return { name: 'home' }
  }
  return true
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd frontend
pnpm test src/router/__tests__/guards.test.ts
```

Expected: PASS。

- [ ] **Step 5: 实现布局组件**

`frontend/src/components/layout/FrontLayout.vue`：

```vue
<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()

function handleCommand(command: string) {
  if (command === 'logout') {
    auth.logout()
    router.push({ name: 'login' })
  } else if (command === 'me') {
    router.push({ name: 'user-me' })
  }
}
</script>

<template>
  <div class="flex min-h-screen flex-col bg-bg">
    <header class="sticky top-0 z-40 border-b border-border bg-surface/95 backdrop-blur">
      <div class="mx-auto flex h-16 max-w-7xl items-center gap-6 px-6">
        <RouterLink to="/" class="flex items-center gap-2 text-lg font-bold text-primary">
          <span class="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-white">E</span>
          edurec
        </RouterLink>
        <el-input
          placeholder="搜索课程、文章、视频…"
          class="max-w-sm"
          clearable
          @keyup.enter="router.push({ name: 'search' })"
        />
        <nav class="ml-4 hidden items-center gap-5 text-sm text-ink-secondary md:flex">
          <RouterLink to="/" class="hover:text-primary">首页</RouterLink>
          <RouterLink :to="{ name: 'search' }" class="hover:text-primary">课程</RouterLink>
          <RouterLink :to="{ name: 'search' }" class="hover:text-primary">文章</RouterLink>
        </nav>
        <div class="ml-auto flex items-center gap-3">
          <template v-if="auth.isLoggedIn">
            <el-dropdown @command="handleCommand">
              <span class="flex cursor-pointer items-center gap-2 text-sm">
                <el-avatar :size="28" :src="auth.user?.avatar_url ?? undefined">
                  {{ (auth.user?.display_name ?? auth.user?.username ?? 'U').charAt(0) }}
                </el-avatar>
                {{ auth.user?.display_name ?? auth.user?.username }}
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="me">个人中心</el-dropdown-item>
                  <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
          <template v-else>
            <RouterLink :to="{ name: 'login' }">
              <el-button type="primary" round>登录 / 注册</el-button>
            </RouterLink>
          </template>
        </div>
      </div>
    </header>

    <main class="flex-1">
      <RouterView />
    </main>

    <footer class="border-t border-border py-6 text-center text-xs text-ink-muted">
      © 2026 edurec · 教育资源推荐平台
    </footer>
  </div>
</template>
```

`frontend/src/components/layout/AdminLayout.vue`：

```vue
<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

function logout() {
  auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="flex min-h-screen bg-bg">
    <aside class="flex w-56 shrink-0 flex-col border-r border-border bg-surface">
      <div class="flex h-16 items-center gap-2 border-b border-border px-5">
        <span class="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-white">E</span>
        <span class="font-bold">管理后台</span>
      </div>
      <el-menu
        :default-active="route.path"
        class="flex-1 border-0"
        router
      >
        <el-menu-item index="/admin">仪表盘</el-menu-item>
        <el-menu-item index="/admin/resources">资源管理</el-menu-item>
        <el-menu-item index="/admin/users">用户管理</el-menu-item>
        <el-menu-item index="/admin/categories">分类管理</el-menu-item>
      </el-menu>
      <div class="border-t border-border p-4 text-xs text-ink-muted">
        当前管理员：{{ auth.user?.display_name ?? auth.user?.username }}
      </div>
    </aside>

    <div class="flex min-w-0 flex-1 flex-col">
      <header class="flex h-16 items-center justify-between border-b border-border bg-surface px-6">
        <div class="text-sm text-ink-secondary">{{ route.name }}</div>
        <div class="flex items-center gap-3">
          <RouterLink :to="{ name: 'home' }">
            <el-button size="small" text>返回前台</el-button>
          </RouterLink>
          <el-button size="small" @click="logout">退出</el-button>
        </div>
      </header>
      <main class="flex-1 overflow-auto p-6">
        <RouterView />
      </main>
    </div>
  </div>
</template>
```

- [ ] **Step 6: 实现页面组件**

`frontend/src/pages/home/index.vue`（本阶段占位，Task 6 接入数据）：

```vue
<template>
  <div class="mx-auto max-w-7xl px-6 py-10">
    <h1 class="text-2xl font-bold">发现优质教育资源</h1>
    <p class="mt-2 text-sm text-ink-secondary">推荐课程、文章与视频（Task 6 接入 mock 数据）</p>
  </div>
</template>
```

`frontend/src/pages/auth/LoginPage.vue`（占位，子项目 2 实现）：

```vue
<template>
  <div class="mx-auto max-w-md px-6 py-16 text-center">
    <h1 class="text-2xl font-bold">登录</h1>
    <p class="mt-2 text-sm text-ink-secondary">登录页面将在后续子项目实现</p>
  </div>
</template>
```

`frontend/src/pages/auth/RegisterPage.vue`（占位，子项目 2 实现）：

```vue
<template>
  <div class="mx-auto max-w-md px-6 py-16 text-center">
    <h1 class="text-2xl font-bold">注册</h1>
    <p class="mt-2 text-sm text-ink-secondary">注册页面将在后续子项目实现</p>
  </div>
</template>
```

`frontend/src/pages/placeholder/PlaceholderPage.vue`：

```vue
<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<template>
  <div class="flex items-center justify-center py-24">
    <div class="text-center">
      <h2 class="text-xl font-semibold">功能开发中</h2>
      <p class="mt-2 text-sm text-ink-muted">路由：{{ route.name }}（后续子项目实现）</p>
    </div>
  </div>
</template>
```

`frontend/src/pages/not-found/NotFound.vue`：

```vue
<template>
  <div class="flex items-center justify-center py-24">
    <div class="text-center">
      <h1 class="text-4xl font-bold text-primary">404</h1>
      <p class="mt-2 text-sm text-ink-secondary">页面不存在</p>
      <RouterLink :to="{ name: 'home' }">
        <el-button type="primary" round class="mt-6">返回首页</el-button>
      </RouterLink>
    </div>
  </div>
</template>
```

- [ ] **Step 7: 实现路由**

`frontend/src/router/index.ts`：

```ts
import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { resolveGuard } from './guards'
import FrontLayout from '@/components/layout/FrontLayout.vue'
import AdminLayout from '@/components/layout/AdminLayout.vue'
import HomePage from '@/pages/home/index.vue'
import LoginPage from '@/pages/auth/LoginPage.vue'
import RegisterPage from '@/pages/auth/RegisterPage.vue'
import PlaceholderPage from '@/pages/placeholder/PlaceholderPage.vue'
import NotFoundPage from '@/pages/not-found/NotFound.vue'

const routes = [
  {
    path: '/',
    component: FrontLayout,
    children: [
      { path: '', name: 'home', component: HomePage, meta: { requiresAuth: true } },
      { path: 'login', name: 'login', component: LoginPage, meta: { guestOnly: true } },
      { path: 'register', name: 'register', component: RegisterPage, meta: { guestOnly: true } },
      { path: 'search', name: 'search', component: PlaceholderPage, meta: { requiresAuth: true } },
      { path: 'resources/:id', name: 'resource-detail', component: PlaceholderPage, meta: { requiresAuth: true } },
      { path: 'user/me', name: 'user-me', component: PlaceholderPage, meta: { requiresAuth: true } },
      {
        path: 'resources/upload',
        name: 'resource-upload',
        component: PlaceholderPage,
        meta: { requiresAuth: true, requiresAdmin: true },
      },
    ],
  },
  {
    path: '/admin',
    component: AdminLayout,
    children: [
      { path: '', name: 'admin-dashboard', component: PlaceholderPage, meta: { requiresAuth: true, requiresAdmin: true } },
      { path: 'users', name: 'admin-users', component: PlaceholderPage, meta: { requiresAuth: true, requiresAdmin: true } },
      { path: 'resources', name: 'admin-resources', component: PlaceholderPage, meta: { requiresAuth: true, requiresAdmin: true } },
      { path: 'categories', name: 'admin-categories', component: PlaceholderPage, meta: { requiresAuth: true, requiresAdmin: true } },
    ],
  },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundPage },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  return resolveGuard(to, auth)
})

export default router
```

- [ ] **Step 8: 验证**

```bash
cd frontend
pnpm type-check
pnpm test
pnpm build
```

Expected: 全部通过。

- [ ] **Step 9: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): 实现布局骨架、路由与守卫"
```

---

### Task 6: 打通首页 mock 数据展示

**Files:**
- Modify: `frontend/src/main.ts`
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/pages/home/index.vue`

**Interfaces:**
- Consumes: `router`（Task 5）、`useAuthStore`（Task 4）、`listResources`（Task 2）、MSW `worker`（Task 3）、`Resource`（Task 1）
- Produces: 完整的应用装配 —— 启动时按 `VITE_MOCK` 决定是否启用 MSW worker，首页展示资源列表

- [ ] **Step 1: 补全 main.ts**

`frontend/src/main.ts`：

```ts
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'
import './styles/index.css'

async function bootstrap() {
  if (import.meta.env.VITE_MOCK === '1') {
    const { worker } = await import('./mocks/browser')
    await worker.start({ onUnhandledRequest: 'bypass' })
  }

  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  app.use(ElementPlus)
  app.mount('#app')
}

bootstrap()
```

- [ ] **Step 2: 更新 App.vue 挂载路由**

`frontend/src/App.vue`：

```vue
<template>
  <RouterView />
</template>
```

- [ ] **Step 3: 首页接入资源列表数据**

`frontend/src/pages/home/index.vue`：

```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listResources } from '@/api/resource'
import type { Resource } from '@/types'

const resources = ref<Resource[]>([])
const loading = ref(false)
const error = ref('')

onMounted(async () => {
  loading.value = true
  try {
    const data = await listResources({ page: 1, page_size: 12, sort: 'latest' })
    resources.value = data.list
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="mx-auto max-w-7xl px-6 py-10">
    <div class="flex items-end justify-between">
      <div>
        <h1 class="text-2xl font-bold">发现优质教育资源</h1>
        <p class="mt-2 text-sm text-ink-secondary">精选课程、文章与视频</p>
      </div>
      <RouterLink :to="{ name: 'search' }">
        <el-button round>查看全部</el-button>
      </RouterLink>
    </div>

    <div v-if="loading" class="py-24 text-center text-sm text-ink-muted">加载中…</div>
    <div v-else-if="error" class="py-24 text-center text-sm text-red-500">{{ error }}</div>
    <div v-else class="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="r in resources"
        :key="r.id"
        class="group rounded-lg border border-border bg-surface p-5 shadow-sm transition hover:shadow-md"
      >
        <div class="flex items-center justify-between text-xs text-ink-muted">
          <el-tag size="small" :type="r.type === 'course' ? 'primary' : r.type === 'video' ? 'success' : 'warning'">
            {{ r.type }}
          </el-tag>
          <span>{{ r.category?.name }}</span>
        </div>
        <h3 class="mt-3 text-lg font-semibold text-ink">{{ r.title }}</h3>
        <p class="mt-1 line-clamp-2 text-sm text-ink-secondary">{{ r.description }}</p>
        <div class="mt-4 flex items-center justify-between border-t border-border pt-3 text-sm">
          <span class="text-amber-500">★ {{ r.avg_rating.toFixed(1) }}</span>
          <span class="text-xs text-ink-muted">{{ r.view_count }} 次浏览</span>
        </div>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 4: 创建本地 .env 并验证完整链路**

`frontend/.env.development`：

```
VITE_MOCK=1
```

```bash
cd frontend
pnpm build
pnpm test
```

Expected: 构建与测试全部通过。

```bash
cd frontend
pnpm dev
```

Expected: 访问 http://localhost:5173/ 时，未登录会被重定向到 /login；手动访问首页路由前先登录会跳转。用浏览器控制台确认：
1. `navigator.serviceWorker` 已注册（MSW 启动成功，Network 面板请求由 Mock Service Worker 标注）
2. `/api/v1/resources` 返回 `{ code: 0, ... }` 且首页展示出 5 条种子资源卡片

> 说明：首页 `requiresAuth: true`，所以未登录直接访问 `/` 会被守卫重定向到 `/login`。这是符合预期的行为——完整登录页面在子项目 2 实现，届时登录后可正常浏览首页。

- [ ] **Step 5: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): 打通首页 mock 数据展示"
```

---

## 验证清单（全部 Task 完成后）

- [ ] `pnpm build` —— vue-tsc 类型检查 + vite 构建通过
- [ ] `pnpm test` —— 全部 Vitest 测试通过（client 解包、MSW handlers、auth store、路由守卫）
- [ ] `pnpm dev` 启动，MSW worker 注册成功
- [ ] 首页能通过 mock 数据渲染出资源卡片列表
- [ ] 6 次提交均在 `feat/frontend-scaffolding` 分支，每次提交后已停下等待用户确认
