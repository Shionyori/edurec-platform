# edurec-platform 前端脚手架 + 基础设施设计

> 日期：2026-08-12 | 状态：已确认
> 关联文档：`docs/design.md`（项目设计）、`docs/api-design.md`（API 规范）

## 1. 背景与目标

后端（Go + Gin）已完成认证、用户、分类、资源、行为、评分、管理 7 个模块共 19 个接口，前端目录为空。本子项目搭建前端工程地基，使后续页面子项目可以独立、快速开发。

**目标：** `pnpm dev` 可启动，首页能通过 MSW mock 加载出符合后端接口结构的真实数据；布局骨架（前台/后台）就位；认证与路由体系可跑通。

## 2. 范围

### 2.1 本子项目包含

- Vite + Vue 3 + TypeScript 工程初始化
- Pinia + Vue Router 4 接入
- Element Plus（按需导入）+ Tailwind CSS 接入
- 设计 tokens（配色、字体、圆角、间距、阴影）
- Axios 封装（拦截器链、统一响应解包、401 刷新/跳转）
- MSW mock（覆盖全部 19 个接口 + 种子数据）
- FrontLayout / AdminLayout 布局骨架
- 路由守卫（未登录 → /login；/admin/* 校验管理员）
- auth store（token 持久化、login/register/logout/refresh）
- Vitest 测试环境
- ESLint + Prettier

### 2.2 不包含（后续子项目）

- 登录/注册页面（子项目 2）
- 首页、资源列表/搜索、资源详情（子项目 3）
- 个人中心（子项目 4）
- 资源上传、管理后台（子项目 5）

## 3. 技术栈

| 用途 | 选择 | 说明 |
|---|---|---|
| 框架 | Vue 3 (Composition API) | |
| 语言 | TypeScript (strict) | |
| 构建 | Vite | |
| 状态管理 | Pinia | |
| 路由 | Vue Router 4 | |
| UI 组件库 | Element Plus | 按需导入 |
| 样式 | Tailwind CSS | 自定义布局与页面级样式 |
| HTTP | Axios | 拦截器链 |
| Mock | MSW | 浏览器层请求拦截 |
| 测试 | Vitest + Vue Test Utils | |
| 规范 | ESLint + Prettier | |
| 包管理 | pnpm | |

## 4. 目录结构

```
frontend/
  index.html
  vite.config.ts
  tsconfig.json
  package.json
  src/
    api/                  # API 层
      client.ts           # axios 实例 + 拦截器
      auth.ts             # 认证相关 API 封装
      resource.ts         # 资源相关 API 封装
      category.ts         # 分类 API 封装
      rating.ts           # 评分 API 封装
      behavior.ts         # 行为 API 封装
      user.ts             # 用户 API 封装
      admin.ts            # 管理 API 封装
      types.ts            # 响应类型（code/message/data）
    components/
      layout/
        FrontLayout.vue   # 前台布局（顶导航 + 内容区）
        AdminLayout.vue   # 后台布局（侧边栏 + 顶栏）
    mocks/
      browser.ts          # MSW worker 启动
      handlers.ts         # 全部接口 mock handlers
      db.ts               # mock 内存数据库 + 种子数据
      types.ts            # mock 数据模型
    router/
      index.ts            # 路由表
      guards.ts           # 路由守卫
    stores/
      auth.ts             # 认证 store
    styles/
      index.css           # Tailwind 入口
      theme.ts            # 设计 tokens 常量
      element.scss        # Element Plus 变量覆盖
    types/
      index.ts            # 全局 TS 类型
    pages/
      home/
        index.vue         # 首页占位（可显示 mock 数据）
      auth/
        LoginPage.vue     # 登录占位（子项目 2 实现）
        RegisterPage.vue  # 注册占位
      not-found/
        NotFound.vue      # 404 占位
    main.ts
    App.vue
    env.d.ts
  .eslintrc.js
  .prettierrc
```

## 5. 视觉基础（frontend-design skill 指导）

- 建立一套设计 tokens，注入 Tailwind theme 与 Element Plus CSS 变量：
  - **主色**：面向教育资源的沉稳、专业配色
  - **中性色**：文字层级（primary/secondary/muted）、边框、背景
  - **语义色**：成功/警告/危险/信息
  - **字体**：中文无衬线栈（system-ui + PingFang/HarmonyOS/Microsoft YaHei）
  - **圆角/间距/阴影**：统一刻度
- 前台布局参考 Coursera/Bilibili 课堂的沉浸式浏览；后台布局参考 Element Plus Admin 的侧边栏管理
- 本子项目确定 tokens 与两套布局骨架，页面级视觉在子项目 2-5 落实

## 6. 基础设施设计

### 6.1 Axios 封装（`src/api/client.ts`）

- `baseURL = '/api/v1'`
- **请求拦截器**：存在 access_token 时自动附 `Authorization: Bearer <token>`
- **响应拦截器**：
  - 解包统一响应 `{ code, message, data }`，直接返回 `data`
  - `code !== 0` 时抛出业务错误（含 message）
  - HTTP 401 → 尝试 refresh（`POST /auth/refresh`），成功后重放原请求；refresh 失败则清 token 跳登录
  - 网络/5xx 错误统一提示

### 6.2 MSW（`src/mocks/`）

- `VITE_MOCK=1` 环境开关，默认开启
- `db.ts`：内存 mock 数据库 + 种子数据（用户、分类、资源、评分、行为、管理员）
- `handlers.ts`：覆盖 API 文档全部 19 个接口，返回结构与后端统一响应格式完全一致
- 关闭 mock 时无缝切换真实后端（只改环境变量）

### 6.3 路由与守卫

| 路径 | 页面 | 守卫 |
|---|---|---|
| `/` | 首页（前台布局） | 需登录 |
| `/login` `/register` | 认证 | 公开 |
| `/resources/:id` | 资源详情（子项目 3） | 需登录 |
| `/search` | 搜索（子项目 3） | 需登录 |
| `/user/me` | 个人中心（子项目 4） | 需登录 |
| `/resources/upload` `/admin/*` | 管理（子项目 5） | 需登录 + 管理员 |

- **认证守卫**：未登录访问受保护路由 → 重定向 `/login`，携带 `redirect` 参数，登录后回跳
- **管理员守卫**：`/admin/*` 校验 `user.is_admin`，非管理员 → 403 页/提示

### 6.4 Pinia auth store（`src/stores/auth.ts`）

- state：`accessToken`、`refreshToken`、`user`、`loading`
- getters：`isLoggedIn`、`isAdmin`
- actions：`login`、`register`、`logout`、`refresh`、`fetchMe`
- token 持久化到 localStorage，初始化时恢复；`fetchMe` 在应用启动时拉取用户信息（含 `is_admin`）

## 7. 布局组件

### 7.1 FrontLayout（前台）

- 顶部导航：Logo、搜索框、分类入口、用户菜单（头像/昵称下拉：个人中心/退出）
- 内容区：`<router-view>`
- 沉浸式浏览体验（参考 Coursera）

### 7.2 AdminLayout（后台）

- 侧边栏：资源管理 / 用户管理 / 分类管理 导航（Element Plus el-menu）
- 顶栏：面包屑 + 返回前台入口 + 用户菜单
- 内容区：`<router-view>`

### 7.3 路由挂载

- 首页 `/`、认证页、404 用 FrontLayout
- `/admin/*`、上传页用 AdminLayout
- 布局切换通过路由 meta + 嵌套路由实现

## 8. 测试

- Vitest + Vue Test Utils 环境就绪
- 本子项目覆盖：auth store 的 login/logout/token 持久化逻辑
- 关键页面交互测试在子项目 2-5 补充

## 9. 交付物与验证

本子项目完成时（一次 commit）：

1. 全部脚手架文件
2. FrontLayout / AdminLayout 骨架
3. MSW mock（全 19 接口 + 种子数据）
4. auth store + 路由守卫
5. 首页占位页能加载 mock 数据
6. `pnpm dev` 启动无报错，`pnpm build`（类型检查 + 构建）通过
7. Vitest 测试通过

## 10. 后续子项目规划

| 序号 | 子项目 | 页面 |
|---|---|---|
| 2 | 认证页面 | 登录 / 注册 |
| 3 | 前台核心页面 | 首页（推荐流）、资源列表/搜索、资源详情（评分评论） |
| 4 | 个人中心 | 个人信息、行为历史 |
| 5 | 管理后台 | 资源上传、用户管理、资源管理、分类管理 |
