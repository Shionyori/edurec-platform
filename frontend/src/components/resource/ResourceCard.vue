<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Resource } from '@/types'
import { recordBehavior } from '@/api/behavior'
import AppIcon from '@/components/ui/AppIcon.vue'

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
      return { label: '课程', color: 'var(--color-line-teal)', icon: 'book' }
    case 'video':
      return { label: '视频', color: 'var(--color-line-blue)', icon: 'play' }
    default:
      return { label: '文章', color: 'var(--color-line-amber)', icon: 'book-open' }
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
    class="group card interactive flex h-full flex-col overflow-hidden"
    @click="onClick"
  >
    <div class="relative">
      <img
        v-if="resource.cover_url && !coverFailed"
        :src="resource.cover_url"
        :alt="resource.title"
        referrerpolicy="no-referrer"
        loading="lazy"
        class="aspect-video w-full object-cover transition-transform duration-[420ms] ease-[cubic-bezier(.22,1,.36,1)] group-hover:scale-[1.03]"
        @error="coverFailed = true"
      />
      <div
        v-else
        class="cover-fallback aspect-video w-full"
        aria-hidden="true"
      >
        <span class="cover-fallback__letter">{{ resource.title.charAt(0) }}</span>
        <AppIcon name="spark" :size="22" class="cover-fallback__spark" />
      </div>

      <span class="chip absolute left-3 top-3 bg-surface/95 !text-[11.5px] shadow-sm">
        <span class="dot" :style="{ color: typeMeta.color }" aria-hidden="true"></span>
        {{ typeMeta.label }}
      </span>
    </div>

    <div class="flex flex-1 flex-col p-4 sm:p-[18px]">
      <div class="flex items-center justify-between gap-3">
        <span class="truncate text-[12px] font-medium text-ink-muted">
          {{ resource.category?.name || '未分类' }}
        </span>
        <span v-if="resource.avg_rating > 0" class="tnum flex shrink-0 items-center gap-1 text-[12px] font-semibold text-ink-secondary">
          <AppIcon name="star-filled" :size="12" class="text-[#d99a20]" />
          {{ resource.avg_rating.toFixed(1) }}
        </span>
      </div>

      <h3 class="card-title mt-1.5 line-clamp-2 text-ink transition-colors duration-150 group-hover:text-primary">
        {{ resource.title }}
      </h3>
      <p v-if="resource.description" class="mt-1.5 line-clamp-2 text-[13px] leading-relaxed text-ink-secondary">
        {{ resource.description }}
      </p>

      <p
        v-if="reasonText"
        class="mt-3 flex items-start gap-1.5 rounded-[9px] bg-primary-softer px-2.5 py-2 text-[12px] leading-relaxed text-primary-active"
      >
        <AppIcon name="spark" :size="13" class="mt-0.5 shrink-0" />
        <span class="line-clamp-2">{{ reasonText }}</span>
      </p>

      <div class="mt-auto flex items-center justify-between gap-3 border-t border-border pt-3 text-[12px] text-ink-muted">
        <span class="tnum">{{ resource.view_count }} 次浏览</span>
        <span class="flex items-center gap-1 font-semibold text-ink-secondary transition-colors group-hover:text-primary">
          查看
          <AppIcon name="arrow-right" :size="13" class="transition-transform duration-200 group-hover:translate-x-0.5" />
        </span>
      </div>
    </div>
  </RouterLink>
</template>
