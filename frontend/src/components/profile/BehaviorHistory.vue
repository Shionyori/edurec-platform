<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listMyBehaviors } from '@/api/behavior'
import { formatDate } from '@/utils/format'
import AppIcon from '@/components/ui/AppIcon.vue'
import type { Behavior, BehaviorAction } from '@/types'

const items = ref<Behavior[]>([])
const loading = ref(true) // 首帧即视为加载中，避免先闪一下空态
const error = ref('')
const total = ref(0)
const action = ref<BehaviorAction | ''>('')
const page = ref(1)
const pageSize = 10

// 首次加载显示骨架；之后切换筛选/翻页保留旧列表 + loading 遮罩，避免整块内容闪烁
const firstLoad = ref(true)

const ACTION_LABELS: Record<BehaviorAction, string> = {
  view: '浏览',
  click: '点击',
  favorite: '收藏',
}
const ACTION_ICONS: Record<BehaviorAction, string> = {
  view: 'eye',
  click: 'arrow-up-right',
  favorite: 'heart',
}
const ACTION_CLASSES: Record<BehaviorAction, string> = {
  view: 'bg-sunken text-ink-secondary',
  click: 'bg-info-soft text-info',
  favorite: 'bg-primary-soft text-primary-active',
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
  <div class="card p-5">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-2.5">
        <span class="flex h-8 w-8 items-center justify-center rounded-[9px] bg-sunken text-ink-secondary">
          <AppIcon name="clock" :size="16" />
        </span>
        <h2 class="text-[16px] font-bold text-ink">行为历史</h2>
      </div>
      <el-radio-group v-model="action" size="small" @change="handleFilter">
        <el-radio-button value="">全部</el-radio-button>
        <el-radio-button value="view">浏览</el-radio-button>
        <el-radio-button value="click">点击</el-radio-button>
        <el-radio-button value="favorite">收藏</el-radio-button>
      </el-radio-group>
    </div>

    <!-- 内容区预留一整页高度（列表 + 分页器）：无论切到哪个筛选，卡片高度恒定，页面不会因变矮而上移 -->
    <div class="mt-4 flex min-h-[30rem] flex-col" data-testid="history-content">
      <div class="flex-1">
        <!-- 首次加载：骨架屏 -->
        <div v-if="firstLoad && loading && !error">
          <el-skeleton :rows="4" animated />
        </div>
        <div v-else-if="error" class="state-box !border-0 !bg-surface-2 !py-16">
          <span class="state-icon !bg-danger-soft !text-danger">
            <AppIcon name="alert" :size="18" />
          </span>
          <p class="state-title">行为历史加载失败</p>
          <p class="state-copy">{{ error }}</p>
        </div>
        <div v-else-if="items.length === 0" class="state-box !border-0 !bg-surface-2 !py-16">
          <span class="state-icon !h-10 !w-10">
            <AppIcon name="clock" :size="18" />
          </span>
          <p class="state-title">暂无行为记录</p>
          <p class="state-copy">浏览、点击或收藏资源后，记录会出现在这里。</p>
        </div>
        <!-- 切换筛选/翻页：保留列表 + 轻量 loading 遮罩，内容不闪 -->
        <div v-else v-loading="loading">
          <TransitionGroup name="list" tag="ul" class="relative divide-y divide-border">
            <li
              v-for="b in items"
              :key="b.id"
              class="-mx-2 flex items-center justify-between gap-3 rounded-[9px] px-2 py-3 transition-colors hover:bg-surface-2"
            >
              <div class="flex min-w-0 items-center gap-2.5">
                <span
                  class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md"
                  :class="ACTION_CLASSES[b.action]"
                >
                  <AppIcon :name="ACTION_ICONS[b.action]" :size="13" />
                </span>
                <span class="chip !py-0 !text-[10.5px]" :class="ACTION_CLASSES[b.action]">
                  {{ ACTION_LABELS[b.action] }}
                </span>
                <RouterLink
                  v-if="b.resource"
                  :to="{ name: 'resource-detail', params: { id: b.resource.id } }"
                  class="truncate text-[13px] font-medium text-ink transition-colors hover:text-primary"
                >
                  {{ b.resource.title }}
                </RouterLink>
                <span v-else class="text-[13px] text-ink-muted">资源已删除</span>
              </div>
              <span class="tnum shrink-0 text-[11.5px] text-ink-muted">{{ formatDate(b.created_at) }}</span>
            </li>
          </TransitionGroup>
        </div>
      </div>

      <!-- 分页器固定在卡片底部：有则显示，无则留空，高度稳定 -->
      <div class="mt-4 flex justify-center" data-testid="history-pagination">
        <el-pagination
          v-if="total > pageSize"
          size="small"
          background
          layout="prev, pager, next"
          :pager-count="5"
          :total="total"
          :page-size="pageSize"
          :current-page="page"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>
