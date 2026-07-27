# edurec-platform API 设计文档

> 版本：0.1.0 | 基础 URL：`/api/v1`

## 规范约定

### 统一响应格式

所有接口返回以下 JSON 结构：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 业务状态码，`0` 表示成功，非 0 表示错误 |
| message | string | 状态描述 |
| data | object / array / null | 响应数据 |

### 分页响应

需要分页的接口，`data` 使用以下结构：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 错误码

| code | HTTP 状态码 | 说明 |
|------|-------------|------|
| 0 | 200 | 成功 |
| 10001 | 400 | 请求参数错误 |
| 10002 | 401 | 未认证 / token 失效 |
| 10003 | 403 | 无权限 |
| 10004 | 404 | 资源不存在 |
| 10005 | 409 | 资源冲突（如用户名已存在） |
| 10006 | 500 | 服务器内部错误 |

### 认证

- 除登录、注册外，所有接口需在 Header 中携带 JWT Access Token：

```
Authorization: Bearer <access_token>
```

- Access Token 过期后，调用 `/auth/refresh` 使用 Refresh Token 换取新 Token

---

## 接口列表

### 1. 认证模块

#### 1.1 注册

```
POST /api/v1/auth/register
```

**Request Body**

```json
{
  "username": "string，必填，3-64 字符",
  "email": "string，必填，合法邮箱格式",
  "password": "string，必填，6-128 字符",
  "display_name": "string，选填"
}
```

**Response (201)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user": {
      "id": 1,
      "username": "zhangsan",
      "email": "zhangsan@example.com",
      "display_name": "张三",
      "avatar_url": null,
      "created_at": "2026-07-27T10:00:00Z"
    }
  }
}
```

---

#### 1.2 登录

```
POST /api/v1/auth/login
```

**Request Body**

```json
{
  "username": "string，必填（支持用户名或邮箱）",
  "password": "string，必填"
}
```

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "access_token": "eyJhbG...",
    "refresh_token": "dGhpcyBp...",
    "expires_in": 900,
    "user": {
      "id": 1,
      "username": "zhangsan",
      "email": "zhangsan@example.com",
      "display_name": "张三",
      "avatar_url": null
    }
  }
}
```

| 字段 | 说明 |
|------|------|
| access_token | JWT，有效期 15 分钟 |
| refresh_token | 用于换取新 access token，有效期 7 天 |
| expires_in | access token 剩余有效期（秒） |

---

#### 1.3 刷新 Token

```
POST /api/v1/auth/refresh
```

**Request Body**

```json
{
  "refresh_token": "string，必填"
}
```

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "access_token": "eyJhbG...",
    "refresh_token": "dGhpcyBp...",
    "expires_in": 900
  }
}
```

---

### 2. 用户模块

#### 2.1 获取当前用户信息

```
GET /api/v1/users/me
```

**需要认证：是**

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": 1,
    "username": "zhangsan",
    "email": "zhangsan@example.com",
    "display_name": "张三",
    "avatar_url": null,
    "is_admin": true,
    "created_at": "2026-07-27T10:00:00Z",
    "updated_at": "2026-07-27T10:00:00Z"
  }
}
```

---

#### 2.2 更新当前用户信息

```
PUT /api/v1/users/me
```

**需要认证：是**

**Request Body**

```json
{
  "display_name": "string，选填",
  "avatar_url": "string，选填"
}
```

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": 1,
    "display_name": "新昵称",
    "avatar_url": "https://example.com/avatar.png",
    "updated_at": "2026-07-27T12:00:00Z"
  }
}
```

---

### 3. 资源模块

#### 3.1 资源列表

```
GET /api/v1/resources
```

**需要认证：是**

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20，最大 100 |
| keyword | string | 否 | 搜索关键词（匹配标题和描述） |
| category_id | int | 否 | 按分类筛选 |
| type | string | 否 | 按类型筛选：course / article / video |
| sort | string | 否 | 排序方式：latest（默认）/ popular / rating |
| tags | string | 否 | 按标签筛选，逗号分隔 |

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": 1,
        "title": "机器学习入门",
        "description": "面向零基础学习者的机器学习课程",
        "cover_url": "https://example.com/cover.jpg",
        "type": "course",
        "category": {
          "id": 1,
          "name": "人工智能"
        },
        "tags": ["AI", "入门", "Python"],
        "author": "吴恩达",
        "avg_rating": 4.5,
        "view_count": 1024,
        "created_at": "2026-07-01T08:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 3.2 资源详情

```
GET /api/v1/resources/:id
```

**需要认证：是**

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": 1,
    "title": "机器学习入门",
    "description": "面向零基础学习者的机器学习课程",
    "cover_url": "https://example.com/cover.jpg",
    "type": "course",
    "category": {
      "id": 1,
      "name": "人工智能"
    },
    "tags": ["AI", "入门", "Python"],
    "metadata": {
      "duration": "12 小时",
      "chapters": 24,
      "language": "中文"
    },
    "author": "吴恩达",
    "source_url": "https://example.com/course/ml-intro",
    "avg_rating": 4.5,
    "view_count": 1024,
    "created_at": "2026-07-01T08:00:00Z",
    "updated_at": "2026-07-15T10:00:00Z"
  }
}
```

---

#### 3.3 创建资源（管理员）

```
POST /api/v1/resources
```

**需要认证：是（管理员）**

**Request Body**

```json
{
  "title": "string，必填",
  "description": "string，必填",
  "cover_url": "string，选填",
  "type": "string，必填，enum: course / article / video",
  "category_id": "int，必填",
  "tags": ["string，选填"],
  "metadata": "object，选填（扩展字段）",
  "author": "string，选填",
  "source_url": "string，选填"
}
```

**Response (201)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": 2,
    "title": "深度学习实战",
    "type": "course",
    "created_at": "2026-07-27T14:00:00Z"
  }
}
```

---

#### 3.4 编辑资源（管理员）

```
PUT /api/v1/resources/:id
```

**需要认证：是（管理员）**

**Request Body**

```json
{
  "title": "string，选填",
  "description": "string，选填",
  "cover_url": "string，选填",
  "type": "string，选填",
  "category_id": "int，选填",
  "tags": ["string，选填"],
  "metadata": "object，选填",
  "author": "string，选填",
  "source_url": "string，选填"
}
```

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": 2,
    "updated_at": "2026-07-27T15:00:00Z"
  }
}
```

---

#### 3.5 删除资源（管理员）

```
DELETE /api/v1/resources/:id
```

**需要认证：是（管理员）**

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": null
}
```

---

### 4. 分类模块

#### 4.1 分类列表

```
GET /api/v1/categories
```

**需要认证：是**

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": [
    { "id": 1, "name": "人工智能", "description": "AI、机器学习、深度学习" },
    { "id": 2, "name": "前端开发", "description": "HTML、CSS、JavaScript" },
    { "id": 3, "name": "后端开发", "description": "Go、Python、Java" }
  ]
}
```

---

#### 4.2 创建分类（管理员）

```
POST /api/v1/categories
```

**需要认证：是（管理员）**

**Request Body**

```json
{
  "name": "string，必填，最大 64 字符",
  "description": "string，选填，最大 256 字符"
}
```

**Response (201)**

```json
{
  "code": 0,
  "message": "ok",
  "data": { "id": 4, "name": "数据科学", "description": "" }
}
```

---

### 5. 评分评论模块

#### 5.1 资源评分评论列表

```
GET /api/v1/resources/:id/ratings
```

**需要认证：是**

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 默认 1 |
| page_size | int | 否 | 默认 20 |

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": 1,
        "user": {
          "id": 1,
          "username": "zhangsan",
          "display_name": "张三",
          "avatar_url": null
        },
        "score": 5,
        "comment": "非常好的入门课程，讲解清晰",
        "created_at": "2026-07-20T10:00:00Z"
      }
    ],
    "total": 15,
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 5.2 提交评分 / 评论

```
POST /api/v1/resources/:id/ratings
```

**需要认证：是**

**Request Body**

```json
{
  "score": "int，必填，1-5",
  "comment": "string，选填"
}
```

**Response (201)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": 16,
    "score": 5,
    "comment": "非常有帮助",
    "created_at": "2026-07-27T16:00:00Z"
  }
}
```

> 同一用户对同一资源只能有一条评分记录，重复提交视为更新原有评分。

---

### 6. 推荐模块

#### 6.1 获取个性化推荐

```
GET /api/v1/recommendations
```

**需要认证：是**

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | int | 否 | 返回数量，默认 20，最大 50 |

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": 5,
        "title": "Python 数据分析",
        "cover_url": "https://example.com/cover2.jpg",
        "type": "course",
        "category": { "id": 1, "name": "人工智能" },
        "tags": ["Python", "数据分析"],
        "avg_rating": 4.7,
        "view_count": 2048,
        "author": "李老师",
        "created_at": "2026-06-15T08:00:00Z"
      }
    ],
    "updated_at": "2026-07-27T12:00:00Z"
  }
}
```

> 后端先查缓存，缓存未命中则调用 edurec-engine 获取推荐，结果写入缓存。

---

### 7. 用户行为模块

#### 7.1 上报用户行为

```
POST /api/v1/resources/:id/behaviors
```

**需要认证：是**

**Request Body**

```json
{
  "action": "string，必填，enum: view / click / favorite"
}
```

**Response (201)**

```json
{
  "code": 0,
  "message": "ok",
  "data": null
}
```

> `view` 行为由前端在进入资源详情页时自动上报；`click` 和 `favorite` 由对应按钮触发。

---

#### 7.2 我的行为历史

```
GET /api/v1/users/me/behaviors
```

**需要认证：是**

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| action | string | 否 | 按行为类型筛选：view / click / favorite |
| page | int | 否 | 默认 1 |
| page_size | int | 否 | 默认 20 |

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": 1,
        "resource": {
          "id": 5,
          "title": "Python 数据分析",
          "cover_url": "https://example.com/cover2.jpg",
          "type": "course"
        },
        "action": "view",
        "created_at": "2026-07-27T15:30:00Z"
      }
    ],
    "total": 42,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 8. 管理模块

#### 8.1 用户列表（管理员）

```
GET /api/v1/admin/users
```

**需要认证：是（管理员）**

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 按用户名 / 邮箱搜索 |
| page | int | 否 | 默认 1 |
| page_size | int | 否 | 默认 20 |

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": 1,
        "username": "zhangsan",
        "email": "zhangsan@example.com",
        "display_name": "张三",
        "is_admin": true,
        "created_at": "2026-07-01T08:00:00Z"
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

---

#### 8.2 资源管理列表（管理员）

```
GET /api/v1/admin/resources
```

**需要认证：是（管理员）**

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 按标题搜索 |
| type | string | 否 | 按类型筛选 |
| category_id | int | 否 | 按分类筛选 |
| page | int | 否 | 默认 1 |
| page_size | int | 否 | 默认 20 |

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": 1,
        "title": "机器学习入门",
        "type": "course",
        "category": { "id": 1, "name": "人工智能" },
        "avg_rating": 4.5,
        "view_count": 1024,
        "created_at": "2026-07-01T08:00:00Z",
        "updated_at": "2026-07-15T10:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

---

## 附录：接口总览

| # | 方法 | 路径 | 认证 | 管理员 | 说明 |
|---|------|------|------|--------|------|
| 1 | POST | `/api/v1/auth/register` | 否 | — | 注册 |
| 2 | POST | `/api/v1/auth/login` | 否 | — | 登录 |
| 3 | POST | `/api/v1/auth/refresh` | 否 | — | 刷新 token |
| 4 | GET | `/api/v1/users/me` | 是 | — | 当前用户信息 |
| 5 | PUT | `/api/v1/users/me` | 是 | — | 更新个人信息 |
| 6 | GET | `/api/v1/resources` | 是 | — | 资源列表（分页/筛选） |
| 7 | GET | `/api/v1/resources/:id` | 是 | — | 资源详情 |
| 8 | POST | `/api/v1/resources` | 是 | 是 | 创建资源 |
| 9 | PUT | `/api/v1/resources/:id` | 是 | 是 | 编辑资源 |
| 10 | DELETE | `/api/v1/resources/:id` | 是 | 是 | 删除资源 |
| 11 | GET | `/api/v1/categories` | 是 | — | 分类列表 |
| 12 | POST | `/api/v1/categories` | 是 | 是 | 创建分类 |
| 13 | GET | `/api/v1/resources/:id/ratings` | 是 | — | 评分评论列表 |
| 14 | POST | `/api/v1/resources/:id/ratings` | 是 | — | 提交评分/评论 |
| 15 | GET | `/api/v1/recommendations` | 是 | — | 个性化推荐 |
| 16 | POST | `/api/v1/resources/:id/behaviors` | 是 | — | 上报行为 |
| 17 | GET | `/api/v1/users/me/behaviors` | 是 | — | 行为历史 |
| 18 | GET | `/api/v1/admin/users` | 是 | 是 | 用户管理 |
| 19 | GET | `/api/v1/admin/resources` | 是 | 是 | 资源管理 |
