import { http, HttpResponse } from 'msw'
import { db, DbBehavior, DbResource, DbUser } from './db'

// MSW 2.15 的 path-only 谓词不匹配带 host 的 URL，统一用通配符前缀匹配任意 origin
const BASE = '*/api/v1'

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
    const body = (await request.json()) as Partial<DbResource> & {
      title: string
      type: DbResource['type']
      category_id: number
    }
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
          resource: r ? { id: r.id, title: r.title, cover_url: r.cover_url, type: r.type } : null,
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
    const list = db.users.filter((u) => !keyword || u.username.includes(keyword) || u.email.includes(keyword))
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
    const list = db.resources.filter((r) => !keyword || r.title.includes(keyword))
    const total = list.length
    const paged = list.slice((page - 1) * pageSize, page * pageSize)
    return ok({ list: paged.map(toPublicResource), total, page, page_size: pageSize })
  }),
]
