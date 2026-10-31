<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getDiscipline, getDisciplineCrawlStatus, startDisciplineCrawl } from '@/api/discipline'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import type { DisciplineDetail, DisciplineCrawlStatus, KnowledgeNode } from '@/types'

const route = useRoute()
const code = computed(() => String(route.params.code ?? ''))

const detail = ref<DisciplineDetail | null>(null)
// 列表字段兜底为空数组：后端历史版本在无数据时返回 null，直接 .length 会让整页渲染崩溃。
const points = computed(() => detail.value?.points ?? [])
const resources = computed(() => detail.value?.resources ?? [])
const loading = ref(true)
const error = ref('')
const crawling = ref(false)
const crawlNotice = ref('')
let crawlAttempted = false

const LINE_COLORS = ['#0E7C7B', '#1D4ED8', '#B45309', '#BE123C']
function levelColor(level: number): string {
  return LINE_COLORS[(Math.max(1, level) - 1) % LINE_COLORS.length]
}

// 知识点按真实层级分组：层级即线路站序
const levelGroups = computed(() => {
  const groups = new Map<number, KnowledgeNode[]>()
  for (const point of points.value) {
    const level = Math.max(1, point.level || 1)
    const list = groups.get(level) ?? []
    list.push(point)
    groups.set(level, list)
  }
  return [...groups.entries()]
    .sort(([a], [b]) => a - b)
    .map(([level, items]) => ({ level, items: [...items].sort((a, b) => a.id - b.id) }))
})

async function load() {
  if (!code.value) return
  loading.value = true
  error.value = ''
  try {
    detail.value = await getDiscipline(code.value, 12)
    // 学科没有资源：自动触发一次 B 站抓取（只在首次进入时触发）
    if (detail.value.resource_total === 0 && !crawlAttempted) {
      crawlAttempted = true
      autoCrawl()
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
    detail.value = null
  } finally {
    loading.value = false
  }
}

async function autoCrawl() {
  crawling.value = true
  crawlNotice.value = '该学科暂无资源，正在自动从 B 站抓取…'
  try {
    let status: DisciplineCrawlStatus = await startDisciplineCrawl(code.value)
    let attempts = 0
    while (status.status === 'running' && attempts < 40) {
      await new Promise((resolve) => setTimeout(resolve, 3000))
      status = await getDisciplineCrawlStatus(code.value)
      attempts += 1
    }
    if (status.status === 'done') {
      crawlNotice.value = `已自动补充 ${status.imported} 条资源（关联 ${status.linked} 条）`
      ElMessage.success(crawlNotice.value)
      await load()
    } else {
      crawlNotice.value = `自动抓取失败：${status.error || '请稍后重试'}`
      ElMessage.error(crawlNotice.value)
    }
  } catch (e) {
    crawlNotice.value = e instanceof Error ? e.message : '自动抓取失败'
  } finally {
    crawling.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 sm:py-10">
    <RouterLink :to="{ name: 'knowledge' }" class="text-sm text-ink-secondary underline-offset-4 hover:text-primary hover:underline">
      ← 返回学科导航
    </RouterLink>

    <div v-if="loading" class="mt-6">
      <el-skeleton :rows="8" animated />
    </div>

    <div v-else-if="error" class="mt-6 flex flex-col items-center gap-3 rounded-[4px] border border-border bg-surface px-6 py-16 text-center">
      <p class="text-sm text-danger">{{ error }}</p>
      <el-button @click="load">重新加载</el-button>
    </div>

    <template v-else-if="detail">
      <header class="mt-4 flex flex-wrap items-end justify-between gap-6 rounded-[4px] bg-rail px-6 py-7 text-white">
        <div class="max-w-3xl">
          <span class="line-chip" :style="{ '--chip': 'rgba(255,255,255,0.14)' }">
            {{ detail.discipline.code.toUpperCase() }}
          </span>
          <h1 class="mt-3 text-2xl font-bold tracking-tight sm:text-3xl">{{ detail.discipline.name }}</h1>
          <p class="mt-2 text-sm leading-relaxed text-white/70">{{ detail.discipline.description }}</p>
          <div class="data-num mt-3 flex gap-5 text-xs text-white/60">
            <span>知识点 <strong class="font-semibold text-white">{{ detail.discipline.point_count }}</strong></span>
            <span>课程资源 <strong class="font-semibold text-white">{{ detail.resource_total }}</strong></span>
          </div>
        </div>
        <el-button v-if="detail.resource_total === 0" :loading="crawling" @click="autoCrawl">
          {{ crawling ? '正在抓取…' : '重新抓取资源' }}
        </el-button>
      </header>

      <div
        v-if="crawlNotice"
        class="mt-4 flex items-start gap-3 rounded-[4px] border border-border bg-surface px-4 py-3 text-sm text-ink-secondary"
      >
        <span
          class="route-node mt-0.5"
          :style="{ '--node': crawling ? 'var(--color-line-amber)' : detail.resource_total > 0 ? 'var(--color-success)' : 'var(--color-line-amber)' }"
          aria-hidden="true"
        ></span>
        <span>{{ crawlNotice }}</span>
      </div>

      <section class="mt-8">
        <div class="flex flex-wrap items-end justify-between gap-2 border-b-2 pb-3" style="border-color: var(--color-line-teal)">
          <div class="flex items-center gap-3">
            <span class="route-node" style="--node: var(--color-line-teal)" aria-hidden="true"></span>
            <h2 class="text-xl font-bold text-ink">知识点线路</h2>
          </div>
          <span class="text-xs text-ink-muted">按层级从左到右、由基础到进阶</span>
        </div>

        <p v-if="points.some((point) => point.concept_count === 0)" class="mt-3 text-xs text-ink-muted">
          标有「概念待生成」的知识点尚无概念图谱，进入详情页点击「重新生成概念」即可尝试补齐。
        </p>

        <div v-if="levelGroups.length > 0" class="relative mt-4 pl-6">
          <!-- 纵向线路：连接每个层级，不承载数据 -->
          <span class="absolute bottom-2 left-[5px] top-2 w-[2px] bg-border" aria-hidden="true"></span>

          <div v-for="group in levelGroups" :key="group.level" class="relative pb-5 last:pb-0">
            <div
              class="flex items-center gap-3 rounded-[3px] px-2 py-1"
              :style="{ backgroundColor: `${levelColor(group.level)}0F` }"
            >
              <span
                class="route-node absolute -left-6 top-1.5"
                :style="{ '--node': levelColor(group.level) }"
                aria-hidden="true"
              ></span>
              <span class="line-chip" :style="{ '--chip': levelColor(group.level) }">L{{ group.level }}</span>
              <span class="text-xs text-ink-muted">{{ group.items.length }} 个知识点</span>
            </div>

            <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              <RouterLink
                v-for="point in group.items"
                :key="point.id"
                :to="{ name: 'knowledge-detail', params: { code: point.code } }"
                class="group flex flex-col rounded-[4px] border border-border bg-surface p-4 transition-all duration-200 hover:-translate-y-0.5 hover:border-ink/25 hover:shadow-[0_10px_24px_-18px_rgba(11,27,43,0.45)]"
              >
                <div class="flex items-start justify-between gap-2">
                  <h3 class="text-sm font-semibold text-ink transition-colors group-hover:text-primary">
                    {{ point.name }}
                  </h3>
                  <svg viewBox="0 0 24 24" class="mt-0.5 h-4 w-4 shrink-0 text-ink-muted transition-colors group-hover:text-primary" fill="none" aria-hidden="true">
                    <path d="M5 12h13M13 6l6 6-6 6" stroke="currentColor" stroke-width="2" stroke-linecap="square" />
                  </svg>
                </div>
                <p class="mt-1 line-clamp-2 text-xs leading-relaxed text-ink-secondary">{{ point.description }}</p>
                <div class="data-num mt-3 flex flex-wrap items-center gap-3 text-[11px] text-ink-muted">
                  <span>资源 {{ point.resource_count }}</span>
                  <span>概念 {{ point.concept_count }}</span>
                  <span
                    v-if="point.concept_count === 0"
                    class="rounded-[3px] bg-bg px-1.5 py-0.5 font-semibold text-ink-secondary"
                  >概念待生成</span>
                </div>
              </RouterLink>
            </div>
          </div>
        </div>

        <div v-else class="mt-4 rounded-[4px] border border-dashed border-border px-6 py-12 text-center">
          <p class="text-sm font-medium text-ink">暂无知识点</p>
          <p class="mt-1 text-xs text-ink-muted">该学科的知识点尚未建立，可先浏览课程资源。</p>
        </div>
      </section>

      <section class="mt-10">
        <div class="flex flex-wrap items-end justify-between gap-2 border-b-2 pb-3" style="border-color: var(--color-line-blue)">
          <div class="flex items-center gap-3">
            <span class="route-node" style="--node: var(--color-line-blue)" aria-hidden="true"></span>
            <h2 class="text-xl font-bold text-ink">课程资源</h2>
          </div>
          <span class="data-num text-xs text-ink-muted">共 {{ detail.resource_total }} 条</span>
        </div>
        <div v-if="resources.length > 0" class="mt-5 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
          <ResourceCard v-for="resource in resources" :key="resource.id" :resource="resource" />
        </div>
        <div v-else class="mt-5 flex flex-col items-center gap-3 rounded-[4px] border border-dashed border-border px-6 py-12 text-center">
          <p class="text-sm font-medium text-ink">
            {{ crawling ? '正在自动抓取资源…' : '该学科暂无资源' }}
          </p>
          <div class="flex gap-3">
            <el-button v-if="!crawling" @click="autoCrawl">重新抓取</el-button>
            <RouterLink :to="{ name: 'search' }"><el-button>去搜索资源</el-button></RouterLink>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
