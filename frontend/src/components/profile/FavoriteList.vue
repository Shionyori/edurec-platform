<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listFavorites, setFavorite } from '@/api/behavior'
import AppIcon from '@/components/ui/AppIcon.vue'
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
  <div class="card p-5">
    <div class="flex items-center justify-between gap-3">
      <div class="flex items-center gap-2.5">
        <span class="flex h-8 w-8 items-center justify-center rounded-[9px] bg-primary-soft text-primary">
          <AppIcon name="heart" :size="16" />
        </span>
        <h2 class="text-[16px] font-bold text-ink">我的收藏</h2>
      </div>
      <span v-if="total > 0" class="tnum text-[11.5px] text-ink-muted">共 {{ total }} 个</span>
    </div>

    <div v-if="firstLoad && loading && !error" class="mt-4 space-y-3">
      <div v-for="i in 4" :key="i" class="flex items-center gap-3">
        <div class="skeleton h-12 w-20 shrink-0"></div>
        <div class="flex-1">
          <div class="skeleton h-3.5 w-3/4"></div>
          <div class="skeleton mt-2 h-3 w-1/2"></div>
        </div>
      </div>
    </div>

    <div v-else-if="error" class="state-box mt-4 !border-0 !bg-surface-2 !py-10">
      <span class="state-icon !bg-danger-soft !text-danger">
        <AppIcon name="alert" :size="18" />
      </span>
      <p class="state-title">收藏加载失败</p>
      <p class="state-copy">{{ error }}</p>
    </div>

    <div v-else-if="items.length === 0" class="state-box mt-4 !border-0 !bg-surface-2 !py-10">
      <span class="state-icon !h-10 !w-10">
        <AppIcon name="heart" :size="18" />
      </span>
      <p class="state-title">还没有收藏</p>
      <p class="state-copy">去资源详情页点「收藏」试试。</p>
    </div>

    <div v-else v-loading="loading" class="mt-4 min-h-[9rem]">
      <TransitionGroup name="list" tag="ul" class="space-y-2.5">
        <li
          v-for="r in items"
          :key="r.id"
          class="group flex items-center gap-3 rounded-[10px] border border-border p-2 transition-colors hover:border-border-strong hover:bg-surface-2"
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
              class="h-12 w-20 shrink-0 rounded-lg object-cover"
            />
            <div
              v-else
              class="flex h-12 w-20 shrink-0 items-center justify-center rounded-lg bg-sunken text-[11px] text-ink-muted"
            >
              {{ r.type }}
            </div>
            <div class="min-w-0">
              <p class="truncate text-[13px] font-semibold text-ink transition-colors group-hover:text-primary">{{ r.title }}</p>
              <p class="mt-0.5 flex items-center gap-2 text-[11.5px] text-ink-muted">
                <span v-if="r.category?.name" class="truncate">{{ r.category.name }}</span>
                <span v-if="r.avg_rating > 0" class="tnum flex shrink-0 items-center gap-1 text-ink-secondary">
                  <AppIcon name="star-filled" :size="10" class="text-[#d99a20]" />
                  {{ r.avg_rating.toFixed(1) }}
                </span>
                <span v-else class="shrink-0">暂无评分</span>
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
      <p v-if="total > items.length" class="mt-3 text-center text-[11.5px] text-ink-muted">
        仅显示最近 {{ items.length }} 个收藏
      </p>
    </div>
  </div>
</template>
