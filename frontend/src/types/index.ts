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
