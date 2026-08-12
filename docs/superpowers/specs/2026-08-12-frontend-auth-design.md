# edurec-platform 前端认证页面设计

> 日期：2026-08-12 | 状态：已确认
> 前置：`2026-08-12-frontend-scaffolding-design.md`（脚手架，已合入 main）
> 关联：`docs/api-design.md`（认证接口规范）、`docs/design.md`（页面设计）

## 1. 目标

实现登录 / 注册两个页面，打通认证闭环：登录后跳转（携带 redirect 回跳）、注册后跳转登录页。使用已建立的主题 tokens 与 auth store、API 层、MSW mock。

## 2. 范围

- 登录页 `/login`
- 注册页 `/register`
- 独立认证布局（居中卡片，无顶导航）
- 路由调整：login/register 从 FrontLayout 子路由改为独立顶层路由
- 表单校验（对齐后端约束）+ 交互反馈（ElMessage）

## 3. 设计决策

| 决策 | 选择 | 理由 |
|---|---|---|
| 页面布局 | 独立居中卡片（AuthShell） | 聚焦、品牌感，符合真实产品惯例 |
| 注册后行为 | 跳转登录页 + 提示"注册成功，请登录" | 后端 register 不返回 token |
| 登录后行为 | 优先跳 `redirect` query，无则回首页 | 守卫已携带 redirect 参数 |
| 认证守卫 | `guestOnly`（已登录访问 /login 或 /register 重定向首页） | 脚手架已实现 |

## 4. 组件设计

### 4.1 `components/auth/AuthShell.vue`（认证布局外壳）

- 全屏浅色渐变背景（`bg-bg` → 主色淡色），垂直水平居中
- 顶部品牌：E logo + "edurec" 标题 + 副标题
- 白色卡片（`bg-surface`、`rounded-lg`、`shadow`、`border-border`），max-w-md
- 卡片内 `<slot />` 放表单
- 底部：版权信息

### 4.2 登录页 `/login`（LoginPage.vue）

- 表单字段：username（支持用户名或邮箱）、password
- 校验规则：
  - username：必填
  - password：必填
- 提交：
  1. `auth.login(username, password)`
  2. 成功 → ElMessage.success("登录成功") → 跳转 `route.query.redirect`（无则 `home`）
  3. 失败 → ElMessage.error(错误信息)，loading 结束
- loading 态：按钮 loading，防重复提交
- 底部链接：没有账号？去注册

### 4.3 注册页 `/register`（RegisterPage.vue）

- 表单字段：username、email、password、confirmPassword、display_name（选填）
- 校验规则（对齐后端约束）：
  - username：必填，3-64 字符
  - email：必填，合法邮箱格式
  - password：必填，6-128 字符
  - confirmPassword：必填，与 password 一致
  - display_name：选填
- 提交：
  1. `authApi.register({ username, email, password, display_name })`
  2. 成功 → ElMessage.success("注册成功，请登录") → 跳转 `login`
  3. 失败 → ElMessage.error(错误信息)
- 底部链接：已有账号？去登录

### 4.4 路由调整（router/index.ts）

- 从 FrontLayout 的 children 中移除 `login`、`register`
- 新增顶层路由：

```ts
{ path: '/login', name: 'login', component: LoginPage, meta: { guestOnly: true } },
{ path: '/register', name: 'register', component: RegisterPage, meta: { guestOnly: true } },
```

- 守卫逻辑不变（`guestOnly` 已覆盖）

## 5. 测试策略

- **LoginPage 组件测试**（Vue Test Utils，mock auth store）：
  - 校验失败不提交
  - 登录成功 → store.login 调用 + 跳转（redirect query 优先）
  - 登录失败 → ElMessage 报错
- **RegisterPage 组件测试**：
  - 密码与确认密码不一致 → 校验失败
  - 注册成功 → 跳转 login + 提示
- 认证守卫 `guestOnly` 已有单测覆盖（脚手架 Task 5）

## 6. 交付物与验证

- AuthShell + LoginPage + RegisterPage + 路由调整
- `pnpm type-check`、`pnpm test`、`pnpm build` 全绿
- `pnpm dev` 手动验证：访问 `/login` 渲染独立卡片；`admin/admin123` 登录成功跳首页；`/register` 注册成功后跳 `/login` 并提示
