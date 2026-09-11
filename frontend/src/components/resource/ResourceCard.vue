<script setup lang="ts">
import { ref } from 'vue'
import type { Resource } from '@/types'

defineProps<{ resource: Resource }>()

// 外链封面（如 B 站）可能失效：no-referrer 已规避防盗链，但仍需对加载失败做降级
const coverFailed = ref(false)
</script>

<template>
  <RouterLink
    :to="{ name: 'resource-detail', params: { id: resource.id } }"
    class="group block overflow-hidden rounded-lg border border-border bg-surface shadow-sm transition hover:shadow-md"
  >
    <img
      v-if="resource.cover_url && !coverFailed"
      :src="resource.cover_url"
      :alt="resource.title"
      referrerpolicy="no-referrer"
      loading="lazy"
      class="aspect-video w-full object-cover"
      @error="coverFailed = true"
    />
    <div
      v-else-if="resource.cover_url"
      class="flex aspect-video w-full items-center justify-center bg-bg text-2xl font-semibold text-ink-muted"
    >
      {{ resource.title.charAt(0) }}
    </div>

    <div class="p-5">
      <div class="flex items-center justify-between text-xs text-ink-muted">
        <el-tag
          size="small"
          :type="resource.type === 'course' ? 'primary' : resource.type === 'video' ? 'success' : 'warning'"
        >
          {{ resource.type }}
        </el-tag>
        <span>{{ resource.category?.name }}</span>
      </div>
      <h3 class="mt-3 text-lg font-semibold text-ink">{{ resource.title }}</h3>
      <p class="mt-1 line-clamp-2 text-sm text-ink-secondary">{{ resource.description }}</p>
      <div class="mt-4 flex items-center justify-between border-t border-border pt-3 text-sm">
        <span class="text-amber-500">★ {{ resource.avg_rating.toFixed(1) }}</span>
        <span class="text-xs text-ink-muted">{{ resource.view_count }} 次浏览</span>
      </div>
    </div>
  </RouterLink>
</template>
