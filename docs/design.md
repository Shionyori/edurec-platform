# edurec-platform 设计文档

> 最终更新：2026-07-27

## 1. 项目概述

edurec-platform 是一个教育资源推荐平台，核心目标：

1. **应用场景** — 作为 [edurec-engine](https://github.com/) 的具体应用场景，用于测试和演示推荐模型的效果
2. **完整实践** — 从头到尾实现完整的前后端项目，覆盖主流技术栈和功能模块，便于学习和参考

## 2. 仓库与分支管理

### 2.1 仓库结构

```
edurec-platform/
  frontend/          # 前端项目
  backend/           # 后端项目
  api/               # API 设计文档
  docs/              # 其他文档
  docker/            # Docker 相关配置文件
  docker-compose.yml # 编排文件
```

- frontend 和 backend 同仓库管理，与 edurec-engine 并列放于同一上级目录，便于展示项目间关系
- 不拆分成独立仓库，避免管理负担

### 2.2 分支与提交规范

- **分支命名**：按模块开发，如 `feat/frontend-user-auth`、`feat/backend-recommendation`
- **提交信息**：遵循 Conventional Commits，使用 scope 区分模块

```
feat(frontend): 实现用户登录页面
feat(backend): 实现 JWT 认证中间件
fix(backend): 修复 token 过期判断
chore: 更新 docker-compose 配置
```

## 3. 技术栈

| 层 | 技术 | 说明 |
|---|---|---|
| **前端** | TypeScript + Vue 3 | 主流前端框架，TypeScript 保证类型安全 |
| **后端** | Go + Gin | 高性能 HTTP 框架 |
| **数据库** | MySQL | 主数据存储 |
| **缓存** | Redis | JWT refresh token 存储 + 推荐结果缓存 |
| **容器化** | Docker + Docker Compose | 统一部署编排 |

### 3.1 前端技术选型

| 用途 | 选择 | 说明 |
|---|---|---|
| 框架 | Vue 3 (Composition API) | |
| 语言 | TypeScript | |
| 构建工具 | Vite | Vue 3 官方推荐 |
| 状态管理 | **Pinia** | Vue 3 官方推荐，TypeScript 友好 |
| 路由 | Vue Router 4 | |
| UI 组件库 | **Element Plus** | Vue 3 最主流组件库，组件齐全 |
| CSS | **Tailwind CSS** | 处理自定义布局和页面级样式 |
| HTTP 客户端 | **Axios** | 拦截器链适合 JWT 自动附加和错误统一处理 |
| Mock | **MSW** | 浏览器级别请求拦截，开发体验接近真实联调 |
| 测试 | **Vitest + Vue Test Utils** | |
| 包管理 | pnpm | |

### 3.2 后端技术选型

| 用途 | 选择 | 说明 |
|---|---|---|
| 语言 | Go 1.21+ | |
| HTTP 框架 | **Gin** | 高性能，中间件生态丰富 |
| ORM | **GORM** | Go 生态 ORM 事实标准，跨语言 ORM 概念一致 |
| 迁移工具 | **golang-migrate** | 独立 SQL up/down 脚本，版本化管理 |
| 配置管理 | **Viper + 环境变量** | YAML 文件 + 环境变量覆盖，适配 Docker 部署 |
| 认证 | **JWT (Access + Refresh Token)** | Refresh token 存 Redis |
| 日志 | **log/slog** | Go 1.21+ 标准库，结构化日志，零依赖 |
| 测试 | **testify + httptest** | 单元测试 + API 集成测试 |

## 4. 架构设计

### 4.1 系统架构

```
┌──────────┐    HTTP/REST    ┌──────────────┐    HTTP/REST    ┌──────────────┐
│  Frontend │ ◄────────────► │   Backend    │ ◄────────────► │ edurec-engine│
│  (Vue 3)  │                │  (Go + Gin)  │                │  (独立服务)   │
└──────────┘                └──────┬───────┘                └──────────────┘
                                   │
                            ┌──────┴───────┐
                            │  MySQL  Redis │
                            └──────────────┘
```

- **Frontend ↔ Backend**：REST API (HTTP + JSON)，JWT 认证
- **Backend ↔ edurec-engine**：独立微服务，通过 REST 调用（engine 具体设计待定，platform 侧定义 engine client 接口即可）
- 所有接口遵循统一响应格式和错误码规范

### 4.2 后端分层架构

采用 **Handler → Service → Repository** 扁平三层：

```
cmd/server/main.go          # 应用入口，初始化依赖
internal/
  handler/                   # HTTP handler — 参数校验、序列化、调用 service
  service/                   # 业务逻辑层
  repository/                # 数据访问层（GORM 操作）
  model/                     # 数据模型 struct 定义
  middleware/                 # 认证、错误处理、CORS 等中间件
migrations/                  # golang-migrate SQL 脚本
configs/                     # YAML 配置模板
```

单向依赖：`handler → service → repository → model`，下层不依赖上层。

### 4.3 前端目录结构

混合组织——公共资源按技术类型放，业务页面按模块放：

```
frontend/
  src/
    api/                     # API client 实例 + Axios 拦截器配置
    components/              # 公共 UI 组件
      common/                # 通用组件（按钮、卡片等）
      layout/                # 布局组件（前台布局、后台布局）
    composables/             # 公共组合式函数
    mocks/                   # MSW handlers
    pages/                   # 页面（按功能模块组织）
      home/                  # 首页
      auth/                  # 登录/注册
      resource/              # 资源详情、搜索
      user/                  # 个人中心
      upload/                # 资源上传（管理员）
      admin/                 # 管理后台
    router/                  # 路由配置 + 路由守卫
    stores/                  # Pinia stores
    types/                   # TypeScript 类型定义
    utils/                   # 工具函数
```

### 4.4 API 设计规范

- **CRUD 操作**：使用 RESTful 风格（`GET /resources/:id`, `POST /resources`）
- **复杂操作**：使用 RPC 风格（`POST /auth/login`, `GET /recommendations`）
- **URL 前缀**：`/api/v1/`
- **统一响应格式**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

- **错误处理**：HTTP 状态码（2xx/4xx/5xx）+ 业务错误码 + Gin middleware 统一封装
- 详细接口定义见 `api/api-design.md`

## 5. 数据模型

### 5.1 实体关系

```
users ──1:1── admins
users ──1:N── user_behaviors
users ──1:N── ratings
resources ──1:N── ratings
resources ──1:N── user_behaviors
resources ──N:1── categories
users ──1:N── recommendations
```

### 5.2 数据表设计

#### users（用户）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INT UNSIGNED AUTO_INCREMENT PK | |
| username | VARCHAR(64) UNIQUE NOT NULL | |
| email | VARCHAR(128) UNIQUE NOT NULL | |
| password_hash | VARCHAR(256) NOT NULL | |
| display_name | VARCHAR(128) | 昵称 |
| avatar_url | VARCHAR(512) | |
| created_at | DATETIME | |
| updated_at | DATETIME | |

#### admins（管理员）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INT UNSIGNED AUTO_INCREMENT PK | |
| user_id | INT UNSIGNED FK → users.id | |

#### resources（教育资源，统一抽象）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INT UNSIGNED AUTO_INCREMENT PK | |
| title | VARCHAR(256) NOT NULL | |
| description | TEXT | |
| cover_url | VARCHAR(512) | 封面图 |
| type | ENUM('course', 'article', 'video') NOT NULL | 资源类型 |
| category_id | INT UNSIGNED FK → categories.id | |
| tags | JSON | 标签数组，例 `["AI", "入门"]` |
| metadata | JSON | 扩展字段（视频时长、文章字数等） |
| author | VARCHAR(128) | 作者/来源 |
| source_url | VARCHAR(512) | 原始链接 |
| avg_rating | DECIMAL(2,1) DEFAULT 0 | 平均评分 |
| view_count | INT UNSIGNED DEFAULT 0 | 浏览数 |
| created_at | DATETIME | |
| updated_at | DATETIME | |

#### categories（分类）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INT UNSIGNED AUTO_INCREMENT PK | |
| name | VARCHAR(64) NOT NULL | |
| description | VARCHAR(256) | |
| created_at | DATETIME | |

#### user_behaviors（用户行为）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGINT UNSIGNED AUTO_INCREMENT PK | |
| user_id | INT UNSIGNED FK → users.id | |
| resource_id | INT UNSIGNED FK → resources.id | |
| action | ENUM('view', 'click', 'favorite') NOT NULL | |
| created_at | DATETIME | |

#### ratings（评分评论）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INT UNSIGNED AUTO_INCREMENT PK | |
| user_id | INT UNSIGNED FK → users.id | |
| resource_id | INT UNSIGNED FK → resources.id | |
| score | TINYINT UNSIGNED NOT NULL | 1-5 评分 |
| comment | TEXT | 评论内容 |
| created_at | DATETIME | |
| updated_at | DATETIME | |

#### recommendations（推荐结果缓存）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INT UNSIGNED AUTO_INCREMENT PK | |
| user_id | INT UNSIGNED FK → users.id | |
| resource_ids | JSON | 推荐资源 ID 列表 |
| created_at | DATETIME | 生成时间 |

## 6. 页面设计

| 页面 | 路由 | 说明 | 认证要求 |
|------|------|------|---------|
| 首页 | `/` | 推荐流、热门资源、分类入口 | 是 |
| 登录 | `/login` | 用户登录 | 否 |
| 注册 | `/register` | 用户注册 | 否 |
| 资源详情 | `/resources/:id` | 内容展示、评分评论 | 是 |
| 资源搜索 | `/search` | 关键词 + 分类/标签筛选 | 是 |
| 个人中心 | `/user/me` | 个人信息、行为历史、收藏 | 是 |
| 资源上传 | `/resources/upload` | 管理员上传资源 | 管理员 |
| 管理后台 | `/admin` | 用户管理、资源审核、分类管理 | 管理员 |

### 6.1 布局方案

- **前台布局**（`/`, `/resources/*`, `/search`, `/user/*`）：顶部导航 + 内容区 — 参考 Coursera、Bilibili 课堂的沉浸式浏览体验
- **后台布局**（`/resources/upload`, `/admin/*`）：侧边栏 + 顶栏 — 参考 Element Plus Admin 的高效管理操作
- 路由 `/admin/*` 前缀自动切换后台布局

## 7. 认证与安全

- 认证协议：JWT Access Token + Refresh Token
- Access Token 有效期：15 分钟
- Refresh Token：存 Redis，有效期 7 天
- Access Token 过期后，客户端用 refresh token 换新 access token（`POST /api/v1/auth/refresh`）
- 路由守卫：未登录自动重定向到 `/login`
- 管理员权限：检查 `admins` 表中是否存在对应用户

## 8. 部署方案

### 8.1 Docker 多阶段构建

- **Frontend**：构建阶段（Node.js + `pnpm build`） → 运行阶段（Nginx 静态托管）
- **Backend**：构建阶段（Go build） → 运行阶段（Alpine + 二进制文件）
- **MySQL / Redis**：使用官方镜像

### 8.2 编排

`docker-compose.yml` 统一编排所有服务：

```
services:
  frontend   — Nginx，端口 80
  backend    — Go server，端口 8080
  mysql      — 端口 3306
  redis      — 端口 6379
```

- Backend 通过 Viper 读 YAML 默认配置，敏感信息通过 compose 环境变量注入

## 9. 测试策略

| 层级 | 工具 | 覆盖范围 |
|------|------|---------|
| 后端单元测试 | testify | Service 层核心逻辑 |
| 后端集成测试 | httptest + testify | API handler 完整链路（含认证、数据库） |
| 前端组件测试 | Vitest + Vue Test Utils | 核心交互流程和关键组件 |

- 实用主义策略：覆盖关键路径（认证、推荐、CRUD），不追求覆盖率数字

## 10. 开发流程

1. 写 API 设计文档（`api/api-design.md`）
2. 前后端各自对照文档实现
3. 分支开发：`feat/frontend-xxx` / `feat/backend-xxx`
4. 完成后合入 main
5. 联调测试

---

## 决策记录

| # | 决策项 | 选择 |
|---|--------|------|
| 1 | 与 edurec-engine 集成方式 | 独立微服务 |
| 2 | 与 engine 通信协议 | REST (HTTP + JSON) |
| 3 | engine 具体设计 | 暂后置，platform 侧定义 engine client 接口 |
| 4 | 后端分层架构 | Handler → Service → Repository |
| 5 | 认证方案 | JWT Access + Refresh Token |
| 6 | API 设计规范 | 混合（CRUD RESTful + 操作 RPC），统一规范 |
| 7 | 状态管理 | Pinia |
| 8 | 数据库 ORM | GORM |
| 9 | 数据库迁移 | golang-migrate |
| 10 | API 文档 | 手写设计文档先行 |
| 11 | UI 组件库 | Element Plus |
| 12 | Mock 方案 | MSW |
| 13 | 分支提交规范 | feat(frontend): / feat(backend): |
| 14 | Docker 部署 | 独立 Dockerfile + 多阶段构建 |
| 15 | 配置管理 | Viper + 环境变量覆盖 |
| 16 | 前端路由守卫 | Vue Router，未登录 → /login |
| 17 | 前台布局 | 顶部导航 + 内容区（参考 Coursera） |
| 18 | 后台布局 | 侧边栏 + 顶栏（参考 Element Plus Admin） |
| 19 | 后端日志 | Go 标准库 log/slog |
| 20 | 错误处理 | HTTP 状态码 + 业务错误码 + middleware |
| 21 | 测试策略 | testify + httptest（后端），Vitest + Vue Test Utils（前端） |
| 22 | 前端目录 | 混合（公共按类型，页面按模块） |
| 23 | 后端目录 | cmd/internal/migrations/configs |
| 24 | 资源建模 | 统一抽象 + JSON metadata，type 字段区分 |
| 25 | 管理员 | 独立 admins 表（id, user_id） |
| 26 | 前端 HTTP 客户端 | Axios |
| 27 | CSS 方案 | Tailwind CSS + Element Plus |
