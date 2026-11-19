<script setup lang="ts">
import { formatUnixDate } from '@/utils/format'
import type { BilibiliComment } from '@/types'

defineProps<{
  comments: BilibiliComment[]
  loading?: boolean
}>()
</script>

<template>
  <div>
    <div v-if="loading" class="py-16 text-center text-[13px] text-ink-muted">正在抓取 B 站评论…</div>
    <div v-else-if="comments.length === 0" class="py-16 text-center text-[13px] text-ink-muted">
      评论暂不可用或暂无评论
    </div>
    <ul v-else class="divide-y divide-border">
      <li v-for="c in comments" :key="c.id" class="flex gap-3 py-4">
        <el-avatar :size="32">{{ (c.author_name || '?').charAt(0) }}</el-avatar>
        <div class="min-w-0 flex-1">
          <div class="flex items-center justify-between gap-3">
            <span class="text-[13px] font-semibold text-ink">{{ c.author_name }}</span>
            <span class="tnum text-[11.5px] text-ink-muted">{{ formatUnixDate(c.published_at) }}</span>
          </div>
          <p class="mt-1 text-[13px] leading-relaxed text-ink-secondary">{{ c.content }}</p>
          <div class="mt-1.5 flex items-center gap-3 text-[11.5px] text-ink-muted">
            <span>#{{ c.floor }} 楼</span>
            <span>👍 {{ c.like_count }}</span>
          </div>
        </div>
      </li>
    </ul>
  </div>
</template>
