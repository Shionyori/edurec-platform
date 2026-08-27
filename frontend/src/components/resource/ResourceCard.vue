<script setup lang="ts">
import type { Resource } from '@/types'

defineProps<{ resource: Resource }>()
</script>

<template>
  <RouterLink
    :to="{ name: 'resource-detail', params: { id: resource.id } }"
    class="group block rounded-lg border border-border bg-surface p-5 shadow-sm transition hover:shadow-md"
  >
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
  </RouterLink>
</template>
