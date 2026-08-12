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
