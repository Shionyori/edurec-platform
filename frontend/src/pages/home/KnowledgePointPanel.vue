<script setup lang="ts">
import type { LearningPlanGroup } from '@/types'
import ResourceCard from '@/components/resource/ResourceCard.vue'

const props = defineProps<{ group: LearningPlanGroup }>()

// 兜底分组的 knowledge_point.id 为 0，用名字保证 name 唯一
const panelName = props.group.knowledge_point.id
  ? String(props.group.knowledge_point.id)
  : props.group.knowledge_point.name
</script>

<template>
  <el-collapse-item :name="panelName">
    <template #title>
      <div class="flex flex-1 items-center justify-between pr-3">
        <span class="font-medium text-ink">{{ group.knowledge_point.name }}</span>
        <span class="text-xs text-ink-muted">{{ group.total }} 个资源</span>
      </div>
    </template>

    <div v-if="group.resources.length === 0" class="py-4 text-sm text-ink-muted">
      这个知识点暂时没有可推荐的资源
    </div>
    <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <ResourceCard v-for="r in group.resources" :key="r.id" :resource="r" />
    </div>

    <RouterLink
      v-if="group.total > group.resources.length"
      :to="{ name: 'search', query: { keyword: group.knowledge_point.name } }"
      class="mt-4 inline-block text-sm text-primary hover:underline"
    >
      去资源库查看全部 {{ group.total }} 个
    </RouterLink>
  </el-collapse-item>
</template>
