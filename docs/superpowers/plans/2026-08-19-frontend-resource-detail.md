# 前端资源详情页实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将路由 `resources/:id` 从占位页替换为真实资源详情页（资源信息 + 评分评论 + view 行为上报），并引入确定性 mock 数据生成器使演示数据便于测试、查找与再生成。

**Architecture:** 页面按"页面模块化"目录组织：`pages/resource/` 编排数据流，`components/resource/` 拆分评分表单与列表。mock 数据采用"核心手写 + 批量生成"混合策略：用户/分类/精选资源手写保 id 稳定，评分与补充资源由固定 seed 的生成器确定性产出。handler 对齐后端补上评分 upsert 后 avg_rating 重算。

**Tech Stack:** Vue 3 (Composition API) + TypeScript + Element Plus（el-rate / el-input / el-pagination）+ Tailwind + Vitest + Vue Test Utils + MSW。

## Global Constraints

- 组件库为 Element Plus；页面样式使用 Tailwind + 既有主题 tokens（`bg-surface`、`ink-secondary`、`border-border`、`bg-bg`、`text-primary` 等），不要引入新色值。
- API 一律通过 `@/api/*` 封装调用，失败抛出 `Error(message)`（client.ts 已统一处理）。
- 测试：Vitest + Vue Test Utils，`mount()` 时挂 `{ global: { plugins: [ElementPlus] } }`；mock 模块用 `vi.mock`（参考 `pages/auth/__tests__/LoginPage.test.ts` 的写法）。
- mock 数据 id 稳定性硬约束（现有测试依赖，不得改动）：`users` 前 2 个（id 1 admin、id 2 user）、`categories` 前 3 个、`resources` 前 5 个（id 1-5）内容与 id 必须保持不变；新增实体一律追加 id 后延。
- 行为上报 `recordBehavior(id, 'view')` 为 fire-and-forget：`.catch(() => {})`，失败不打断页面渲染。
- 提交规范：`feat(frontend): ...`，每个 Task 一个独立可验证的提交。
- 认证约束：`resources/:id` 路由 `meta.requiresAuth` 保持不变（守卫已在脚手架实现）。

---

### Task 1: 确定性 mock 数据生成器

**Files:**
- Create: `frontend/src/mocks/generator.ts`
- Test: `frontend/src/mocks/__tests__/generator.test.ts`

**Interfaces:**
- Produces:
  - `DATA_CONFIG`（`{ seed, extraResourceCount, minRatingsPerResource, maxRatingsPerResource, hotResourceIds, hotResourceRatingTarget }`）
  - `mulberry32(seed: number): () => number`
  - `generateMockData(input: { users: DbUser[]; resources: DbResource[] }, overrides?: Partial<typeof DATA_CONFIG>): { resources: DbResource[]; ratings: DbRating[] }`
  - 返回的 `resources` 为"补充资源"（id 从 `input.resources` 最大 id 后顺延），`ratings` 覆盖全部资源（含 core）。

- [ ] **Step 1: 写失败测试**

```ts
// frontend/src/mocks/__tests__/generator.test.ts
import { describe, expect, it } from 'vitest'
import { DATA_CONFIG, generateMockData } from '../generator'
import type { DbResource, DbUser } from '../db'

const baseUsers: DbUser[] = [
  { id: 1, username: 'admin', email: 'admin@edurec.dev', password: 'admin123', display_name: null, avatar_url: null, is_admin: true, created_at: '2026-07-01T08:00:00Z' },
  { id: 2, username: 'user', email: 'user@edurec.dev', password: 'user123', display_name: '张三', avatar_url: null, is_admin: false, created_at: '2026-07-05T08:00:00Z' },
]

const baseResources: DbResource[] = [
  {
    id: 1, title: '机器学习入门', description: '零基础入门', cover_url: null, type: 'course',
    category_id: 1, tags: ['AI'], metadata: {}, author: '吴恩达', source_url: null,
    avg_rating: 4.5, view_count: 100, created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-01T08:00:00Z',
  },
]

describe('generateMockData', () => {
  it('同一 seed 两次生成结果一致（确定性）', () => {
    const a = generateMockData({ users: baseUsers, resources: baseResources })
    const b = generateMockData({ users: baseUsers, resources: baseResources })
    expect(a).toEqual(b)
  })

  it('补充资源数量与 id 顺延符合配置', () => {
    const result = generateMockData(
      { users: baseUsers, resources: baseResources },
      { extraResourceCount: 2, minRatingsPerResource: 1, maxRatingsPerResource: 1, hotResourceIds: [], hotResourceRatingTarget: 0 },
    )
    expect(result.resources).toHaveLength(2)
    expect(result.resources[0].id).toBe(2)
    expect(result.resources[1].id).toBe(3)
  })

  it('热门资源堆评分达到目标数（分页演示）', () => {
    const result = generateMockData(
      { users: baseUsers, resources: baseResources },
      { extraResourceCount: 0, hotResourceIds: [1], hotResourceRatingTarget: 14 },
    )
    expect(result.ratings.filter((r) => r.resource_id === 1)).toHaveLength(14)
  })

  it('生成的资源具有重算后的 avg_rating 与正 view_count', () => {
    const result = generateMockData(
      { users: baseUsers, resources: baseResources },
      { extraResourceCount: 1, minRatingsPerResource: 3, maxRatingsPerResource: 3, hotResourceIds: [], hotResourceRatingTarget: 0 },
    )
    const gen = result.resources[0]
    const rs = result.ratings.filter((r) => r.resource_id === gen.id)
    expect(rs).toHaveLength(3)
    expect(gen.avg_rating).toBeGreaterThan(0)
    expect(gen.avg_rating).toBeLessThanOrEqual(5)
    expect(gen.view_count).toBeGreaterThanOrEqual(0)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && pnpm vitest run src/mocks/__tests__/generator.test.ts`
Expected: FAIL — 模块不存在 / `Cannot find module '../generator'`

- [ ] **Step 3: 实现生成器**

```ts
// frontend/src/mocks/generator.ts
import type { DbResource, DbRating, DbUser } from './db'

export const DATA_CONFIG = {
  seed: 42,
  extraResourceCount: 7,
  minRatingsPerResource: 2,
  maxRatingsPerResource: 8,
  hotResourceIds: [1] as number[],
  hotResourceRatingTarget: 14,
}

export type DataConfig = typeof DATA_CONFIG

// mulberry32：确定性的轻量 PRNG（不依赖 Math.random，保证同 seed 结果一致）
export function mulberry32(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

const BASE_DATE = '2026-06-01T08:00:00Z'

type ResourceTemplate = Omit<DbResource, 'id' | 'avg_rating' | 'view_count' | 'created_at' | 'updated_at'>

const RESOURCE_POOL: ResourceTemplate[] = [
  { title: '深度学习实战', description: '从 CNN 到 Transformer，用 PyTorch 搭建真实项目。', cover_url: null, type: 'course', category_id: 1, tags: ['深度学习', 'PyTorch'], metadata: { duration: '10 小时', chapters: 20, language: '中文' }, author: '孙老师', source_url: 'https://example.com/course/dl' },
  { title: '机器学习面试 100 问', description: '高频面试题与思路解析，覆盖算法原理与工程落地。', cover_url: null, type: 'article', category_id: 1, tags: ['面试', '机器学习'], metadata: { word_count: 6000, reading_minutes: 25 }, author: '周老师', source_url: 'https://example.com/article/ml-interview' },
  { title: '大模型微调入门', description: 'LoRA 与 QLoRA 微调实操，从环境搭建到效果评估。', cover_url: null, type: 'video', category_id: 1, tags: ['大模型', 'LoRA'], metadata: { duration: '60 分钟', resolution: '1080p' }, author: '郑老师', source_url: 'https://example.com/video/llm-finetune' },
  { title: 'React 18 从入门到实践', description: 'Hooks、并发渲染与工程化配置，构建可维护的组件库。', cover_url: null, type: 'course', category_id: 2, tags: ['React', 'Hooks'], metadata: { duration: '9 小时', chapters: 22, language: '中文' }, author: '冯老师', source_url: 'https://example.com/course/react18' },
  { title: 'Vue 性能优化指南', description: '运行时开销、懒加载与缓存策略，让应用更快。', cover_url: null, type: 'article', category_id: 2, tags: ['Vue', '性能'], metadata: { word_count: 4200, reading_minutes: 16 }, author: '蒋老师', source_url: 'https://example.com/article/vue-perf' },
  { title: 'Go 微服务实战', description: 'gRPC、服务发现与链路追踪，从单体到微服务演进。', cover_url: null, type: 'course', category_id: 3, tags: ['Go', '微服务'], metadata: { duration: '11 小时', chapters: 26, language: '中文' }, author: '沈老师', source_url: 'https://example.com/course/go-micro' },
  { title: 'MySQL 索引优化', description: '执行计划、慢查询分析与索引设计最佳实践。', cover_url: null, type: 'video', category_id: 3, tags: ['MySQL', '性能'], metadata: { duration: '40 分钟', resolution: '1080p' }, author: '韩老师', source_url: 'https://example.com/video/mysql-index' },
  { title: 'pandas 数据清洗', description: '缺失值、重复值与特征工程，产出高质量数据集。', cover_url: null, type: 'course', category_id: 4, tags: ['pandas', '数据清洗'], metadata: { duration: '7 小时', chapters: 15, language: '中文' }, author: '杨老师', source_url: 'https://example.com/course/pandas' },
  { title: '测试金字塔实践', description: '单元、集成与端到端测试的取舍，降低回归成本。', cover_url: null, type: 'article', category_id: 5, tags: ['测试', '工程实践'], metadata: { word_count: 3800, reading_minutes: 14 }, author: '朱老师', source_url: 'https://example.com/article/test-pyramid' },
  { title: '交互设计基础', description: '信息架构与可用性测试，打造易用的产品体验。', cover_url: null, type: 'video', category_id: 6, tags: ['交互设计', 'UX'], metadata: { duration: '50 分钟', resolution: '1080p' }, author: '秦老师', source_url: 'https://example.com/video/ux-basics' },
]

const COMMENT_POOL = [
  '讲解清晰，收获很大',
  '内容实用，适合进阶',
  '例子很接地气，好评',
  '节奏稍快，建议有一定基础再看',
  '质量不错，值得一看',
  '一般般，期望能更高',
  '干货满满，推荐给同事了',
  '部分章节有点拖沓',
  '讲得很细，跟着做完了',
  '概念讲清楚了，练习再多点更好',
]

// 评分偏置：4-5 居多、少量 3、极少数 1-2
function randomScore(rand: () => number): number {
  const r = rand()
  if (r < 0.5) return 5
  if (r < 0.8) return 4
  if (r < 0.92) return 3
  if (r < 0.98) return 2
  return 1
}

function shuffle<T>(arr: T[], rand: () => number): T[] {
  const a = [...arr]
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(rand() * (i + 1))
    ;[a[i], a[j]] = [a[j], a[i]]
  }
  return a
}

function pick<T>(arr: T[], rand: () => number): T {
  return arr[Math.floor(rand() * arr.length)]
}

export interface GenerateInput {
  users: DbUser[]
  resources: DbResource[]
}

export interface GeneratedMockData {
  resources: DbResource[]
  ratings: DbRating[]
}

export function generateMockData(input: GenerateInput, overrides: Partial<DataConfig> = {}): GeneratedMockData {
  const config = { ...DATA_CONFIG, ...overrides }
  const rand = mulberry32(config.seed)
  const nextId = Math.max(...input.resources.map((r) => r.id), 0) + 1

  const generatedResources: DbResource[] = shuffle(RESOURCE_POOL, rand)
    .slice(0, config.extraResourceCount)
    .map((tpl, i) => ({
      ...tpl,
      id: nextId + i,
      avg_rating: 0,
      view_count: 50 + Math.floor(rand() * 900),
      created_at: BASE_DATE,
      updated_at: BASE_DATE,
    }))

  const allResources = [...input.resources, ...generatedResources]
  const regularUsers = input.users.filter((u) => !u.is_admin)

  const ratings: DbRating[] = []
  let ratingId = 1
  for (const r of allResources) {
    if (regularUsers.length === 0) continue
    const isHot = config.hotResourceIds.includes(r.id)
    const count = isHot
      ? config.hotResourceRatingTarget
      : config.minRatingsPerResource +
        Math.floor(rand() * (config.maxRatingsPerResource - config.minRatingsPerResource + 1))
    for (let k = 0; k < count; k++) {
      const user = pick(regularUsers, rand)
      ratings.push({
        id: ratingId++,
        user_id: user.id,
        resource_id: r.id,
        score: randomScore(rand),
        comment: rand() < 0.8 ? pick(COMMENT_POOL, rand) : null,
        created_at: BASE_DATE,
      })
    }
  }

  // 按生成的评分回填 avg_rating（保持首页/列表展示与评分数据一致）
  for (const r of allResources) {
    const rs = ratings.filter((x) => x.resource_id === r.id)
    if (rs.length > 0) {
      r.avg_rating = Math.round((rs.reduce((s, x) => s + x.score, 0) / rs.length) * 10) / 10
    }
  }

  return { resources: generatedResources, ratings }
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && pnpm vitest run src/mocks/__tests__/generator.test.ts`
Expected: PASS（4 个用例全绿）

- [ ] **Step 5: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add frontend/src/mocks/generator.ts frontend/src/mocks/__tests__/generator.test.ts
git commit -m "feat(frontend): 添加确定性 mock 数据生成器"
```

---

### Task 2: 扩充 db.ts 核心实体并集成生成器

**Files:**
- Modify: `frontend/src/mocks/db.ts`（追加用户/分类、保留核心资源、组装生成数据）
- Test: `frontend/src/mocks/__tests__/db.test.ts`（新建）

**Interfaces:**
- Consumes: `generateMockData`（Task 1）、`DATA_CONFIG`（Task 1）
- Produces: `db`（含 4 用户、6 分类、12 资源、生成评分、原有行为示例）——保持现有导出形状不变

- [ ] **Step 1: 写失败测试**

```ts
// frontend/src/mocks/__tests__/db.test.ts
import { describe, expect, it } from 'vitest'
import { db } from '../db'
import { DATA_CONFIG } from '../generator'

describe('mock db 种子数据', () => {
  it('核心实体 id 稳定（现有测试依赖）', () => {
    expect(db.users[0].username).toBe('admin')
    expect(db.users[1].username).toBe('user')
    expect(db.categories[0].name).toBe('人工智能')
    expect(db.resources[0].id).toBe(1)
    expect(db.resources[0].title).toBe('机器学习入门')
  })

  it('用户 4 个、分类 6 个', () => {
    expect(db.users).toHaveLength(4)
    expect(db.categories).toHaveLength(6)
  })

  it('资源总数 = 精选 + 生成数量', () => {
    expect(db.resources).toHaveLength(5 + DATA_CONFIG.extraResourceCount)
  })

  it('热门资源 1 评分数达到分页演示阈值', () => {
    const count = db.ratings.filter((r) => r.resource_id === 1).length
    expect(count).toBeGreaterThanOrEqual(DATA_CONFIG.hotResourceRatingTarget)
  })

  it('生成资源无重复 id，且 id 全部顺延', () => {
    const ids = db.resources.map((r) => r.id)
    expect(new Set(ids).size).toBe(ids.length)
    expect(ids).toEqual([...ids].sort((a, b) => a - b))
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && pnpm vitest run src/mocks/__tests__/db.test.ts`
Expected: FAIL — 用户数/分类数/资源数断言不通过（当前 2 用户 / 3 分类 / 5 资源）

- [ ] **Step 3: 修改 db.ts**

在 `db.ts` 顶部引入生成器：

```ts
import { generateMockData } from './generator'
```

保留全部接口定义（`DbUser`、`DbCategory`、`DbResourceType`、`DbResource`、`DbRating`、`DbBehaviorAction`、`DbBehavior`）与 `now()` 助手不变。将 `users` 扩为 4 个（前两个保持不变，追加）：

```ts
    {
      id: 3,
      username: 'lisi',
      email: 'lisi@edurec.dev',
      password: 'lisi123',
      display_name: '李四',
      avatar_url: null,
      is_admin: false,
      created_at: '2026-07-08T08:00:00Z',
    },
    {
      id: 4,
      username: 'wangwu',
      email: 'wangwu@edurec.dev',
      password: 'wangwu123',
      display_name: '王五',
      avatar_url: null,
      is_admin: false,
      created_at: '2026-07-10T08:00:00Z',
    },
```

`categories` 扩为 6 个（前 3 个保持不变，追加）：

```ts
    { id: 4, name: '数据科学', description: '数据清洗、统计分析、可视化' },
    { id: 5, name: '软件工程', description: '设计模式、架构、测试与工程实践' },
    { id: 6, name: '设计', description: 'UI、UX、交互设计基础' },
```

**将 `db.ts` 改造成"核心数组 + 组装"结构**。保留全部接口定义、`now()` 助手不变；删除原 `export const db = { ... }` 字面量，改为：

```ts
import { generateMockData } from './generator'

// （中间保留全部接口定义与 now() 助手，原样不动）

// 核心用户：id 稳定（1-4），前两个是现有账号，后两个新增
const coreUsers: DbUser[] = [
  { id: 1, username: 'admin', email: 'admin@edurec.dev', password: 'admin123', display_name: '平台管理员', avatar_url: null, is_admin: true, created_at: '2026-07-01T08:00:00Z' },
  { id: 2, username: 'user', email: 'user@edurec.dev', password: 'user123', display_name: '张三', avatar_url: null, is_admin: false, created_at: '2026-07-05T08:00:00Z' },
  { id: 3, username: 'lisi', email: 'lisi@edurec.dev', password: 'lisi123', display_name: '李四', avatar_url: null, is_admin: false, created_at: '2026-07-08T08:00:00Z' },
  { id: 4, username: 'wangwu', email: 'wangwu@edurec.dev', password: 'wangwu123', display_name: '王五', avatar_url: null, is_admin: false, created_at: '2026-07-10T08:00:00Z' },
] as DbUser[]

const coreCategories: DbCategory[] = [
  { id: 1, name: '人工智能', description: 'AI、机器学习、深度学习' },
  { id: 2, name: '前端开发', description: 'HTML、CSS、JavaScript' },
  { id: 3, name: '后端开发', description: 'Go、Python、Java' },
  { id: 4, name: '数据科学', description: '数据清洗、统计分析、可视化' },
  { id: 5, name: '软件工程', description: '设计模式、架构、测试与工程实践' },
  { id: 6, name: '设计', description: 'UI、UX、交互设计基础' },
] as DbCategory[]

// 精选资源：id 稳定（1-5），内容取原 db.ts resources 数组里的 5 个字面量，逐字保留
const coreResources: DbResource[] = [
  { id: 1, title: '机器学习入门', /* …原有全部字段，原样保留… */ },
  { id: 2, title: 'Vue 3 实战', /* … */ },
  { id: 3, title: 'Go 语言高并发编程', /* … */ },
  { id: 4, title: 'Python 数据分析', /* … */ },
  { id: 5, title: 'TypeScript 类型体操', /* … */ },
] as DbResource[]

const generated = generateMockData({ users: coreUsers, resources: coreResources })

export const db = {
  users: coreUsers,
  categories: coreCategories,
  resources: [...coreResources, ...generated.resources],
  ratings: generated.ratings,
  behaviors: [
    { id: 1, user_id: 2, resource_id: 1, action: 'view', created_at: now() },
    { id: 2, user_id: 2, resource_id: 2, action: 'click', created_at: now() },
    { id: 3, user_id: 2, resource_id: 4, action: 'favorite', created_at: now() },
  ] as DbBehavior[],
}
```

> 说明：原 `db` 对象里手写的 `ratings`（3 条）由生成器产出的评分取代；原 `resources` 数组 5 个资源原样搬进 `coreResources`（仅改所在数组，字段逐字不变），再与生成资源拼接。

- [ ] **Step 4: 运行 db 测试 + 既有测试确认通过**

Run: `cd frontend && pnpm vitest run src/mocks/__tests__`
Expected: PASS — 新增 db.test.ts 与既有 handlers.test.ts 全绿（证明核心 id 未破坏）

- [ ] **Step 5: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add frontend/src/mocks/db.ts frontend/src/mocks/__tests__/db.test.ts
git commit -m "feat(frontend): 扩充 mock 种子数据并集成生成器"
```

---

### Task 3: mock 评分 upsert 后重算 avg_rating

**Files:**
- Modify: `frontend/src/mocks/handlers.ts`（`POST /resources/:id/ratings` handler）
- Test: `frontend/src/mocks/__tests__/handlers.test.ts`（补一个用例）

**Interfaces:**
- Consumes: `db`（Task 2 的组装结果）、已有 `upsertRating`（`@/api/rating`）
- Produces: 无新导出；修正 `POST /resources/:id/ratings` 行为——创建或更新评分后重算该资源 `avg_rating`（保留一位小数），对齐后端 `RatingService.updateResourceAverage`。

- [ ] **Step 1: 写失败测试**

在 `handlers.test.ts` 顶部修改两处导入：将现有 `import { listResources } from '@/api/resource'` 改为 `import { listResources, getResource } from '@/api/resource'`，并新增 `import { upsertRating } from '@/api/rating'`。

在 `describe('MSW handlers')` 内追加用例：

```ts
  it('评分 upsert 后重算 avg_rating', async () => {
    await loginAsAdmin()
    const before = await getResource(1)
    await upsertRating(1, { score: 1 })
    const after = await getResource(1)
    expect(after.avg_rating).not.toBe(before.avg_rating)
    expect(after.avg_rating).toBeGreaterThan(0)
    expect(after.avg_rating).toBeLessThanOrEqual(5)
  })
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && pnpm vitest run src/mocks/__tests__/handlers.test.ts`
Expected: FAIL — upsert 后 avg_rating 未变化（`after.avg_rating === before.avg_rating`）

- [ ] **Step 3: 实现重算**

在 `handlers.ts` 中，`// 评分` 区块之前添加辅助函数：

```ts
function recomputeResourceAverage(resourceId: number) {
  const list = db.ratings.filter((r) => r.resource_id === resourceId)
  const resource = db.resources.find((r) => r.id === resourceId)
  if (!resource) return
  if (list.length === 0) {
    resource.avg_rating = 0
    return
  }
  const avg = list.reduce((sum, r) => sum + r.score, 0) / list.length
  resource.avg_rating = Math.round(avg * 10) / 10
}
```

在 `http.post(`${BASE}/resources/:id/ratings`)` handler 内，创建/更新分支完成后、`return ok(...)` 之前调用（两处都要）：

```ts
    recomputeResourceAverage(resourceId)
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && pnpm vitest run src/mocks/__tests__/handlers.test.ts`
Expected: PASS（含新增用例）

- [ ] **Step 5: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add frontend/src/mocks/handlers.ts frontend/src/mocks/__tests__/handlers.test.ts
git commit -m "fix(frontend): mock 评分 upsert 后重算 avg_rating"
```

---

### Task 4: 评价表单组件 RatingForm

**Files:**
- Create: `frontend/src/components/resource/RatingForm.vue`
- Test: `frontend/src/components/resource/__tests__/RatingForm.test.ts`

**Interfaces:**
- Consumes: 无
- Produces: `RatingForm` 组件
  - Props: `initialScore?: number`（默认 0）、`initialComment?: string`（默认 ''）、`submitting?: boolean`
  - Emits: `submit(payload: { score: number; comment: string })`
  - 内部状态 `score`/`comment` 通过 props watch 回填/重置

- [ ] **Step 1: 写失败测试**

```ts
// frontend/src/components/resource/__tests__/RatingForm.test.ts
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import RatingForm from '../RatingForm.vue'

async function setScore(wrapper: ReturnType<typeof mount>, score: number) {
  await wrapper.findComponent({ name: 'ElRate' }).vm.$emit('update:modelValue', score)
}

describe('RatingForm', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('未选评分提交时提示且不 emit', async () => {
    vi.spyOn(ElMessage, 'warning').mockImplementation(() => ({}) as never)
    const wrapper = mount(RatingForm, { global: { plugins: [ElementPlus] } })
    await wrapper.find('button').trigger('click')
    expect(ElMessage.warning).toHaveBeenCalledWith('请先选择评分')
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('选评分 + 评论后 emit 载荷', async () => {
    const wrapper = mount(RatingForm, { global: { plugins: [ElementPlus] } })
    await setScore(wrapper, 4)
    await wrapper.find('textarea').setValue('  讲解清晰  ')
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({ score: 4, comment: '讲解清晰' })
  })

  it('initialScore / initialComment 回填', () => {
    const wrapper = mount(RatingForm, {
      props: { initialScore: 3, initialComment: '回填评论' },
      global: { plugins: [ElementPlus] },
    })
    expect(wrapper.findComponent({ name: 'ElRate' }).props('modelValue')).toBe(3)
    expect(wrapper.find('textarea').element.value).toBe('回填评论')
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && pnpm vitest run src/components/resource/__tests__/RatingForm.test.ts`
Expected: FAIL — 模块不存在

- [ ] **Step 3: 实现组件**

```vue
<!-- frontend/src/components/resource/RatingForm.vue -->
<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  initialScore?: number
  initialComment?: string
  submitting?: boolean
}>()

const emit = defineEmits<{
  submit: [payload: { score: number; comment: string }]
}>()

const score = ref(props.initialScore ?? 0)
const comment = ref(props.initialComment ?? '')

watch(
  () => [props.initialScore, props.initialComment] as const,
  ([s, c]) => {
    score.value = s ?? 0
    comment.value = c ?? ''
  },
)

function onSubmit() {
  if (score.value < 1) {
    ElMessage.warning('请先选择评分')
    return
  }
  emit('submit', { score: score.value, comment: comment.value.trim() })
}
</script>

<template>
  <div class="space-y-4">
    <div>
      <div class="mb-1 text-sm text-ink-secondary">你的评分</div>
      <el-rate v-model="score" :max="5" />
    </div>
    <el-input
      v-model="comment"
      type="textarea"
      :rows="3"
      maxlength="500"
      show-word-limit
      placeholder="说说你的学习感受（选填）"
    />
    <el-button type="primary" class="w-full" :loading="submitting" @click="onSubmit">
      提交评价
    </el-button>
  </div>
</template>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && pnpm vitest run src/components/resource/__tests__/RatingForm.test.ts`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add frontend/src/components/resource/RatingForm.vue frontend/src/components/resource/__tests__/RatingForm.test.ts
git commit -m "feat(frontend): 实现评价表单组件"
```

---

### Task 5: 评价列表组件 RatingList + 日期格式化工具

**Files:**
- Create: `frontend/src/components/resource/RatingList.vue`
- Create: `frontend/src/utils/format.ts`
- Test: `frontend/src/components/resource/__tests__/RatingList.test.ts`

**Interfaces:**
- Consumes: `Rating`（`@/types`）、`formatDate`（本任务产出）
- Produces:
  - `formatDate(iso: string): string`（`YYYY-MM-DD HH:mm`，非法输入原样返回）
  - `RatingList` 组件
    - Props: `ratings: Rating[]`、`total: number`、`page: number`、`loading?: boolean`
    - Emits: `page-change(page: number)`
    - 分页 `page_size` 固定 10；`total > 10` 时才显示分页器

- [ ] **Step 1: 写失败测试**

```ts
// frontend/src/components/resource/__tests__/RatingList.test.ts
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { formatDate } from '@/utils/format'
import type { Rating } from '@/types'
import RatingList from '../RatingList.vue'

const ratings: Rating[] = [
  {
    id: 1,
    user: { id: 2, username: 'user', display_name: '张三', avatar_url: null },
    score: 5,
    comment: '讲解清晰，收获很大',
    // 无时区后缀的本地时间字符串，使断言与运行环境时区无关
    created_at: '2026-07-20T10:00:00',
  },
]

function mountList(props: Record<string, unknown>) {
  return mount(RatingList, { props, global: { plugins: [ElementPlus] } })
}

describe('RatingList', () => {
  it('渲染评价条目（昵称 / 评分 / 评论 / 时间）', () => {
    const wrapper = mountList({ ratings, total: 1, page: 1 })
    expect(wrapper.text()).toContain('张三')
    expect(wrapper.text()).toContain('讲解清晰，收获很大')
    expect(wrapper.text()).toContain('2026-07-20 10:00')
    expect(wrapper.findComponent({ name: 'ElRate' }).props('modelValue')).toBe(5)
  })

  it('空数据展示占位文案', () => {
    const wrapper = mountList({ ratings: [], total: 0, page: 1 })
    expect(wrapper.text()).toContain('暂无评价')
  })

  it('total > 10 时切页 emit page-change', async () => {
    const wrapper = mountList({ ratings, total: 15, page: 1 })
    await wrapper.findComponent({ name: 'ElPagination' }).vm.$emit('current-change', 2)
    expect(wrapper.emitted('page-change')?.[0]?.[0]).toBe(2)
  })

  it('total <= 10 时不显示分页器', () => {
    const wrapper = mountList({ ratings, total: 5, page: 1 })
    expect(wrapper.findComponent({ name: 'ElPagination' }).exists()).toBe(false)
  })

  it('formatDate 格式化本地时间，非法输入原样返回', () => {
    expect(formatDate('2026-07-20T10:00:00')).toBe('2026-07-20 10:00')
    expect(formatDate('not-a-date')).toBe('not-a-date')
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && pnpm vitest run src/components/resource/__tests__/RatingList.test.ts`
Expected: FAIL — 模块不存在

- [ ] **Step 3: 实现工具与组件**

```ts
// frontend/src/utils/format.ts
export function formatDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
```

```vue
<!-- frontend/src/components/resource/RatingList.vue -->
<script setup lang="ts">
import { formatDate } from '@/utils/format'
import type { Rating } from '@/types'

defineProps<{
  ratings: Rating[]
  total: number
  page: number
  loading?: boolean
}>()

const emit = defineEmits<{
  'page-change': [page: number]
}>()
</script>

<template>
  <div>
    <div v-if="loading" class="py-16 text-center text-sm text-ink-muted">加载中…</div>
    <div v-else-if="ratings.length === 0" class="py-16 text-center text-sm text-ink-muted">
      暂无评价，来抢沙发吧
    </div>
    <ul v-else class="divide-y divide-border">
      <li v-for="r in ratings" :key="r.id" class="flex gap-4 py-4">
        <el-avatar :size="36">{{ (r.user.display_name ?? r.user.username).charAt(0) }}</el-avatar>
        <div class="min-w-0 flex-1">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium text-ink">{{ r.user.display_name ?? r.user.username }}</span>
            <span class="text-xs text-ink-muted">{{ formatDate(r.created_at) }}</span>
          </div>
          <el-rate :model-value="r.score" disabled class="mt-1" />
          <p v-if="r.comment" class="mt-1 text-sm text-ink-secondary">{{ r.comment }}</p>
        </div>
      </li>
    </ul>
    <div v-if="total > 10" class="mt-4 flex justify-center">
      <el-pagination
        :current-page="page"
        :page-size="10"
        :total="total"
        layout="prev, pager, next"
        @current-change="(p: number) => emit('page-change', p)"
      />
    </div>
  </div>
</template>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && pnpm vitest run src/components/resource/__tests__/RatingList.test.ts`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add frontend/src/utils/format.ts frontend/src/components/resource/RatingList.vue frontend/src/components/resource/__tests__/RatingList.test.ts
git commit -m "feat(frontend): 实现评价列表组件与日期格式化"
```

---

### Task 6: 资源详情页 + 路由接入

**Files:**
- Create: `frontend/src/pages/resource/ResourceDetailPage.vue`
- Test: `frontend/src/pages/resource/__tests__/ResourceDetailPage.test.ts`（新建）
- Modify: `frontend/src/router/index.ts`（`resources/:id` 的 component）

**Interfaces:**
- Consumes: `getResource`（`@/api/resource`）、`listRatings`/`upsertRating`（`@/api/rating`）、`recordBehavior`（`@/api/behavior`）、`useAuthStore`、`RatingForm`（Task 4）、`RatingList`（Task 5）、`formatDate`（Task 5）
- Produces: `ResourceDetailPage` 组件（挂载于路由 `resources/:id`）
  - 内部 `PAGE_SIZE = 10`
  - 对外行为：挂载后 `getResource(id)`（成功后静默 `recordBehavior(id, 'view').catch(() => {})`，404 不上报）+ `listRatings(id, page 1)`；提交评分后重拉资源与评分列表并重置表单；`auth.user` 在初始评分页命中时回填表单一次（`prefilled` 守卫，提交后不再回填）

- [ ] **Step 1: 写失败测试**

```ts
// frontend/src/pages/resource/__tests__/ResourceDetailPage.test.ts
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { Rating, Resource } from '@/types'
import ResourceDetailPage from '../ResourceDetailPage.vue'

const pushMock = vi.fn()
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '1' } }),
  useRouter: () => ({ push: pushMock }),
  RouterLink: { template: '<a><slot /></a>' },
}))

const getResourceMock = vi.fn()
const listRatingsMock = vi.fn()
const upsertRatingMock = vi.fn()
const recordBehaviorMock = vi.fn()

vi.mock('@/api/resource', () => ({ getResource: getResourceMock }))
vi.mock('@/api/rating', () => ({
  listRatings: listRatingsMock,
  upsertRating: upsertRatingMock,
}))
vi.mock('@/api/behavior', () => ({ recordBehavior: recordBehaviorMock }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { id: 2, username: 'user' } }) }))

const resource: Resource = {
  id: 1, title: '机器学习入门', description: '面向零基础学习者',
  cover_url: null, type: 'course',
  category: { id: 1, name: '人工智能' },
  tags: ['AI'], metadata: { duration: '12 小时', chapters: 24, language: '中文' },
  author: '吴恩达', source_url: 'https://example.com',
  avg_rating: 4.5, view_count: 1024,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-15T10:00:00Z',
}

const emptyPage = { list: [] as Rating[], total: 0, page: 1, page_size: 10 }

async function mountPage() {
  const wrapper = mount(ResourceDetailPage, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

describe('ResourceDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getResourceMock.mockResolvedValue(resource)
    listRatingsMock.mockResolvedValue(emptyPage)
    recordBehaviorMock.mockResolvedValue(null)
  })

  it('渲染资源信息并自动上报 view 行为', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('机器学习入门')
    expect(wrapper.text()).toContain('面向零基础学习者')
    expect(recordBehaviorMock).toHaveBeenCalledWith(1, 'view')
    expect(listRatingsMock).toHaveBeenCalledWith(1, { page: 1, page_size: 10 })
  })

  it('资源 404 时展示错误态', async () => {
    getResourceMock.mockRejectedValue(new Error('资源不存在'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('资源不存在')
    expect(recordBehaviorMock).not.toHaveBeenCalled()
  })

  it('提交评分后重拉资源与评分列表', async () => {
    upsertRatingMock.mockResolvedValue({ id: 99, score: 4, comment: '好', created_at: '2026-08-01T00:00:00Z' })
    const wrapper = await mountPage()
    getResourceMock.mockClear()
    listRatingsMock.mockClear()
    const form = wrapper.findComponent({ name: 'RatingForm' })
    form.vm.$emit('submit', { score: 4, comment: '好' })
    await flushPromises()
    expect(upsertRatingMock).toHaveBeenCalledWith(1, { score: 4, comment: '好' })
    expect(getResourceMock).toHaveBeenCalledWith(1)
    expect(listRatingsMock).toHaveBeenCalledWith(1, { page: 1, page_size: 10 })
  })

  it('切页时重新拉取评分', async () => {
    const wrapper = await mountPage()
    listRatingsMock.mockClear()
    const list = wrapper.findComponent({ name: 'RatingList' })
    list.vm.$emit('page-change', 2)
    await flushPromises()
    expect(listRatingsMock).toHaveBeenCalledWith(1, { page: 2, page_size: 10 })
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && pnpm vitest run src/pages/resource/__tests__/ResourceDetailPage.test.ts`
Expected: FAIL — 模块不存在

- [ ] **Step 3: 实现页面**

```vue
<!-- frontend/src/pages/resource/ResourceDetailPage.vue -->
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getResource } from '@/api/resource'
import { listRatings, upsertRating } from '@/api/rating'
import { recordBehavior } from '@/api/behavior'
import { useAuthStore } from '@/stores/auth'
import { formatDate } from '@/utils/format'
import type { Rating, Resource } from '@/types'
import RatingForm from '@/components/resource/RatingForm.vue'
import RatingList from '@/components/resource/RatingList.vue'

const PAGE_SIZE = 10

const route = useRoute()
const auth = useAuthStore()
const resourceId = computed(() => Number(route.params.id))

const resource = ref<Resource | null>(null)
const resourceLoading = ref(true)
const resourceError = ref('')

const ratings = ref<Rating[]>([])
const total = ref(0)
const page = ref(1)
const ratingsLoading = ref(false)

const submitting = ref(false)
const initialScore = ref(0)
const initialComment = ref('')
let prefilled = false

async function loadResource() {
  resourceLoading.value = true
  resourceError.value = ''
  try {
    resource.value = await getResource(resourceId.value)
    // 资源加载成功才上报 view（404 时不产生行为记录）
    recordBehavior(resourceId.value, 'view').catch(() => {})
  } catch (e) {
    resourceError.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    resourceLoading.value = false
  }
}

// 仅初始加载回填一次"我的评分"；提交后不再回填（走重置）
function prefillMyRating() {
  if (prefilled) return
  const mine = ratings.value.find((r) => r.user.id === auth.user?.id)
  if (mine) {
    initialScore.value = mine.score
    initialComment.value = mine.comment ?? ''
    prefilled = true
  }
}

async function loadRatings(targetPage = page.value) {
  ratingsLoading.value = true
  try {
    const data = await listRatings(resourceId.value, { page: targetPage, page_size: PAGE_SIZE })
    ratings.value = data.list
    total.value = data.total
    page.value = data.page
    prefillMyRating()
  } finally {
    ratingsLoading.value = false
  }
}

async function handleSubmit(payload: { score: number; comment: string }) {
  submitting.value = true
  try {
    await upsertRating(resourceId.value, payload)
    ElMessage.success('评价成功')
    await Promise.all([loadResource(), loadRatings()])
    // 提交成功后表单重置（RatingForm 通过 props watch 同步）
    initialScore.value = 0
    initialComment.value = ''
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '提交失败，请重试')
  } finally {
    submitting.value = false
  }
}

function handlePageChange(p: number) {
  loadRatings(p)
}

function openSource() {
  if (resource.value?.source_url) {
    window.open(resource.value.source_url, '_blank')
  }
}

onMounted(() => {
  loadResource()
  loadRatings(1)
})
</script>

<template>
  <div class="mx-auto max-w-7xl px-6 py-8">
    <div v-if="resourceLoading" class="py-24 text-center text-sm text-ink-muted">加载中…</div>
    <div v-else-if="resourceError" class="py-24 text-center">
      <p class="text-sm text-red-500">{{ resourceError }}</p>
      <RouterLink :to="{ name: 'home' }">
        <el-button class="mt-4">返回首页</el-button>
      </RouterLink>
    </div>

    <template v-else-if="resource">
      <div class="grid grid-cols-1 gap-8 lg:grid-cols-3">
        <!-- 主内容 -->
        <div class="lg:col-span-2">
          <div class="flex items-center gap-2">
            <el-tag size="small" :type="resource.type === 'course' ? 'primary' : resource.type === 'video' ? 'success' : 'warning'">
              {{ resource.type }}
            </el-tag>
            <span class="text-xs text-ink-muted">更新于 {{ formatDate(resource.updated_at) }}</span>
          </div>
          <h1 class="mt-3 text-2xl font-bold text-ink">{{ resource.title }}</h1>
          <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-ink-secondary">
            <span>{{ resource.category?.name }}</span>
            <span v-for="t in resource.tags" :key="t" class="text-primary">#{{ t }}</span>
            <span>作者：{{ resource.author }}</span>
            <span class="text-amber-500">★ {{ resource.avg_rating.toFixed(1) }}</span>
            <span>{{ resource.view_count }} 次浏览</span>
          </div>

          <div class="mt-6 rounded-lg border border-border bg-surface p-5">
            <h2 class="text-base font-semibold text-ink">简介</h2>
            <p class="mt-2 whitespace-pre-line text-sm leading-relaxed text-ink-secondary">{{ resource.description }}</p>
          </div>

          <div v-if="Object.keys(resource.metadata).length > 0" class="mt-4 rounded-lg border border-border bg-surface p-5">
            <h2 class="text-base font-semibold text-ink">元信息</h2>
            <dl class="mt-2 grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
              <div v-for="(value, key) in resource.metadata" :key="key">
                <dt class="text-ink-muted">{{ key }}</dt>
                <dd class="text-ink">{{ value }}</dd>
              </div>
            </dl>
          </div>

          <div v-if="resource.source_url" class="mt-4">
            <el-button type="primary" round @click="openSource">前往原站学习</el-button>
          </div>

          <div class="mt-8">
            <h2 class="text-base font-semibold text-ink">用户评价（{{ total }}）</h2>
            <RatingList
              :ratings="ratings"
              :total="total"
              :page="page"
              :loading="ratingsLoading"
              @page-change="handlePageChange"
            />
          </div>
        </div>

        <!-- 右侧评分卡 -->
        <aside>
          <div class="sticky top-20 rounded-lg border border-border bg-surface p-5">
            <div class="flex items-end justify-between">
              <div class="text-3xl font-bold text-ink">{{ resource.avg_rating.toFixed(1) }}</div>
              <div class="text-sm text-ink-muted">{{ total }} 条评价</div>
            </div>
            <el-rate :model-value="Math.round(resource.avg_rating)" disabled class="mt-2" />
            <div class="mt-4 border-t border-border pt-4">
              <RatingForm
                :initial-score="initialScore"
                :initial-comment="initialComment"
                :submitting="submitting"
                @submit="handleSubmit"
              />
            </div>
          </div>
        </aside>
      </div>
    </template>
  </div>
</template>
```

修改 `frontend/src/router/index.ts`：顶部导入替换占位页引用。

```ts
import ResourceDetailPage from '@/pages/resource/ResourceDetailPage.vue'
```

将子路由改为：

```ts
      { path: 'resources/:id', name: 'resource-detail', component: ResourceDetailPage, meta: { requiresAuth: true } },
```

- [ ] **Step 4: 运行页面测试 + 类型检查**

Run: `cd frontend && pnpm vitest run src/pages/resource && pnpm type-check`
Expected: PASS + 类型检查无错误

- [ ] **Step 5: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add frontend/src/pages/resource/ResourceDetailPage.vue frontend/src/pages/resource/__tests__/ResourceDetailPage.test.ts frontend/src/router/index.ts
git commit -m "feat(frontend): 实现资源详情页"
```

---

### Task 7: 全量验证

**Files:**
- 无代码改动（仅验证）

- [ ] **Step 1: 全量测试 + 构建**

Run: `cd frontend && pnpm test`
Expected: 全部测试通过（含既有认证、脚手架、mock 测试）

Run: `cd frontend && pnpm build`
Expected: 构建成功，无类型错误

- [ ] **Step 2: 手动验证清单（pnpm dev）**

Run: `cd frontend && pnpm dev`，浏览器访问 http://localhost:5173

| 检查项 | 预期 |
|---|---|
| `admin/admin123` 登录后访问 `/resources/1` | 两栏布局完整渲染：标题/分类/标签/作者/星级/浏览数/描述/元信息/原站链接 |
| 右侧评分卡 | 显示平均分 + 评价条数 + 评价表单 |
| 评价列表 | 显示多条评价（头像/昵称/星级/评论/时间），资源 1 有 2 页分页 |
| 提交评分（选 5 星 + 评论） | ElMessage 成功提示，平均分变化，列表刷新，表单回填为当前评分 |
| 刷新页面 | 浏览数 +1，行为历史中新增 view 记录 |
| 访问 `/resources/999` | 显示"资源不存在" + 返回首页按钮 |
| 首页 `/` | 资源卡片增至 12 个 |

- [ ] **Step 3: 请求合入确认**

此时实现完成，向用户确认后按项目流程将 `feat/frontend-resource-detail` 合入 `main`。
