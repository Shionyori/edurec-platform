<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { adminListResources, adminListUsers } from '@/api/admin'
import AppIcon from '@/components/ui/AppIcon.vue'

const resourceTotal = ref(0)
const userTotal = ref(0)
const loading = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    const [res, users] = await Promise.all([
      adminListResources({ page: 1, page_size: 1 }),
      adminListUsers({ page: 1, page_size: 1 }),
    ])
    resourceTotal.value = res.total
    userTotal.value = users.total
  } catch {
    // 统计加载失败不阻塞页面
  } finally {
    loading.value = false
  }
})

const quickLinks = [
  { to: { name: 'admin-resources' }, label: '资源管理', copy: '维护平台教育资源', icon: 'folder' },
  { to: { name: 'admin-users' }, label: '用户管理', copy: '查看平台注册用户', icon: 'users' },
  { to: { name: 'admin-categories' }, label: '分类管理', copy: '维护教育资源分类', icon: 'tag' },
]
</script>

<template>
  <div class="mx-auto max-w-[1200px]">
    <header>
      <h1 class="headline">仪表盘</h1>
      <p class="mt-2 text-[13px] text-ink-secondary">查看平台规模，并快速进入日常管理工作。</p>
    </header>

    <div v-if="loading" class="mt-7 grid grid-cols-1 gap-5 sm:grid-cols-2">
      <div v-for="i in 2" :key="i" class="card p-6">
        <div class="skeleton h-4 w-20"></div>
        <div class="skeleton mt-4 h-9 w-24"></div>
      </div>
    </div>

    <div v-else class="mt-7 grid grid-cols-1 gap-5 sm:grid-cols-2">
      <RouterLink :to="{ name: 'admin-resources' }" class="group card interactive p-6">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="text-[12.5px] font-semibold text-ink-muted">资源总数</p>
            <p class="tnum mt-2 text-[38px] font-extrabold leading-none tracking-[-0.05em] text-ink">{{ resourceTotal }}</p>
          </div>
          <span class="flex h-11 w-11 items-center justify-center rounded-[11px] bg-primary-soft text-primary">
            <AppIcon name="folder" :size="20" />
          </span>
        </div>
        <p class="mt-5 flex items-center gap-1.5 text-[12px] font-semibold text-ink-secondary transition-colors group-hover:text-primary">
          进入资源管理
          <AppIcon name="arrow-right" :size="13" />
        </p>
      </RouterLink>

      <RouterLink :to="{ name: 'admin-users' }" class="group card interactive p-6">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="text-[12.5px] font-semibold text-ink-muted">用户总数</p>
            <p class="tnum mt-2 text-[38px] font-extrabold leading-none tracking-[-0.05em] text-ink">{{ userTotal }}</p>
          </div>
          <span class="flex h-11 w-11 items-center justify-center rounded-[11px] bg-surface-2 text-ink-secondary">
            <AppIcon name="users" :size="20" />
          </span>
        </div>
        <p class="mt-5 flex items-center gap-1.5 text-[12px] font-semibold text-ink-secondary transition-colors group-hover:text-primary">
          进入用户管理
          <AppIcon name="arrow-right" :size="13" />
        </p>
      </RouterLink>
    </div>

    <section class="mt-8">
      <h2 class="text-[16px] font-bold text-ink">快捷入口</h2>
      <div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <RouterLink
          v-for="link in quickLinks"
          :key="link.label"
          :to="link.to"
          class="group card interactive flex items-center gap-3 p-4"
        >
          <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-[10px] bg-surface-2 text-ink-secondary transition-colors group-hover:bg-primary-soft group-hover:text-primary">
            <AppIcon :name="link.icon" :size="18" />
          </span>
          <span class="min-w-0">
            <span class="block text-[13.5px] font-bold text-ink">{{ link.label }}</span>
            <span class="block truncate text-[11.5px] text-ink-muted">{{ link.copy }}</span>
          </span>
          <AppIcon name="chevron-right" :size="15" class="ml-auto text-ink-muted transition-transform group-hover:translate-x-0.5" />
        </RouterLink>
      </div>
    </section>
  </div>
</template>
