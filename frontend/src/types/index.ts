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
  // 结构化推荐特征：难度（空串=未知）、时长（分钟，0=未知）
  difficulty?: Difficulty | ''
  duration_minutes?: number
  // 推荐理由（仅推荐接口返回时带，可解释性）
  reason?: string
  created_at: string
  updated_at: string
}

// 学习难度：与后端 model.Difficulty* 常量一致
export type Difficulty = 'beginner' | 'intermediate' | 'advanced'

export interface Rating {
  id: number
  user: Pick<User, 'id' | 'username' | 'display_name' | 'avatar_url'>
  score: number
  comment: string | null
  created_at: string
}

export interface BilibiliComment {
  id: number
  author_name: string
  content: string
  like_count: number
  floor: number
  published_at: number
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
  has_more?: boolean
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
  run_id?: string
}

// 一次推荐推理运行的记录（来自 engine 旁挂信封，导入时落库）
export interface RecommendationRun {
  id: number
  run_id: string
  snapshot_run_id: string
  model_name: string
  model_version: string
  encoder: string
  generated_at: number
  top_n: number
  users_count: number
  imported_users: number
  skipped_users: number
  imported_resources: number
  skipped_resources: number
  created_at: number
}

// 推荐效果统计：CTR = clicks / impressions
export interface RecommendationStats {
  impressions: number
  clicks: number
  favorites: number
  views: number
  recommendation_users: number
  click_through_rate: number
}
