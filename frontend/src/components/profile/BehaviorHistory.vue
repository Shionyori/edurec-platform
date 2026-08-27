<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listMyBehaviors } from '@/api/behavior'
import { formatDate } from '@/utils/format'
import type { Behavior, BehaviorAction } from '@/types'

const items = ref<Behavior[]>([])
const loading = ref(false)
const error = ref('')
const total = ref(0)
const action = ref<BehaviorAction | ''>('')
const page = ref(1)
const pageSize = 10

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
  <div class="rounded-lg border border-border bg-surface p-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h2 class="text-lg font-semibold text-ink">行为历史</h2>
      <el-radio-group v-model="action" size="small" @change="handleFilter">
        <el-radio-button value="">全部</el-radio-button>
        <el-radio-button value="view">浏览</el-radio-button>
        <el-radio-button value="click">点击</el-radio-button>
        <el-radio-button value="favorite">收藏</el-radio-button>
      </el-radio-group>
    </div>

    <div v-if="loading" class="py-16 text-center text-sm text-ink-muted">加载中…</div>
    <div v-else-if="error" class="py-16 text-center text-sm text-red-500">{{ error }}</div>
    <div v-else-if="items.length === 0" class="py-16 text-center text-sm text-ink-muted">暂无行为记录</div>
    <ul v-else class="mt-4 divide-y divide-border">
      <li v-for="b in items" :key="b.id" class="flex items-center justify-between py-3">
        <div class="flex items-center gap-3">
          <el-tag
            size="small"
            :type="b.action === 'favorite' ? 'warning' : b.action === 'click' ? 'success' : 'info'"
          >
            {{ b.action }}
          </el-tag>
          <RouterLink
            v-if="b.resource"
            :to="{ name: 'resource-detail', params: { id: b.resource.id } }"
            class="text-sm text-ink hover:text-primary"
          >
            {{ b.resource.title }}
          </RouterLink>
          <span v-else class="text-sm text-ink-muted">资源已删除</span>
        </div>
        <span class="text-xs text-ink-muted">{{ formatDate(b.created_at) }}</span>
      </li>
    </ul>

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
