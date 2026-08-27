<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { adminListResources, adminListUsers } from '@/api/admin'

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
</script>

<template>
  <div>
    <h1 class="text-xl font-bold">仪表盘</h1>
    <p class="mt-1 text-sm text-ink-secondary">平台运营概览</p>

    <div v-if="loading" class="py-16 text-center text-sm text-ink-muted">加载中…</div>
    <div v-else class="mt-6 grid grid-cols-1 gap-6 sm:grid-cols-2">
      <RouterLink
        :to="{ name: 'admin-resources' }"
        class="rounded-lg border border-border bg-surface p-6 transition hover:shadow-md"
      >
        <p class="text-sm text-ink-muted">资源总数</p>
        <p class="mt-2 text-3xl font-bold text-ink">{{ resourceTotal }}</p>
      </RouterLink>
      <RouterLink
        :to="{ name: 'admin-users' }"
        class="rounded-lg border border-border bg-surface p-6 transition hover:shadow-md"
      >
        <p class="text-sm text-ink-muted">用户总数</p>
        <p class="mt-2 text-3xl font-bold text-ink">{{ userTotal }}</p>
      </RouterLink>
    </div>

    <div class="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
      <RouterLink
        :to="{ name: 'admin-resources' }"
        class="rounded-lg border border-border bg-surface p-5 text-sm text-ink-secondary transition hover:shadow-md"
      >
        资源管理
      </RouterLink>
      <RouterLink
        :to="{ name: 'admin-users' }"
        class="rounded-lg border border-border bg-surface p-5 text-sm text-ink-secondary transition hover:shadow-md"
      >
        用户管理
      </RouterLink>
      <RouterLink
        :to="{ name: 'admin-categories' }"
        class="rounded-lg border border-border bg-surface p-5 text-sm text-ink-secondary transition hover:shadow-md"
      >
        分类管理
      </RouterLink>
    </div>
  </div>
</template>
