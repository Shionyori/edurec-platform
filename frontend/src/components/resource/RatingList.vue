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
    <div v-if="loading" class="py-16 text-center text-[13px] text-ink-muted">加载中…</div>
    <div v-else-if="ratings.length === 0" class="flex flex-col items-center gap-2 py-16 text-center">
      <p class="text-[14px] font-semibold text-ink">暂无评价</p>
      <p class="text-[12.5px] text-ink-muted">来抢沙发，写下第一条学习感受吧。</p>
    </div>
    <ul v-else class="divide-y divide-border">
      <li v-for="r in ratings" :key="r.id" class="flex gap-3.5 py-4">
        <el-avatar :size="36">{{ (r.user.display_name ?? r.user.username).charAt(0) }}</el-avatar>
        <div class="min-w-0 flex-1">
          <div class="flex items-center justify-between gap-3">
            <span class="text-[13px] font-semibold text-ink">{{ r.user.display_name ?? r.user.username }}</span>
            <span class="tnum text-[11.5px] text-ink-muted">{{ formatDate(r.created_at) }}</span>
          </div>
          <el-rate :model-value="r.score" disabled class="mt-1" />
          <p v-if="r.comment" class="mt-1.5 text-[13px] leading-relaxed text-ink-secondary">{{ r.comment }}</p>
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
