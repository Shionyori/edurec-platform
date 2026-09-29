<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listMyBehaviors } from '@/api/behavior'
import { formatDate } from '@/utils/format'
import type { Behavior, BehaviorAction } from '@/types'

const items = ref<Behavior[]>([])
const loading = ref(true) // 首帧即视为加载中，避免先闪一下空态
const error = ref('')
const total = ref(0)
const action = ref<BehaviorAction | ''>('')
const page = ref(1)
const pageSize = 10

// 首次加载显示骨架；之后切换筛选/翻页保留旧列表 + loading 遮罩，避免整块内容闪烁与高度塌陷
const firstLoad = ref(true)

const ACTION_LABELS: Record<BehaviorAction, string> = {
  view: '浏览',
  click: '点击',
  favorite: '收藏',
}
const ACTION_TAG_TYPE: Record<BehaviorAction, 'info' | 'success' | 'warning'> = {
  view: 'info',
  click: 'success',
  favorite: 'warning',
}

async function fetchList() {
  loading.value = true
  error.value = ''
  try {
    const data = await listMyBehaviors({
      action: action.value || undefined,
      page: page.value,
      page_size: pageSize,
    })
    items.value = data.list
    total.value = data.total
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
    firstLoad.value = false
  }
}

function handleFilter() {
  page.value = 1
  fetchList()
}

function handlePageChange(p: number) {
  page.value = p
  fetchList()
}

onMounted(fetchList)
</script>

<template>
  <div class="rounded-xl border border-border bg-surface p-6 shadow-sm">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h2 class="text-lg font-semibold text-ink">行为历史</h2>
      <el-radio-group v-model="action" size="small" @change="handleFilter">
        <el-radio-button value="">全部</el-radio-button>
        <el-radio-button value="view">浏览</el-radio-button>
        <el-radio-button value="click">点击</el-radio-button>
        <el-radio-button value="favorite">收藏</el-radio-button>
      </el-radio-group>
    </div>

    <!-- 首次加载：骨架屏 -->
    <div v-if="firstLoad && loading && !error" class="mt-4">
      <el-skeleton :rows="4" animated />
    </div>
    <div v-else-if="error" class="py-16 text-center text-sm text-red-500">{{ error }}</div>
    <el-empty
      v-else-if="items.length === 0"
      description="暂无行为记录"
      :image-size="80"
      class="py-8"
    />
    <!-- 切换筛选/翻页：保留列表 + 轻量 loading 遮罩，内容不闪、高度不塌 -->
    <div v-else v-loading="loading" class="mt-4 min-h-[220px]">
      <TransitionGroup name="list" tag="ul" class="relative divide-y divide-border">
        <li
          v-for="b in items"
          :key="b.id"
          class="-mx-2 flex items-center justify-between rounded-md px-2 py-3 transition-colors hover:bg-bg/70"
        >
          <div class="flex min-w-0 items-center gap-3">
            <el-tag size="small" :type="ACTION_TAG_TYPE[b.action]">
              {{ ACTION_LABELS[b.action] }}
            </el-tag>
            <RouterLink
              v-if="b.resource"
              :to="{ name: 'resource-detail', params: { id: b.resource.id } }"
              class="truncate text-sm text-ink transition-colors hover:text-primary"
            >
              {{ b.resource.title }}
            </RouterLink>
            <span v-else class="text-sm text-ink-muted">资源已删除</span>
          </div>
          <span class="shrink-0 pl-3 text-xs text-ink-muted">{{ formatDate(b.created_at) }}</span>
        </li>
      </TransitionGroup>
    </div>

    <div v-if="total > pageSize" class="mt-4 flex justify-center">
      <el-pagination
        background
        layout="prev, pager, next"
        :total="total"
        :page-size="pageSize"
        :current-page="page"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>
