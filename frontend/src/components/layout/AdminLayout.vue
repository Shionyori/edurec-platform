<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

function logout() {
  auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="flex min-h-screen bg-bg">
    <aside class="flex w-56 shrink-0 flex-col border-r border-border bg-surface">
      <div class="flex h-16 items-center gap-2 border-b border-border px-5">
        <span class="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-white">E</span>
        <span class="font-bold">管理后台</span>
      </div>
      <el-menu :default-active="route.path" class="flex-1 border-0" router>
        <el-menu-item index="/admin">仪表盘</el-menu-item>
        <el-menu-item index="/admin/resources">资源管理</el-menu-item>
        <el-menu-item index="/admin/users">用户管理</el-menu-item>
        <el-menu-item index="/admin/categories">分类管理</el-menu-item>
      </el-menu>
      <div class="border-t border-border p-4 text-xs text-ink-muted">
        当前管理员：{{ auth.user?.display_name ?? auth.user?.username }}
      </div>
    </aside>

    <div class="flex min-w-0 flex-1 flex-col">
      <header class="flex h-16 items-center justify-between border-b border-border bg-surface px-6">
        <div class="text-sm text-ink-secondary">{{ route.name }}</div>
        <div class="flex items-center gap-3">
          <RouterLink :to="{ name: 'home' }">
            <el-button size="small" text>返回前台</el-button>
          </RouterLink>
          <el-button size="small" @click="logout">退出</el-button>
        </div>
      </header>
      <main class="flex-1 overflow-auto p-6">
        <RouterView />
      </main>
    </div>
  </div>
</template>
