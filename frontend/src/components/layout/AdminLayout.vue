<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const TITLES: Record<string, string> = {
  'admin-dashboard': '仪表盘',
  'admin-resources': '资源管理',
  'admin-users': '用户管理',
  'admin-categories': '分类管理',
  'admin-recommendations': '推荐管理',
}

const pageTitle = computed(() => TITLES[String(route.name ?? '')] ?? '管理后台')

const navItems = [
  { to: { name: 'admin-dashboard' }, label: '仪表盘', color: 'var(--color-line-teal)' },
  { to: { name: 'admin-resources' }, label: '资源管理', color: 'var(--color-line-blue)' },
  { to: { name: 'admin-users' }, label: '用户管理', color: 'var(--color-line-amber)' },
  { to: { name: 'admin-categories' }, label: '分类管理', color: 'var(--color-line-rose)' },
  { to: { name: 'admin-recommendations' }, label: '推荐管理', color: 'var(--color-success)' },
]

function isActive(name: string): boolean {
  return route.name === name
}

function logout() {
  auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="flex min-h-screen bg-bg">
    <aside class="hidden w-60 shrink-0 flex-col bg-rail text-white md:flex">
      <RouterLink :to="{ name: 'admin-dashboard' }" class="flex h-16 items-center gap-2.5 border-b border-white/10 px-5">
        <span class="flex h-8 w-8 items-center justify-center rounded-[3px] bg-white" aria-hidden="true">
          <svg viewBox="0 0 24 24" class="h-5 w-5" fill="none">
            <path d="M8 5v6.5a3 3 0 0 0 3 3h5" stroke="#0B1B2B" stroke-width="2" stroke-linecap="square" />
            <circle cx="8" cy="4" r="2" fill="#C9451F" />
            <circle cx="17" cy="14.5" r="2" fill="#0E7C7B" />
          </svg>
        </span>
        <span class="flex flex-col leading-none">
          <span class="text-[17px] font-bold tracking-tight">edurec</span>
          <span class="mt-1 text-[10px] font-semibold tracking-[0.18em] text-white/60">管理后台</span>
        </span>
      </RouterLink>

      <nav class="flex-1 space-y-1 px-3 py-4" aria-label="管理导航">
        <RouterLink
          v-for="item in navItems"
          :key="item.label"
          :to="item.to"
          :aria-current="isActive(String(item.to.name)) ? 'page' : undefined"
          class="flex items-center gap-2.5 rounded-[3px] px-3 py-2 text-sm transition-colors"
          :class="isActive(String(item.to.name))
            ? 'bg-white/10 font-semibold text-white'
            : 'text-white/60 hover:bg-white/5 hover:text-white'"
        >
          <span
            class="route-node"
            :class="{ 'route-node--filled': isActive(String(item.to.name)) }"
            :style="{ '--node': item.color }"
            aria-hidden="true"
          ></span>
          {{ item.label }}
        </RouterLink>
      </nav>

      <div class="border-t border-white/10 p-4 text-xs text-white/55">
        当前管理员：{{ auth.user?.display_name ?? auth.user?.username }}
      </div>
    </aside>

    <div class="flex min-w-0 flex-1 flex-col">
      <header class="flex h-16 items-center justify-between gap-3 border-b border-border bg-surface px-4 sm:px-6">
        <div class="flex min-w-0 items-center gap-3">
          <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-[3px] bg-rail md:hidden" aria-hidden="true">
            <svg viewBox="0 0 24 24" class="h-5 w-5" fill="none">
              <path d="M8 5v6.5a3 3 0 0 0 3 3h5" stroke="#FFFFFF" stroke-width="2" stroke-linecap="square" />
              <circle cx="8" cy="4" r="2" fill="#C9451F" />
              <circle cx="17" cy="14.5" r="2" fill="#0E7C7B" />
            </svg>
          </span>
          <h1 class="truncate text-base font-bold text-ink">{{ pageTitle }}</h1>
        </div>
        <div class="flex shrink-0 items-center gap-3">
          <RouterLink :to="{ name: 'home' }">
            <el-button size="small" text>返回前台</el-button>
          </RouterLink>
          <el-button size="small" @click="logout">退出</el-button>
        </div>
      </header>

      <nav class="flex gap-1 overflow-x-auto border-b border-border bg-surface px-4 py-2 md:hidden" aria-label="管理导航">
        <RouterLink
          v-for="item in navItems"
          :key="item.label"
          :to="item.to"
          :aria-current="isActive(String(item.to.name)) ? 'page' : undefined"
          class="shrink-0 rounded-[3px] px-3 py-1.5 text-sm"
          :class="isActive(String(item.to.name))
            ? 'bg-rail font-semibold text-white'
            : 'text-ink-secondary'"
        >{{ item.label }}</RouterLink>
      </nav>

      <main class="flex-1 overflow-auto p-4 sm:p-6">
        <RouterView v-slot="{ Component }">
          <Transition name="route" mode="out-in">
            <component :is="Component" />
          </Transition>
        </RouterView>
      </main>
    </div>
  </div>
</template>
