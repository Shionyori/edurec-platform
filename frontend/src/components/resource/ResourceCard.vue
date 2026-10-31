<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Resource } from '@/types'
import { recordBehavior } from '@/api/behavior'

const props = defineProps<{ resource: Resource; showReasonFallback?: boolean }>()

// 外链封面（如 B 站）可能失效：no-referrer 已规避防盗链，但仍需对加载失败做降级
const coverFailed = ref(false)

// 推荐流里没有个性化理由时给出诚实的兜底说明；普通列表不显示理由。
const reasonText = computed(() => {
  if (props.resource.reason) return props.resource.reason
  return props.showReasonFallback ? '为你推荐' : ''
})

const typeMeta = computed(() => {
  switch (props.resource.type) {
    case 'course':
      return { label: '课程', color: 'var(--color-line-teal)' }
    case 'video':
      return { label: '视频', color: 'var(--color-line-blue)' }
    default:
      return { label: '文章', color: 'var(--color-line-amber)' }
  }
})

// 点击卡片 = 从列表（推荐/搜索）点进详情，是推荐的「点击转化」信号（CTR 的分子）。
// 上报失败静默：埋点不能阻断导航。
function onClick() {
  recordBehavior(props.resource.id, 'click').catch(() => {})
}
</script>

<template>
  <RouterLink
    :to="{ name: 'resource-detail', params: { id: resource.id } }"
    class="group fade-up block overflow-hidden rounded-[4px] border border-border bg-surface transition-all duration-200 ease-out hover:-translate-y-0.5 hover:border-ink/25 hover:shadow-[0_10px_30px_-18px_rgba(11,27,43,0.45)]"
    @click="onClick"
  >
    <div class="relative">
      <img
        v-if="resource.cover_url && !coverFailed"
        :src="resource.cover_url"
        :alt="resource.title"
        referrerpolicy="no-referrer"
        loading="lazy"
        class="aspect-video w-full object-cover transition-transform duration-300 ease-out group-hover:scale-[1.03]"
        @error="coverFailed = true"
      />
      <div
        v-else
        class="relative flex aspect-video w-full items-center justify-center overflow-hidden bg-bg"
        aria-hidden="true"
      >
        <span class="absolute left-0 right-0 top-1/2 h-[2px] bg-border"></span>
        <span class="route-node relative z-10" style="--node: var(--color-line-teal)"></span>
        <span class="ml-3 text-2xl font-bold text-ink/20">{{ resource.title.charAt(0) }}</span>
      </div>
    </div>

    <div class="p-4 sm:p-5">
      <div class="flex items-center justify-between gap-3">
        <span class="line-chip" :style="{ '--chip': typeMeta.color }">{{ typeMeta.label }}</span>
        <span class="truncate text-xs text-ink-muted">{{ resource.category?.name }}</span>
      </div>

      <h3 class="mt-3 line-clamp-2 text-base font-semibold leading-snug text-ink transition-colors duration-150 group-hover:text-primary">
        {{ resource.title }}
      </h3>
      <p v-if="resource.description" class="mt-1.5 line-clamp-2 text-sm leading-relaxed text-ink-secondary">
        {{ resource.description }}
      </p>

      <p v-if="reasonText" class="mt-3 flex items-start gap-2 text-xs leading-relaxed text-line-teal">
        <span class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-current" aria-hidden="true"></span>
        <span>{{ reasonText }}</span>
      </p>

      <div class="mt-4 flex items-center justify-between border-t border-border pt-3 text-xs text-ink-muted">
        <span v-if="resource.avg_rating > 0" class="data-num flex items-center gap-1.5">
          <svg viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="currentColor" aria-hidden="true">
            <path d="M12 3.5l2.6 5.3 5.9.9-4.3 4.1 1 5.8-5.2-2.7-5.2 2.7 1-5.8L3.5 9.7l5.9-.9L12 3.5z" />
          </svg>
          {{ resource.avg_rating.toFixed(1) }}
        </span>
        <span v-else>暂无评分</span>
        <span class="data-num">{{ resource.view_count }} 次浏览</span>
      </div>
    </div>
  </RouterLink>
</template>
