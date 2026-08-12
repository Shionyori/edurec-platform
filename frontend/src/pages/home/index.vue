<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listResources } from '@/api/resource'
import type { Resource } from '@/types'

const resources = ref<Resource[]>([])
const loading = ref(false)
const error = ref('')

onMounted(async () => {
  loading.value = true
  try {
    const data = await listResources({ page: 1, page_size: 12, sort: 'latest' })
    resources.value = data.list
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="mx-auto max-w-7xl px-6 py-10">
    <div class="flex items-end justify-between">
      <div>
        <h1 class="text-2xl font-bold">发现优质教育资源</h1>
        <p class="mt-2 text-sm text-ink-secondary">精选课程、文章与视频</p>
      </div>
      <RouterLink :to="{ name: 'search' }">
        <el-button round>查看全部</el-button>
      </RouterLink>
    </div>

    <div v-if="loading" class="py-24 text-center text-sm text-ink-muted">加载中…</div>
    <div v-else-if="error" class="py-24 text-center text-sm text-red-500">{{ error }}</div>
    <div v-else class="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <div
        v-for="r in resources"
        :key="r.id"
        class="group rounded-lg border border-border bg-surface p-5 shadow-sm transition hover:shadow-md"
      >
        <div class="flex items-center justify-between text-xs text-ink-muted">
          <el-tag size="small" :type="r.type === 'course' ? 'primary' : r.type === 'video' ? 'success' : 'warning'">
            {{ r.type }}
          </el-tag>
          <span>{{ r.category?.name }}</span>
        </div>
        <h3 class="mt-3 text-lg font-semibold text-ink">{{ r.title }}</h3>
        <p class="mt-1 line-clamp-2 text-sm text-ink-secondary">{{ r.description }}</p>
        <div class="mt-4 flex items-center justify-between border-t border-border pt-3 text-sm">
          <span class="text-amber-500">★ {{ r.avg_rating.toFixed(1) }}</span>
          <span class="text-xs text-ink-muted">{{ r.view_count }} 次浏览</span>
        </div>
      </div>
    </div>
  </div>
</template>
