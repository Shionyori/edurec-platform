<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

// 顶栏搜索框：原先没有 v-model，输入的内容会被渲染清掉（看起来"打不进字"）。
// 现在绑定 keyword 并回车跳搜索页；已在搜索页时改写 query，由页面 watch 触发重新搜索。
const headerKeyword = ref('')

function handleCommand(command: string) {
  if (command === 'logout') {
    auth.logout()
    router.push({ name: 'login' })
  } else if (command === 'me') {
    router.push({ name: 'user-me' })
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
</script>

<template>
  <div class="flex min-h-screen flex-col bg-bg">
    <header class="sticky top-0 z-40 border-b border-border bg-surface/95 backdrop-blur">
      <div class="mx-auto flex h-16 max-w-7xl items-center gap-6 px-6">
        <RouterLink to="/" class="flex items-center gap-2 text-lg font-bold text-primary">
          <span class="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-white">E</span>
          edurec
        </RouterLink>
        <el-input
          v-model="headerKeyword"
          placeholder="搜索课程、文章、视频…"
          class="max-w-sm"
          clearable
          @keyup.enter="submitSearch"
        />
        <nav class="ml-4 hidden items-center gap-5 text-sm text-ink-secondary md:flex">
          <RouterLink to="/" class="hover:text-primary">首页</RouterLink>
          <RouterLink :to="{ name: 'search' }" class="hover:text-primary">课程</RouterLink>
          <RouterLink :to="{ name: 'search' }" class="hover:text-primary">文章</RouterLink>
        </nav>
        <div class="ml-auto flex items-center gap-3">
          <template v-if="auth.isLoggedIn">
            <el-dropdown @command="handleCommand">
              <span class="flex cursor-pointer items-center gap-2 text-sm">
                <el-avatar :size="28" :src="auth.user?.avatar_url ?? undefined">
                  {{ (auth.user?.display_name ?? auth.user?.username ?? 'U').charAt(0) }}
                </el-avatar>
                {{ auth.user?.display_name ?? auth.user?.username }}
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="me">个人中心</el-dropdown-item>
                  <el-dropdown-item divided command="logout">退出登录</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
          <template v-else>
            <RouterLink :to="{ name: 'login' }">
              <el-button type="primary" round>登录 / 注册</el-button>
            </RouterLink>
          </template>
        </div>
      </div>
    </header>

    <main class="flex-1">
      <RouterView />
    </main>

    <footer class="border-t border-border py-6 text-center text-xs text-ink-muted">
      © 2026 edurec · 教育资源推荐平台
    </footer>
  </div>
</template>
