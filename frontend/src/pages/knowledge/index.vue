<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { listKnowledgePoints } from '@/api/knowledge'
import type { KnowledgeNode } from '@/types'

const points = ref<KnowledgeNode[]>([])
const loading = ref(true)
const error = ref('')
const keyword = ref('')

const levelLabels: Record<number, string> = {
  1: '基础',
  2: '进阶',
  3: '专业',
  4: '研究',
}

function levelLabel(level: number): string {
  return levelLabels[level] ?? `第 ${level} 层`
}

const filtered = computed(() => {
  const kw = keyword.value.trim()
  if (!kw) return points.value
  return points.value.filter(
    (point) => point.name.includes(kw) || point.description.includes(kw),
  )
})

// 按层级分组：同层知识点可以并排学，层级表示先后顺序
const grouped = computed(() => {
  const map = new Map<number, KnowledgeNode[]>()
  for (const point of filtered.value) {
    const level = Math.max(1, point.level || 1)
    const group = map.get(level) ?? []
    group.push(point)
    map.set(level, group)
  }
  return [...map.entries()]
    .sort(([a], [b]) => a - b)
    .map(([level, items]) => ({
      level,
      label: levelLabel(level),
      items: [...items].sort((a, b) => a.id - b.id),
    }))
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    points.value = await listKnowledgePoints()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-7xl px-6 py-10">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold">知识点导航</h1>
        <p class="mt-2 text-sm text-ink-secondary">
          按学科组织学习内容：先看先修关系，再进入每个知识点查看课程资源。
        </p>
      </div>
      <div class="flex items-center gap-2">
        <RouterLink :to="{ name: 'questionnaire' }">
          <el-button round>学习问卷</el-button>
        </RouterLink>
        <el-input
          v-model="keyword"
          placeholder="搜索知识点，如 线性代数、机器学习…"
          clearable
          class="w-72"
        />
      </div>
    </div>

    <div v-if="loading" class="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <el-skeleton v-for="i in 6" :key="i" :rows="3" animated />
    </div>
    <div v-else-if="error" class="py-24 text-center text-sm text-danger">{{ error }}</div>
    <el-empty v-else-if="filtered.length === 0" description="没有匹配的知识点" class="mt-8" />

    <template v-else>
      <section v-for="group in grouped" :key="group.level" class="mt-10">
        <div class="flex items-center gap-3">
          <h2 class="text-lg font-semibold text-ink">
            第 {{ group.level }} 层 · {{ group.label }}
          </h2>
          <span class="text-xs text-ink-muted">
            {{ group.level === 1 ? '无先修要求' : '需先完成前一层' }}
          </span>
        </div>

        <div class="mt-4 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
          <RouterLink
            v-for="point in group.items"
            :key="point.id"
            :to="{ name: 'knowledge-detail', params: { code: point.code } }"
            class="group rounded-lg border border-border bg-surface p-5 transition hover:border-primary hover:shadow-md"
          >
            <div class="flex items-start justify-between gap-3">
              <h3 class="text-base font-semibold text-ink group-hover:text-primary">
                {{ point.name }}
              </h3>
              <el-tag size="small" type="primary">L{{ point.level }}</el-tag>
            </div>
            <p class="mt-2 line-clamp-2 text-sm text-ink-secondary">
              {{ point.description || '暂无描述' }}
            </p>
            <div class="mt-4 flex items-center gap-4 text-xs text-ink-muted">
              <span>课程资源 {{ point.resource_count }}</span>
              <span>概念 {{ point.concept_count }}</span>
            </div>
          </RouterLink>
        </div>
      </section>
    </template>
  </div>
</template>
