<script setup lang="ts">
import { formatDate } from '@/utils/format'
import type { Rating } from '@/types'

defineProps<{
  ratings: Rating[]
  total: number
  page: number
  loading?: boolean
}>()

const emit = defineEmits<{
  'page-change': [page: number]
}>()
</script>

<template>
  <div>
    <div v-if="loading" class="py-16 text-center text-sm text-ink-muted">加载中…</div>
    <div v-else-if="ratings.length === 0" class="py-16 text-center text-sm text-ink-muted">
      暂无评价，来抢沙发吧
    </div>
    <ul v-else class="divide-y divide-border">
      <li v-for="r in ratings" :key="r.id" class="flex gap-4 py-4">
        <el-avatar :size="36">{{ (r.user.display_name ?? r.user.username).charAt(0) }}</el-avatar>
        <div class="min-w-0 flex-1">
          <div class="flex items-center justify-between">
            <span class="text-sm font-medium text-ink">{{ r.user.display_name ?? r.user.username }}</span>
            <span class="text-xs text-ink-muted">{{ formatDate(r.created_at) }}</span>
          </div>
          <el-rate :model-value="r.score" disabled class="mt-1" />
          <p v-if="r.comment" class="mt-1 text-sm text-ink-secondary">{{ r.comment }}</p>
        </div>
      </li>
    </ul>
    <div v-if="total > 10" class="mt-4 flex justify-center">
      <el-pagination
        :current-page="page"
        :page-size="10"
        :total="total"
        layout="prev, pager, next"
        @current-change="(p: number) => emit('page-change', p)"
      />
    </div>
  </div>
</template>
