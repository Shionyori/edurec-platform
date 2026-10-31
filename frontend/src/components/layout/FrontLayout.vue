<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

// 当前搜索页的类型筛选，用于顶栏导航高亮（路由的 active 只看路径，无法区分 query）
const activeType = computed(() => {
  const raw = route.query.type
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === 'string' ? value : ''
})

const headerKeyword = ref('')

function handleCommand(command: string) {
  if (command === 'logout') {
    auth.logout()
    router.push({ name: 'login' })
  } else if (command === 'me') {
    router.push({ name: 'user-me' })
  } else if (command === 'questionnaire') {
    router.push({ name: 'questionnaire' })
  }
}

function submitSearch() {
  const kw = headerKeyword.value.trim()
  if (!kw) return
  const current = typeof route.query.keyword === 'string' ? route.query.keyword : ''
  // 已在搜索页且关键词未变：不重复导航（输入框里的文字保留，方便继续改）
  if (route.name === 'search' && current === kw) return
  router.push({ name: 'search', query: { keyword: kw } })
}

const navItems = [
  { key: 'home', label: '首页', to: { name: 'home' } },
  { key: 'knowledge', label: '知识点', to: { name: 'knowledge' } },
  { key: 'all', label: '全部', to: { name: 'search' } },
  { key: 'course', label: '课程', to: { name: 'search', query: { type: 'course' } } },
  { key: 'video', label: '视频', to: { name: 'search', query: { type: 'video' } } },
  { key: 'article', label: '文章', to: { name: 'search', query: { type: 'article' } } },
]

function isActive(key: string): boolean {
  if (key === 'home') return route.name === 'home'
  if (key === 'knowledge') return String(route.name ?? '').startsWith('knowledge') || String(route.name ?? '').startsWith('discipline')
  if (key === 'all') return route.name === 'search' && !activeType.value
  return route.name === 'search' && activeType.value === key
}
</script>

<template>
  <div class="flex min-h-screen flex-col bg-bg">
    <a
      href="#main-content"
      class="sr-only z-50 bg-primary px-4 py-2 text-sm font-semibold text-white focus:not-sr-only focus:absolute focus:left-4 focus:top-4"
    >跳到主要内容</a>

    <header class="sticky top-0 z-40 bg-rail text-white">
      <div class="mx-auto flex h-16 max-w-7xl items-center gap-5 px-4 sm:px-6">
        <RouterLink to="/" class="flex shrink-0 items-center gap-2.5" aria-label="edurec 首页">
          <span class="flex h-8 w-8 items-center justify-center rounded-[3px] bg-white" aria-hidden="true">
            <svg viewBox="0 0 24 24" class="h-5 w-5" fill="none">
              <path d="M8 5v6.5a3 3 0 0 0 3 3h5" stroke="#0B1B2B" stroke-width="2" stroke-linecap="square" />
              <circle cx="8" cy="4" r="2" fill="#E4572E" />
              <circle cx="17" cy="14.5" r="2" fill="#0E7C7B" />
            </svg>
          </span>
          <span class="flex flex-col leading-none">
            <span class="text-[17px] font-bold tracking-tight">edurec</span>
            <span class="mt-1 text-[10px] font-semibold tracking-[0.18em] text-white/45">学习线路</span>
          </span>
        </RouterLink>

        <div class="rail-control hidden w-64 lg:block">
          <el-input
            v-model="headerKeyword"
            placeholder="搜索课程、文章、视频…"
            clearable
            @keyup.enter="submitSearch"
          />
        </div>

        <nav class="hidden items-center gap-1 md:flex" aria-label="主导航">
          <RouterLink
            v-for="item in navItems"
            :key="item.key"
            :to="item.to"
            class="border-b-2 px-3 py-5 text-sm transition-colors duration-150"
            :class="isActive(item.key)
              ? 'border-primary font-semibold text-white'
              : 'border-transparent text-white/65 hover:text-white'"
          >{{ item.label }}</RouterLink>
        </nav>

        <div class="ml-auto flex items-center gap-3">
          <template v-if="auth.isLoggedIn">
            <el-dropdown @command="handleCommand">
              <span class="flex cursor-pointer items-center gap-2 text-sm text-white/85 hover:text-white">
                <el-avatar :size="28" :src="auth.user?.avatar_url ?? undefined" class="ring-1 ring-white/25">
                  {{ (auth.user?.display_name ?? auth.user?.username ?? 'U').charAt(0) }}
                </el-avatar>
                <span class="hidden max-w-28 truncate sm:block">
                  {{ auth.user?.display_name ?? auth.user?.username }}
                </span>
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="me">个人中心</el-dropdown-item>
                  <el-dropdown-item command="questionnaire">学习问卷</el-dropdown-item>
                  <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
          <template v-else>
            <RouterLink :to="{ name: 'login' }">
              <el-button type="primary">登录 / 注册</el-button>
            </RouterLink>
          </template>
        </div>
      </div>

      <!-- 移动端：横向滚动的线路导航，避免窄屏完全失去导航 -->
      <nav
        class="flex gap-1 overflow-x-auto border-t border-white/10 px-4 py-2 md:hidden"
        aria-label="移动端导航"
      >
        <RouterLink
          v-for="item in navItems"
          :key="item.key"
          :to="item.to"
          class="shrink-0 rounded-[3px] px-3 py-1.5 text-sm"
          :class="isActive(item.key)
            ? 'bg-white/12 font-semibold text-white'
            : 'text-white/60'"
        >{{ item.label }}</RouterLink>
      </nav>
    </header>

    <main id="main-content" class="flex-1">
      <RouterView v-slot="{ Component }">
        <Transition name="route" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>

    <footer class="border-t border-border bg-surface">
      <div class="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-3 px-6 py-5 text-xs text-ink-muted">
        <span class="flex items-center gap-2">
          <span class="route-node" style="--node: var(--color-line-teal)" aria-hidden="true"></span>
          edurec · 教育资源推荐平台
        </span>
        <span class="data-num">以知识点与先修关系组织学习</span>
      </div>
    </footer>
  </div>
</template>
