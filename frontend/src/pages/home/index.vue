<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getRecommendations } from '@/api/recommendation'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import type { Resource } from '@/types'

const resources = ref<Resource[]>([])
const loading = ref(false)
const error = ref('')

onMounted(async () => {
  loading.value = true
  try {
    const data = await getRecommendations(12)
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
        <p class="mt-2 text-sm text-ink-secondary">为你推荐的精选课程、文章与视频</p>
      </div>
      <RouterLink :to="{ name: 'search' }">
        <el-button round>查看全部</el-button>
      </RouterLink>
    </div>

    <div v-if="loading" class="py-24 text-center text-sm text-ink-muted">加载中…</div>
    <div v-else-if="error" class="py-24 text-center text-sm text-red-500">{{ error }}</div>
    <div v-else class="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <ResourceCard v-for="r in resources" :key="r.id" :resource="r" />
    </div>
  </div>
</template>
