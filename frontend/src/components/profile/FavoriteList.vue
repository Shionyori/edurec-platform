<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listFavorites, setFavorite } from '@/api/behavior'
import type { Resource } from '@/types'

const items = ref<Resource[]>([])
const total = ref(0)
const loading = ref(true)
const firstLoad = ref(true)
const error = ref('')
const removingId = ref<number | null>(null)

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
    firstLoad.value = false
  }
}

async function remove(resource: Resource) {
  removingId.value = resource.id
  try {
    await setFavorite(resource.id, false)
    items.value = items.value.filter((r) => r.id !== resource.id)
    total.value = Math.max(0, total.value - 1)
    ElMessage.success('已取消收藏')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败，请重试')
  } finally {
    removingId.value = null
  }
}

onMounted(fetchFavorites)
</script>

<template>
  <div class="rounded-xl border border-border bg-surface p-6 shadow-sm">
    <div class="flex items-center justify-between">
      <h2 class="text-lg font-semibold text-ink">我的收藏</h2>
      <span v-if="total > 0" class="text-xs text-ink-muted">共 {{ total }} 个</span>
    </div>

    <!-- 首次加载：骨架 -->
    <div v-if="firstLoad && loading && !error" class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
      <el-skeleton v-for="i in 4" :key="i" :rows="1" animated />
    </div>
    <div v-else-if="error" class="py-12 text-center text-sm text-red-500">{{ error }}</div>
    <el-empty
      v-else-if="items.length === 0"
      description="还没有收藏，去资源详情页点「收藏」试试"
      :image-size="80"
      class="py-6"
    />
    <!-- 紧凑条目：小缩略图 + 标题 + 元信息，避免大封面卡片占满屏 -->
    <div v-else v-loading="loading" class="mt-4 min-h-[9rem]">
      <TransitionGroup name="list" tag="ul" class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <li
          v-for="r in items"
          :key="r.id"
          class="flex items-center gap-3 rounded-lg border border-border p-2 transition-colors hover:border-primary/30 hover:bg-bg/70"
        >
          <RouterLink
            :to="{ name: 'resource-detail', params: { id: r.id } }"
            class="flex min-w-0 flex-1 items-center gap-3"
          >
            <img
              v-if="r.cover_url"
              :src="r.cover_url"
              :alt="r.title"
              referrerpolicy="no-referrer"
              loading="lazy"
              class="h-12 w-20 shrink-0 rounded-md object-cover"
            />
            <div
              v-else
              class="flex h-12 w-20 shrink-0 items-center justify-center rounded-md bg-bg text-xs text-ink-muted"
            >
              {{ r.type }}
            </div>
            <div class="min-w-0">
              <p class="truncate text-sm font-medium text-ink">{{ r.title }}</p>
              <p class="mt-0.5 flex items-center gap-2 text-xs text-ink-muted">
                <span v-if="r.category?.name" class="truncate">{{ r.category.name }}</span>
                <span class="shrink-0 text-amber-500">★ {{ r.avg_rating.toFixed(1) }}</span>
              </p>
            </div>
          </RouterLink>
          <el-button
            text
            size="small"
            class="shrink-0"
            :loading="removingId === r.id"
            @click="remove(r)"
          >
            取消收藏
          </el-button>
        </li>
      </TransitionGroup>
      <p v-if="total > items.length" class="mt-3 text-center text-xs text-ink-muted">
        仅显示最近 {{ items.length }} 个收藏
      </p>
    </div>
  </div>
</template>
