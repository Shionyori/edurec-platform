# 前端认证页面实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现登录 `/login`、注册 `/register` 两个页面，打通认证闭环（登录跳转 redirect、注册跳转登录页）。

**Architecture:** 独立居中卡片布局（AuthShell），登录/注册页从 FrontLayout 子路由改为顶层路由。页面通过 auth store / API 层提交，Element Plus 表单校验 + ElMessage 反馈。

**Tech Stack:** Vue 3 / Vue Router 5 / Pinia / Element Plus / Vitest + Vue Test Utils

## Global Constraints

- **提交规则（CLAUDE.md）**：每个 Task 结束 commit 一次，commit 后必须停下等待用户确认。
- **分支**：所有工作都在 `feat/frontend-auth` 分支上进行。
- **提交信息格式**：`<type>(frontend): <subject>`。
- **已存在**（脚手架产出，合入 main）：auth store（`useAuthStore`）、API 层（`@/api/auth` 的 `login/register`）、守卫（`resolveGuard` + `guestOnly`）、主题 tokens、MSW mock。
- **登录页行为**：成功后优先跳 `route.query.redirect`，无则跳 `{ name: 'home' }`。
- **注册页行为**：成功后 ElMessage 提示"注册成功，请登录"，跳 `{ name: 'login' }`。

---

### Task 1: 认证布局外壳（AuthShell）与路由调整

**Files:**
- Create: `frontend/src/components/auth/AuthShell.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/pages/auth/LoginPage.vue`（包 AuthShell，保留占位内容）
- Modify: `frontend/src/pages/auth/RegisterPage.vue`（包 AuthShell，保留占位内容）
- Test: `frontend/src/components/auth/__tests__/AuthShell.test.ts`

**Interfaces:**
- Consumes: `src/styles/theme.ts` 的 `themeTokens`（Task 1，脚手架）
- Produces: `AuthShell` 默认导出 Vue 组件（居中卡片外壳，含品牌 + slot + 版权）

- [ ] **Step 1: 写失败测试**

`frontend/src/components/auth/__tests__/AuthShell.test.ts`：

```ts
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AuthShell from '../AuthShell.vue'

vi.mock('vue-router', () => ({
  RouterLink: { template: '<a><slot /></a>' },
}))

describe('AuthShell', () => {
  it('渲染品牌标题与插槽内容', () => {
    const wrapper = mount(AuthShell, {
      slots: { default: '<p class="test-slot">卡片内容</p>' },
    })
    expect(wrapper.text()).toContain('edurec')
    expect(wrapper.find('.test-slot').text()).toBe('卡片内容')
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd frontend
pnpm test src/components/auth/__tests__/AuthShell.test.ts
```

Expected: FAIL，`Cannot find module '../AuthShell.vue'`。

- [ ] **Step 3: 实现 AuthShell**

`frontend/src/components/auth/AuthShell.vue`：

```vue
<script setup lang="ts">
import { themeTokens } from '@/styles/theme'
</script>

<template>
  <div
    class="flex min-h-screen items-center justify-center bg-bg px-4"
    :style="{
      backgroundImage: `radial-gradient(ellipse at top left, ${themeTokens.color.primary}14, transparent 50%), radial-gradient(ellipse at bottom right, ${themeTokens.color.primary}0f, transparent 50%)`,
    }"
  >
    <div class="w-full max-w-md">
      <div class="mb-8 text-center">
        <RouterLink to="/" class="inline-flex items-center gap-2 text-2xl font-bold text-primary">
          <span class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-white">E</span>
          edurec
        </RouterLink>
        <p class="mt-2 text-sm text-ink-secondary">发现优质教育资源</p>
      </div>
      <div class="rounded-xl border border-border bg-surface p-8 shadow-sm">
        <slot />
      </div>
      <p class="mt-6 text-center text-xs text-ink-muted">© 2026 edurec · 教育资源推荐平台</p>
    </div>
  </div>
</template>
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd frontend
pnpm test src/components/auth/__tests__/AuthShell.test.ts
```

Expected: PASS。

- [ ] **Step 5: 页面包 AuthShell**

`frontend/src/pages/auth/LoginPage.vue`：

```vue
<script setup lang="ts">
import AuthShell from '@/components/auth/AuthShell.vue'
</script>

<template>
  <AuthShell>
    <p class="text-sm text-ink-secondary">登录页面将在 Task 2 实现</p>
  </AuthShell>
</template>
```

`frontend/src/pages/auth/RegisterPage.vue`：

```vue
<script setup lang="ts">
import AuthShell from '@/components/auth/AuthShell.vue'
</script>

<template>
  <AuthShell>
    <p class="text-sm text-ink-secondary">注册页面将在 Task 3 实现</p>
  </AuthShell>
</template>
```

- [ ] **Step 6: 路由调整 —— login/register 改为顶层路由**

`frontend/src/router/index.ts`：

在 FrontLayout 的 children 中**删除**这两行：

```ts
{ path: 'login', name: 'login', component: LoginPage, meta: { guestOnly: true } },
{ path: 'register', name: 'register', component: RegisterPage, meta: { guestOnly: true } },
```

在 routes 数组中（`{ path: '/', component: FrontLayout }` 之后）**新增**：

```ts
{
  path: '/login',
  name: 'login',
  component: LoginPage,
  meta: { guestOnly: true },
},
{
  path: '/register',
  name: 'register',
  component: RegisterPage,
  meta: { guestOnly: true },
},
```

- [ ] **Step 7: 验证**

```bash
cd frontend
pnpm type-check
pnpm test
pnpm build
```

Expected: 全部通过（原 17 测试 + AuthShell 1 测试）。

- [ ] **Step 8: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): 实现认证布局外壳与路由调整"
```

---

### Task 2: 登录页实现与测试

**Files:**
- Modify: `frontend/src/pages/auth/LoginPage.vue`
- Test: `frontend/src/pages/auth/__tests__/LoginPage.test.ts`

**Interfaces:**
- Consumes: `useAuthStore`（login）、`useRouter`/`useRoute`、`ElMessage`、`AuthShell`（Task 1）
- Produces: 完整登录页 —— 提交后调 `auth.login`，成功跳 `redirect` 或 `home`，失败 ElMessage 报错

- [ ] **Step 1: 写失败测试**

`frontend/src/pages/auth/__tests__/LoginPage.test.ts`：

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import LoginPage from '../LoginPage.vue'

const pushMock = vi.fn()
const loginMock = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
  useRoute: () => ({ query: { redirect: '/resources/1' } }),
  RouterLink: { template: '<a><slot /></a>' },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ login: loginMock }),
}))

async function submit(wrapper: ReturnType<typeof mount>) {
  await wrapper.find('input[placeholder="用户名或邮箱"]').setValue('admin')
  await wrapper.find('input[placeholder="密码"]').setValue('admin123')
  await wrapper.find('button').trigger('click')
  await flushPromises()
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('登录成功跳转 redirect', async () => {
    loginMock.mockResolvedValue({ id: 1, username: 'admin' })
    const wrapper = mount(LoginPage, { global: { plugins: [ElementPlus] } })
    await submit(wrapper)
    expect(loginMock).toHaveBeenCalledWith('admin', 'admin123')
    expect(pushMock).toHaveBeenCalledWith('/resources/1')
  })

  it('登录失败弹出错误信息', async () => {
    vi.spyOn(ElMessage, 'error').mockImplementation(() => ({}) as never)
    loginMock.mockRejectedValue(new Error('用户名或密码错误'))
    const wrapper = mount(LoginPage, { global: { plugins: [ElementPlus] } })
    await submit(wrapper)
    expect(ElMessage.error).toHaveBeenCalledWith('用户名或密码错误')
  })

  it('表单为空时不提交', async () => {
    const wrapper = mount(LoginPage, { global: { plugins: [ElementPlus] } })
    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(loginMock).not.toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd frontend
pnpm test src/pages/auth/__tests__/LoginPage.test.ts
```

Expected: FAIL（校验失败/未实现）。

- [ ] **Step 3: 实现登录页**

`frontend/src/pages/auth/LoginPage.vue`：

```vue
<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import AuthShell from '@/components/auth/AuthShell.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const formRef = ref<{ validate: () => Promise<void> }>()
const loading = ref(false)
const form = reactive({
  username: '',
  password: '',
})

const rules = {
  username: [{ required: true, message: '请输入用户名或邮箱', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function onSubmit() {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    ElMessage.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : ''
    router.push(redirect || { name: 'home' })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '登录失败，请重试')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthShell>
    <h1 class="text-xl font-bold text-ink">欢迎回来</h1>
    <p class="mt-1 text-sm text-ink-secondary">登录 edurec，继续你的学习之旅</p>

    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="mt-6">
      <el-form-item label="用户名 / 邮箱" prop="username">
        <el-input
          v-model="form.username"
          placeholder="用户名或邮箱"
          size="large"
          autocomplete="username"
        />
      </el-form-item>
      <el-form-item label="密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          placeholder="密码"
          size="large"
          show-password
          autocomplete="current-password"
          @keyup.enter="onSubmit"
        />
      </el-form-item>
      <el-button type="primary" size="large" class="mt-2 w-full" :loading="loading" @click="onSubmit">
        登 录
      </el-button>
    </el-form>

    <p class="mt-6 text-center text-sm text-ink-secondary">
      还没有账号？
      <RouterLink :to="{ name: 'register' }" class="text-primary hover:underline">去注册</RouterLink>
    </p>
  </AuthShell>
</template>
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd frontend
pnpm test src/pages/auth/__tests__/LoginPage.test.ts
```

Expected: PASS。若 `setValue` 未触发 v-model，改用 `wrapper.findComponent({ name: 'ElInput' }).vm.$emit('update:modelValue', value)` 设置输入。

- [ ] **Step 5: 验证**

```bash
cd frontend
pnpm type-check
pnpm test
```

Expected: 全部通过。

- [ ] **Step 6: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): 实现登录页与测试"
```

---

### Task 3: 注册页实现与测试

**Files:**
- Modify: `frontend/src/pages/auth/RegisterPage.vue`
- Test: `frontend/src/pages/auth/__tests__/RegisterPage.test.ts`

**Interfaces:**
- Consumes: `@/api/auth` 的 `register`、`useRouter`、`ElMessage`、`AuthShell`（Task 1）
- Produces: 完整注册页 —— 提交后调 `register`，成功提示并跳 `login`，失败 ElMessage 报错

- [ ] **Step 1: 写失败测试**

`frontend/src/pages/auth/__tests__/RegisterPage.test.ts`：

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import { register } from '@/api/auth'
import RegisterPage from '../RegisterPage.vue'

const pushMock = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
  RouterLink: { template: '<a><slot /></a>' },
}))

vi.mock('@/api/auth', () => ({ register: vi.fn() }))

const mockedRegister = vi.mocked(register)

async function fillAndSubmit(wrapper: ReturnType<typeof mount>) {
  await wrapper.find('input[placeholder="用户名"]').setValue('newuser')
  await wrapper.find('input[placeholder="邮箱"]').setValue('newuser@example.com')
  await wrapper.find('input[placeholder="密码"]').setValue('secret123')
  await wrapper.find('input[placeholder="再次输入密码"]').setValue('secret123')
  await wrapper.find('button').trigger('click')
  await flushPromises()
}

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('注册成功跳转登录页并提示', async () => {
    vi.spyOn(ElMessage, 'success').mockImplementation(() => ({}) as never)
    mockedRegister.mockResolvedValue({
      user: { id: 3, username: 'newuser', email: 'newuser@example.com', display_name: null, avatar_url: null, created_at: '2026-08-12T00:00:00Z' },
    })
    const wrapper = mount(RegisterPage, { global: { plugins: [ElementPlus] } })
    await fillAndSubmit(wrapper)
    expect(mockedRegister).toHaveBeenCalledWith(
      expect.objectContaining({ username: 'newuser', email: 'newuser@example.com', password: 'secret123' }),
    )
    expect(ElMessage.success).toHaveBeenCalledWith('注册成功，请登录')
    expect(pushMock).toHaveBeenCalledWith({ name: 'login' })
  })

  it('两次密码不一致时不提交', async () => {
    const wrapper = mount(RegisterPage, { global: { plugins: [ElementPlus] } })
    await wrapper.find('input[placeholder="用户名"]').setValue('newuser')
    await wrapper.find('input[placeholder="邮箱"]').setValue('newuser@example.com')
    await wrapper.find('input[placeholder="密码"]').setValue('secret123')
    await wrapper.find('input[placeholder="再次输入密码"]').setValue('different')
    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(mockedRegister).not.toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd frontend
pnpm test src/pages/auth/__tests__/RegisterPage.test.ts
```

Expected: FAIL（校验失败/未实现）。

- [ ] **Step 3: 实现注册页**

`frontend/src/pages/auth/RegisterPage.vue`：

```vue
<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { register } from '@/api/auth'
import AuthShell from '@/components/auth/AuthShell.vue'

const router = useRouter()

const formRef = ref<{ validate: () => Promise<void> }>()
const loading = ref(false)
const form = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  display_name: '',
})

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 64, message: '用户名长度 3-64 个字符', trigger: 'blur' },
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 128, message: '密码长度 6-128 个字符', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    {
      validator: (_rule: unknown, value: string, callback: (err?: Error) => void) => {
        if (value !== form.password) callback(new Error('两次输入的密码不一致'))
        else callback()
      },
      trigger: 'blur',
    },
  ],
}

async function onSubmit() {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  loading.value = true
  try {
    await register({
      username: form.username,
      email: form.email,
      password: form.password,
      display_name: form.display_name || undefined,
    })
    ElMessage.success('注册成功，请登录')
    router.push({ name: 'login' })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '注册失败，请重试')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthShell>
    <h1 class="text-xl font-bold text-ink">创建账号</h1>
    <p class="mt-1 text-sm text-ink-secondary">加入 edurec，发现适合你的教育资源</p>

    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="mt-6">
      <el-form-item label="用户名" prop="username">
        <el-input v-model="form.username" placeholder="用户名" size="large" autocomplete="username" />
      </el-form-item>
      <el-form-item label="邮箱" prop="email">
        <el-input v-model="form.email" placeholder="邮箱" size="large" autocomplete="email" />
      </el-form-item>
      <el-form-item label="密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          placeholder="密码"
          size="large"
          show-password
          autocomplete="new-password"
        />
      </el-form-item>
      <el-form-item label="确认密码" prop="confirmPassword">
        <el-input
          v-model="form.confirmPassword"
          type="password"
          placeholder="再次输入密码"
          size="large"
          show-password
          autocomplete="new-password"
        />
      </el-form-item>
      <el-form-item label="昵称（选填）" prop="display_name">
        <el-input v-model="form.display_name" placeholder="昵称" size="large" />
      </el-form-item>
      <el-button type="primary" size="large" class="mt-2 w-full" :loading="loading" @click="onSubmit">
        注 册
      </el-button>
    </el-form>

    <p class="mt-6 text-center text-sm text-ink-secondary">
      已有账号？
      <RouterLink :to="{ name: 'login' }" class="text-primary hover:underline">去登录</RouterLink>
    </p>
  </AuthShell>
</template>
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd frontend
pnpm test src/pages/auth/__tests__/RegisterPage.test.ts
```

Expected: PASS。若 `setValue` 未触发 v-model，改用 `wrapper.findComponent({ name: 'ElInput' }).vm.$emit('update:modelValue', value)`。

- [ ] **Step 5: 验证**

```bash
cd frontend
pnpm type-check
pnpm test
pnpm build
```

Expected: 全部通过。

- [ ] **Step 6: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): 实现注册页与测试"
```

---

## 验证清单（全部 Task 完成后）

- [ ] `pnpm build` —— vue-tsc 类型检查 + vite 构建通过
- [ ] `pnpm test` —— 全部测试通过（含 AuthShell 1、LoginPage 3、RegisterPage 2）
- [ ] `pnpm dev` 手动验证：`/login` 独立卡片渲染、`admin/admin123` 登录跳首页、`/register` 注册后跳 `/login` 提示
- [ ] 3 次提交均在 `feat/frontend-auth` 分支，每次提交后停下等待用户确认
