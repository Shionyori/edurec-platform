import { generateMockData } from './generator'

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

export interface DbComment {
  id: number
  resource_id: number
  bvid: string
  author_name: string
  content: string
  like_count: number
  floor: number
  published_at: number
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

// 核心用户：id 稳定（1-4），前两个是现有账号
const coreUsers: DbUser[] = [
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
] as DbUser[]

const coreCategories: DbCategory[] = [
  { id: 1, name: '人工智能', description: 'AI、机器学习、深度学习' },
  { id: 2, name: '前端开发', description: 'HTML、CSS、JavaScript' },
  { id: 3, name: '后端开发', description: 'Go、Python、Java' },
  { id: 4, name: '数据科学', description: '数据清洗、统计分析、可视化' },
  { id: 5, name: '软件工程', description: '设计模式、架构、测试与工程实践' },
  { id: 6, name: '设计', description: 'UI、UX、交互设计基础' },
] as DbCategory[]

// 精选资源：id 稳定（1-5），供"找数据"与测试使用
const coreResources: DbResource[] = [
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
  {
    // B 站视频：演示「打开详情页自动拉取 B 站评论」功能
    id: 6,
    title: '李宏毅机器学习 2024',
    description: 'B 站搬运的李宏毅机器学习课程，从线性代数到 Transformer。',
    cover_url: null,
    type: 'video',
    category_id: 1,
    tags: ['机器学习', 'B站'],
    metadata: { bvid: 'BV1DgxCzREbM', duration: '16:08', pubdate: 1759988744, typename: '知识' },
    author: '李宏毅',
    source_url: 'https://www.bilibili.com/video/BV1DgxCzREbM',
    avg_rating: 4.9,
    view_count: 12345,
    created_at: '2026-08-01T08:00:00Z',
    updated_at: '2026-08-10T10:00:00Z',
  },
] as DbResource[]

const generated = generateMockData({ users: coreUsers, resources: coreResources })

export const db = {
  users: coreUsers,
  categories: coreCategories,
  resources: [...coreResources, ...generated.resources],
  ratings: generated.ratings,
  comments: [
    {
      id: 1,
      resource_id: 6,
      bvid: 'BV1DgxCzREbM',
      author_name: '小明',
      content: '讲得太清楚了，点赞',
      like_count: 120,
      floor: 1,
      published_at: 1759990000,
    },
    {
      id: 2,
      resource_id: 6,
      bvid: 'BV1DgxCzREbM',
      author_name: '阿强',
      content: '跟着学完了，收获很大',
      like_count: 88,
      floor: 2,
      published_at: 1759991000,
    },
    {
      id: 3,
      resource_id: 6,
      bvid: 'BV1DgxCzREbM',
      author_name: '路人甲',
      content: '第 3 章有点难，但值得',
      like_count: 45,
      floor: 3,
      published_at: 1759992000,
    },
  ] as DbComment[],
  behaviors: [
    { id: 1, user_id: 2, resource_id: 1, action: 'view', created_at: now() },
    { id: 2, user_id: 2, resource_id: 2, action: 'click', created_at: now() },
    { id: 3, user_id: 2, resource_id: 4, action: 'favorite', created_at: now() },
  ] as DbBehavior[],
}
