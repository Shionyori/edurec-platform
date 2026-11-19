<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppIcon from '@/components/ui/AppIcon.vue'

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
  { to: { name: 'admin-dashboard' }, label: '仪表盘', icon: 'dashboard' },
  { to: { name: 'admin-resources' }, label: '资源管理', icon: 'folder' },
  { to: { name: 'admin-users' }, label: '用户管理', icon: 'users' },
  { to: { name: 'admin-categories' }, label: '分类管理', icon: 'tag' },
  { to: { name: 'admin-recommendations' }, label: '推荐管理', icon: 'chart' },
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
  <div class="min-h-screen bg-bg md:flex">
    <aside class="sticky top-0 hidden h-screen w-[248px] shrink-0 flex-col border-r border-border bg-surface md:flex">
      <RouterLink
        :to="{ name: 'admin-dashboard' }"
        class="flex h-[72px] items-center gap-2.5 border-b border-border px-5"
        aria-label="edurec 管理后台"
      >
        <span class="flex h-9 w-9 items-center justify-center rounded-[10px] bg-ink text-primary" aria-hidden="true">
          <AppIcon name="spark-filled" :size="19" />
        </span>
        <span class="flex items-center gap-2">
          <span class="text-[18px] font-extrabold leading-none tracking-[-0.045em] text-ink">edurec</span>
          <span class="rounded-md bg-sunken px-1.5 py-0.5 text-[10.5px] font-bold text-ink-muted">管理</span>
        </span>
      </RouterLink>

      <nav class="flex-1 space-y-1 px-3 py-5" aria-label="管理导航">
        <p class="mb-2 px-3 text-[11.5px] font-semibold text-ink-muted">工作台</p>
        <RouterLink
          v-for="item in navItems"
          :key="item.label"
          :to="item.to"
          :aria-current="isActive(String(item.to.name)) ? 'page' : undefined"
          class="flex items-center gap-2.5 rounded-[10px] px-3 py-2.5 text-[13.5px] font-semibold transition-colors"
          :class="isActive(String(item.to.name))
            ? 'bg-primary-soft text-primary-active'
            : 'text-ink-secondary hover:bg-surface-2 hover:text-ink'"
        >
          <AppIcon :name="item.icon" :size="17" />
          {{ item.label }}
        </RouterLink>
      </nav>

      <div class="border-t border-border p-3">
        <RouterLink
          :to="{ name: 'home' }"
          class="flex items-center gap-2 rounded-[10px] px-3 py-2.5 text-[13px] font-semibold text-ink-secondary transition-colors hover:bg-surface-2 hover:text-ink"
        >
          <AppIcon name="arrow-left" :size="16" />
          返回前台
        </RouterLink>
        <div class="mt-2 flex items-center gap-2.5 rounded-[10px] bg-surface-2 px-3 py-2.5">
          <el-avatar :size="30" :src="auth.user?.avatar_url ?? undefined" class="!bg-ink">
            {{ (auth.user?.display_name ?? auth.user?.username ?? 'A').charAt(0) }}
          </el-avatar>
          <div class="min-w-0 flex-1">
            <p class="truncate text-[12.5px] font-semibold text-ink">
              {{ auth.user?.display_name ?? auth.user?.username }}
            </p>
            <p class="truncate text-[11px] text-ink-muted">管理员</p>
          </div>
          <button type="button" class="icon-btn !h-8 !w-8" aria-label="退出登录" @click="logout">
            <AppIcon name="logout" :size="16" />
          </button>
        </div>
      </div>
    </aside>

    <div class="flex min-w-0 flex-1 flex-col">
      <header class="sticky top-0 z-30 border-b border-border bg-surface">
        <div class="flex h-16 items-center justify-between gap-3 px-4 sm:px-6 lg:px-8">
          <div class="flex min-w-0 items-center gap-3">
            <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-[9px] bg-ink text-primary md:hidden" aria-hidden="true">
              <AppIcon name="spark-filled" :size="17" />
            </span>
            <div class="min-w-0">
              <p class="hidden text-[11.5px] font-semibold text-ink-muted md:block">管理后台</p>
              <p class="truncate text-[16px] font-bold text-ink md:text-[18px]">{{ pageTitle }}</p>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <RouterLink :to="{ name: 'home' }" class="btn btn-secondary btn-sm hidden sm:inline-flex">
              <AppIcon name="external" :size="14" />
              返回前台
            </RouterLink>
            <button type="button" class="btn btn-ghost btn-sm md:hidden" @click="logout">退出</button>
          </div>
        </div>

        <nav class="flex gap-1 overflow-x-auto border-t border-border px-3 py-2 md:hidden" aria-label="管理导航">
          <RouterLink
            v-for="item in navItems"
            :key="item.label"
            :to="item.to"
            :aria-current="isActive(String(item.to.name)) ? 'page' : undefined"
            class="shrink-0 rounded-full px-3 py-1.5 text-[12.5px] font-semibold"
            :class="isActive(String(item.to.name))
              ? 'bg-primary-soft text-primary-active'
              : 'text-ink-secondary'"
          >{{ item.label }}</RouterLink>
        </nav>
      </header>

      <main class="flex-1 px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
        <RouterView v-slot="{ Component }">
          <Transition name="route" mode="out-in">
            <component :is="Component" />
          </Transition>
        </RouterView>
      </main>
    </div>
  </div>
</template>
