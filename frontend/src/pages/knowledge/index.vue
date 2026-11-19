<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { listKnowledgePoints } from '@/api/knowledge'
import type { KnowledgeNode } from '@/types'
import AppIcon from '@/components/ui/AppIcon.vue'

const points = ref<KnowledgeNode[]>([])
const loading = ref(true)
const error = ref('')
const keyword = ref('')

const LINE_COLORS = ['#0E7C7B', '#1D4ED8', '#B45309', '#BE123C']

function levelColor(level: number): string {
  return LINE_COLORS[(Math.max(1, level) - 1) % LINE_COLORS.length]
}
const LEVEL_INKS = ['#0B6463', '#1839A8', '#7A4500', '#9B1032']
function levelInk(level: number): string {
  return LEVEL_INKS[(Math.max(1, level) - 1) % LEVEL_INKS.length]
}

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
  <div class="page shell">
    <header class="flex flex-wrap items-end justify-between gap-6">
      <div class="max-w-2xl">
        <h1 class="headline">知识点导航</h1>
        <p class="lede mt-2.5 max-w-xl">
          按学科组织学习内容：先看先修关系，再进入每个知识点查看课程资源。
        </p>
      </div>
      <div class="flex w-full flex-wrap items-center gap-2.5 sm:w-auto">
        <RouterLink :to="{ name: 'questionnaire' }" class="btn btn-secondary">
          <AppIcon name="target" :size="16" />
          学习问卷
        </RouterLink>
        <div class="min-w-0 flex-1 sm:w-72 sm:flex-none">
          <el-input v-model="keyword" placeholder="搜索知识点，如 线性代数、机器学习…" clearable>
            <template #prefix>
              <AppIcon name="search" :size="16" />
            </template>
          </el-input>
        </div>
      </div>
    </header>

    <div v-if="loading" class="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <div v-for="i in 6" :key="i" class="card p-5">
        <div class="skeleton h-5 w-1/2"></div>
        <div class="skeleton mt-3 h-3.5 w-full"></div>
        <div class="skeleton mt-2 h-3.5 w-3/4"></div>
        <div class="skeleton mt-5 h-3.5 w-2/3"></div>
      </div>
    </div>

    <div v-else-if="error" class="state-box mt-8">
      <span class="state-icon !bg-danger-soft !text-danger">
        <AppIcon name="alert" :size="22" />
      </span>
      <p class="state-title">知识点加载失败</p>
      <p class="state-copy">{{ error }}</p>
      <button type="button" class="btn btn-secondary btn-sm mt-2" @click="load">重新加载</button>
    </div>

    <div v-else-if="filtered.length === 0" class="state-box mt-8">
      <span class="state-icon">
        <AppIcon name="search" :size="22" />
      </span>
      <p class="state-title">没有匹配的知识点</p>
      <p class="state-copy">换个关键词试试，或清空搜索后浏览全部层级。</p>
      <button type="button" class="btn btn-secondary btn-sm mt-2" @click="keyword = ''">清空搜索</button>
    </div>

    <template v-else>
      <section v-for="group in grouped" :key="group.level" class="mt-10 first:mt-8">
        <div class="flex flex-wrap items-center gap-2.5">
          <span
            class="flex h-7 min-w-7 items-center justify-center rounded-full px-2 text-[11px] font-bold"
            :style="{ backgroundColor: `color-mix(in srgb, ${levelColor(group.level)} 12%, white)`, color: levelInk(group.level) }"
          >L{{ group.level }}</span>
          <h2 class="text-[18px] font-bold text-ink">第 {{ group.level }} 层 · {{ group.label }}</h2>
          <span class="text-[12px] text-ink-muted">
            {{ group.level === 1 ? '无先修要求' : '需先完成前一层' }}
          </span>
          <span class="tnum ml-auto text-[12px] text-ink-muted">{{ group.items.length }} 个</span>
        </div>

        <div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <RouterLink
            v-for="point in group.items"
            :key="point.id"
            :to="{ name: 'knowledge-detail', params: { code: point.code } }"
            class="group card interactive flex flex-col p-5"
          >
            <div class="flex items-start justify-between gap-3">
              <h3 class="text-[15px] font-bold leading-snug text-ink transition-colors group-hover:text-primary">
                {{ point.name }}
              </h3>
              <AppIcon name="arrow-up-right" :size="15" class="mt-0.5 shrink-0 text-ink-muted transition-colors group-hover:text-primary" />
            </div>
            <p class="mt-1.5 line-clamp-2 text-[12.5px] leading-relaxed text-ink-secondary">
              {{ point.description || '暂无描述' }}
            </p>
            <div class="tnum mt-4 flex items-center gap-3 text-[11.5px] text-ink-muted">
              <span>课程资源 {{ point.resource_count }}</span>
              <span class="h-3 w-px bg-border"></span>
              <span>概念 {{ point.concept_count }}</span>
            </div>
          </RouterLink>
        </div>
      </section>
    </template>
  </div>
</template>
