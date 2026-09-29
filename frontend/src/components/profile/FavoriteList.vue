<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listFavorites } from '@/api/behavior'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import type { Resource } from '@/types'

const items = ref<Resource[]>([])
const total = ref(0)
const loading = ref(false)
const error = ref('')

async function fetchFavorites() {
  loading.value = true
  error.value = ''
  try {
    const data = await listFavorites({ page: 1, page_size: 12 })
    items.value = data.list
    total.value = data.total
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

defineExpose({ fetchFavorites })
onMounted(fetchFavorites)
</script>

<template>
  <div class="rounded-xl border border-border bg-surface p-6 shadow-sm">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-semibold text-ink">我的收藏</h2>
      <span v-if="total > 0" class="text-xs text-ink-muted">共 {{ total }} 个</span>
    </div>

    <el-skeleton v-if="loading" :rows="3" animated class="mt-4" />
    <div v-else-if="error" class="py-12 text-center text-sm text-red-500">{{ error }}</div>
    <el-empty
      v-else-if="items.length === 0"
      description="还没有收藏任何资源，去详情页点「收藏」试试"
      :image-size="80"
    />
    <div v-else class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <ResourceCard v-for="r in items" :key="r.id" :resource="r" />
    </div>
  </div>
</template>
