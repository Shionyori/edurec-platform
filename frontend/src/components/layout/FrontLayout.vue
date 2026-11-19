<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppIcon from '@/components/ui/AppIcon.vue'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const headerKeyword = ref('')
const mobileSearchOpen = ref(false)

const navItems = [
  { key: 'home', label: '首页', to: { name: 'home' }, icon: 'home' },
  { key: 'discover', label: '发现', to: { name: 'search' }, icon: 'compass' },
  { key: 'knowledge', label: '知识点', to: { name: 'knowledge' }, icon: 'layers' },
]

const mobileNav = computed(() => [
  { key: 'home', label: '首页', icon: 'home', to: { name: 'home' } },
  { key: 'discover', label: '发现', icon: 'compass', to: { name: 'search' } },
  { key: 'knowledge', label: '知识点', icon: 'layers', to: { name: 'knowledge' } },
  {
    key: 'me',
    label: auth.isLoggedIn ? '我的' : '登录',
    icon: 'user',
    to: auth.isLoggedIn ? { name: 'user-me' } : { name: 'login' },
  },
])

function isActive(key: string): boolean {
  const name = String(route.name ?? '')
  if (key === 'home') return route.name === 'home'
  if (key === 'discover') return route.name === 'search'
  if (key === 'knowledge') return name.startsWith('knowledge') || name.startsWith('discipline')
  return false
}

function handleCommand(command: string) {
  if (command === 'logout') {
    auth.logout()
    router.push({ name: 'login' })
  } else if (command === 'me') {
    router.push({ name: 'user-me' })
  } else if (command === 'questionnaire') {
    router.push({ name: 'questionnaire' })
  } else if (command === 'admin') {
    router.push({ name: 'admin-dashboard' })
  }
}

function submitSearch() {
  const kw = headerKeyword.value.trim()
  if (!kw) return
  const current = typeof route.query.keyword === 'string' ? route.query.keyword : ''
  if (route.name === 'search' && current === kw) {
    mobileSearchOpen.value = false
    return
  }
  mobileSearchOpen.value = false
  router.push({ name: 'search', query: { keyword: kw } })
}
</script>

<template>
  <div class="flex min-h-screen flex-col">
    <a
      href="#main-content"
      class="sr-only z-50 bg-primary px-4 py-2 text-sm font-semibold text-white focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:rounded-lg"
    >跳到主要内容</a>

    <header class="sticky top-0 z-40 border-b border-border bg-surface">
      <div class="shell flex h-16 items-center gap-4 md:h-[72px]">
        <RouterLink to="/" class="flex shrink-0 items-center gap-2.5" aria-label="edurec 首页">
          <span
            class="flex h-9 w-9 items-center justify-center rounded-[10px] bg-ink text-primary"
            aria-hidden="true"
          >
            <AppIcon name="spark-filled" :size="19" />
          </span>
          <span class="text-[19px] font-extrabold leading-none tracking-[-0.045em] text-ink">
            edurec
          </span>
        </RouterLink>

        <nav class="ml-2 hidden items-center gap-1 md:flex" aria-label="主导航">
          <RouterLink
            v-for="item in navItems"
            :key="item.key"
            :to="item.to"
            :aria-current="isActive(item.key) ? 'page' : undefined"
            class="rounded-[9px] px-3.5 py-2 text-[13.5px] font-semibold transition-colors duration-150"
            :class="isActive(item.key)
              ? 'bg-primary-soft text-primary-active'
              : 'text-ink-secondary hover:bg-sunken hover:text-ink'"
          >{{ item.label }}</RouterLink>
        </nav>

        <div class="ml-auto hidden w-56 lg:block">
          <el-input
            v-model="headerKeyword"
            placeholder="搜索课程、文章、视频…"
            clearable
            @keyup.enter="submitSearch"
          >
            <template #prefix>
              <AppIcon name="search" :size="16" />
            </template>
          </el-input>
        </div>

        <div class="ml-auto flex items-center gap-2 lg:ml-0">
          <button
            type="button"
            class="icon-btn lg:hidden"
            :aria-label="mobileSearchOpen ? '收起搜索' : '展开搜索'"
            :aria-expanded="mobileSearchOpen"
            @click="mobileSearchOpen = !mobileSearchOpen"
          >
            <AppIcon :name="mobileSearchOpen ? 'close' : 'search'" :size="19" />
          </button>

          <template v-if="auth.isLoggedIn">
            <el-dropdown trigger="click" @command="handleCommand">
              <button
                type="button"
                class="flex items-center gap-2 rounded-full border border-border bg-surface p-1 pr-2.5 transition-colors hover:border-border-strong hover:bg-surface-2"
              >
                <el-avatar
                  :size="30"
                  :src="auth.user?.avatar_url ?? undefined"
                  class="!bg-ink"
                >
                  {{ (auth.user?.display_name ?? auth.user?.username ?? 'U').charAt(0) }}
                </el-avatar>
                <span class="hidden max-w-24 truncate text-[13px] font-semibold text-ink sm:block">
                  {{ auth.user?.display_name ?? auth.user?.username }}
                </span>
                <AppIcon name="chevron-down" :size="14" class="text-ink-muted" />
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="me">
                    <span class="flex items-center gap-2"><AppIcon name="user" :size="15" />个人中心</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="questionnaire">
                    <span class="flex items-center gap-2"><AppIcon name="target" :size="15" />学习问卷</span>
                  </el-dropdown-item>
                  <el-dropdown-item v-if="auth.isAdmin" command="admin">
                    <span class="flex items-center gap-2"><AppIcon name="dashboard" :size="15" />管理后台</span>
                  </el-dropdown-item>
                  <el-dropdown-item divided command="logout">
                    <span class="flex items-center gap-2"><AppIcon name="logout" :size="15" />退出登录</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
          <template v-else>
            <RouterLink :to="{ name: 'login' }" class="btn btn-primary btn-sm sm:hidden">
              登录
            </RouterLink>
            <RouterLink :to="{ name: 'login' }" class="btn btn-primary hidden sm:inline-flex">
              登录
            </RouterLink>
          </template>
        </div>
      </div>

      <!-- Mobile search: the same field, revealed on demand instead of a cramped rail -->
      <div v-if="mobileSearchOpen" class="shell border-t border-border py-3 lg:hidden">
        <el-input
          v-model="headerKeyword"
          placeholder="搜索课程、文章、视频…"
          clearable
          @keyup.enter="submitSearch"
        >
          <template #prefix>
            <AppIcon name="search" :size="16" />
          </template>
        </el-input>
      </div>
    </header>

    <div class="flex-1 pb-[calc(76px+env(safe-area-inset-bottom))] md:pb-0">
      <main id="main-content">
        <RouterView v-slot="{ Component }">
          <Transition name="route" mode="out-in">
            <component :is="Component" />
          </Transition>
        </RouterView>
      </main>

      <footer class="mt-auto border-t border-border bg-surface">
        <div class="shell py-10 md:py-12">
          <div class="flex flex-col gap-8 md:flex-row md:items-start md:justify-between">
            <div class="max-w-sm">
              <RouterLink to="/" class="flex items-center gap-2.5" aria-label="edurec 首页">
                <span class="flex h-8 w-8 items-center justify-center rounded-[9px] bg-ink text-primary" aria-hidden="true">
                  <AppIcon name="spark-filled" :size="17" />
                </span>
                <span class="text-[17px] font-extrabold tracking-[-0.04em] text-ink">edurec</span>
              </RouterLink>
              <p class="mt-3 text-[13px] leading-relaxed text-ink-muted">
                用知识点与先修关系组织学习，跳过已掌握的内容，只推荐你当前该学的一节。
              </p>
            </div>
            <nav class="flex flex-wrap gap-x-8 gap-y-3 text-[13px] font-semibold text-ink-secondary" aria-label="页脚导航">
              <RouterLink :to="{ name: 'search' }" class="link-quiet">发现资源</RouterLink>
              <RouterLink :to="{ name: 'knowledge' }" class="link-quiet">知识点</RouterLink>
              <RouterLink :to="{ name: 'user-me' }" class="link-quiet">个人中心</RouterLink>
            </nav>
          </div>
          <div class="mt-8 flex flex-wrap items-center justify-between gap-3 border-t border-border pt-5 text-[12px] text-ink-muted">
            <span>© 2026 edurec · 教育资源推荐平台</span>
            <span class="flex items-center gap-1.5">
              <AppIcon name="spark" :size="13" class="text-primary" />
              先修关系驱动，推荐理由可读
            </span>
          </div>
        </div>
      </footer>
    </div>

    <!-- Mobile tab bar: the primary four destinations, always one thumb away -->
    <nav
      class="fixed inset-x-0 bottom-0 z-40 border-t border-border bg-surface pb-[env(safe-area-inset-bottom)] md:hidden"
      aria-label="移动端主导航"
    >
      <div class="grid grid-cols-4">
        <RouterLink
          v-for="item in mobileNav"
          :key="item.key"
          :to="item.to"
          :aria-current="isActive(item.key) || (item.key === 'me' && route.name === 'user-me') ? 'page' : undefined"
          class="flex flex-col items-center gap-1 py-2.5 text-[11px] font-semibold transition-colors"
          :class="isActive(item.key) || (item.key === 'me' && route.name === 'user-me')
            ? 'text-primary'
            : 'text-ink-muted'"
        >
          <span
            class="flex h-7 w-12 items-center justify-center rounded-full transition-colors"
            :class="isActive(item.key) || (item.key === 'me' && route.name === 'user-me') ? 'bg-primary-soft' : ''"
          >
            <AppIcon :name="item.icon" :size="19" />
          </span>
          {{ item.label }}
        </RouterLink>
      </div>
    </nav>
  </div>
</template>
