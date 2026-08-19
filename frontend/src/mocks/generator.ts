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
