<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getDiscipline, getDisciplineCrawlStatus, startDisciplineCrawl } from '@/api/discipline'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import type { DisciplineDetail, DisciplineCrawlStatus, KnowledgeNode } from '@/types'
import AppIcon from '@/components/ui/AppIcon.vue'

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

const LINE_COLORS = ['#0F7A66', '#2F62C4', '#A26800', '#B84A62']
function levelColor(level: number): string {
  return LINE_COLORS[(Math.max(1, level) - 1) % LINE_COLORS.length]
}
const LEVEL_INKS = ['#0B6354', '#2757B5', '#7A4D00', '#913B4D']
function levelInk(level: number): string {
  return LEVEL_INKS[(Math.max(1, level) - 1) % LEVEL_INKS.length]
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
  <div class="page shell">
    <RouterLink :to="{ name: 'knowledge' }" class="link-quiet inline-flex items-center gap-1.5 text-[12.5px]">
      <AppIcon name="arrow-left" :size="14" />
      返回学科导航
    </RouterLink>

    <div v-if="loading" class="mt-6">
      <h1 class="sr-only">学科详情</h1>
      <div class="skeleton h-48 w-full !rounded-[16px]"></div>
      <div class="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
        <div v-for="i in 6" :key="i" class="skeleton h-32 !rounded-[14px]"></div>
      </div>
    </div>

    <div v-else-if="error" class="mt-6">
      <h1 class="headline">学科详情</h1>
      <div class="state-box mt-4">
      <span class="state-icon !bg-danger-soft !text-danger">
        <AppIcon name="alert" :size="22" />
      </span>
      <p class="state-title">学科加载失败</p>
      <p class="state-copy">{{ error }}</p>
      <button type="button" class="btn btn-secondary btn-sm mt-2" @click="load">重新加载</button>
      </div>
    </div>

    <template v-else-if="detail">
      <header class="panel-tint mt-4 px-6 py-7 sm:px-8 sm:py-9">
        <svg
          class="pointer-events-none absolute -right-16 -top-24 h-[320px] w-[320px] text-primary/10"
          viewBox="0 0 320 320"
          fill="none"
          aria-hidden="true"
        >
          <circle cx="160" cy="160" r="118" stroke="currentColor" stroke-width="1" />
          <circle cx="160" cy="160" r="150" stroke="currentColor" stroke-width="1" stroke-dasharray="2 8" />
          <circle cx="278" cy="160" r="5" fill="var(--color-primary)" />
        </svg>

        <div class="relative flex flex-wrap items-end justify-between gap-6">
          <div class="max-w-3xl">
            <span class="chip !text-[11px]">{{ detail.discipline.code.toUpperCase() }}</span>
            <h1 class="mt-3 text-[24px] font-bold leading-tight tracking-[-0.025em] text-ink sm:text-[30px]">
              {{ detail.discipline.name }}
            </h1>
            <p class="mt-2 max-w-2xl text-[13.5px] leading-relaxed text-ink-secondary">
              {{ detail.discipline.description }}
            </p>
            <div class="tnum mt-5 flex flex-wrap gap-x-6 gap-y-2 text-[12px] text-ink-muted">
              <span>知识点 <strong class="font-semibold text-ink">{{ detail.discipline.point_count }}</strong></span>
              <span>课程资源 <strong class="font-semibold text-ink">{{ detail.resource_total }}</strong></span>
            </div>
          </div>
          <el-button v-if="detail.resource_total === 0" :loading="crawling" @click="autoCrawl">
            {{ crawling ? '正在抓取…' : '重新抓取资源' }}
          </el-button>
        </div>
      </header>

      <div
        v-if="crawlNotice"
        class="mt-4 flex items-start gap-3 rounded-[12px] border border-border bg-surface px-4 py-3 text-[13px] text-ink-secondary"
      >
        <span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-warning-soft text-warning">
          <AppIcon name="info" :size="12" />
        </span>
        <span>{{ crawlNotice }}</span>
      </div>

      <section class="mt-10">
        <div class="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 class="section-title">知识点</h2>
            <p class="mt-1.5 text-[13px] text-ink-muted">按层级由基础到进阶排列，同层可以并行学习</p>
          </div>
          <span class="tnum text-[12px] text-ink-muted">共 {{ points.length }} 个</span>
        </div>

        <p v-if="points.some((point) => point.concept_count === 0)" class="mt-3 text-[12px] text-ink-muted">
          标有「概念待生成」的知识点尚无概念图谱，进入详情页点击「重新生成概念」即可尝试补齐。
        </p>

        <div v-if="levelGroups.length > 0" class="mt-6 space-y-8">
          <div v-for="group in levelGroups" :key="group.level">
            <div class="flex items-center gap-2.5">
              <span
                class="flex h-6 min-w-6 items-center justify-center rounded-full px-2 text-[11px] font-bold"
                :style="{ backgroundColor: `color-mix(in srgb, ${levelColor(group.level)} 12%, white)`, color: levelInk(group.level) }"
              >L{{ group.level }}</span>
              <span class="text-[13px] font-bold text-ink">第 {{ group.level }} 层</span>
              <span class="tnum text-[12px] text-ink-muted">{{ group.items.length }} 个知识点</span>
            </div>

            <div class="mt-3 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <RouterLink
                v-for="point in group.items"
                :key="point.id"
                :to="{ name: 'knowledge-detail', params: { code: point.code } }"
                class="group card interactive flex flex-col p-4"
              >
                <div class="flex items-start justify-between gap-3">
                  <h3 class="text-[14.5px] font-bold leading-snug text-ink transition-colors group-hover:text-primary">
                    {{ point.name }}
                  </h3>
                  <AppIcon name="arrow-up-right" :size="15" class="mt-0.5 shrink-0 text-ink-muted transition-colors group-hover:text-primary" />
                </div>
                <p class="mt-1.5 line-clamp-2 text-[12.5px] leading-relaxed text-ink-secondary">{{ point.description }}</p>
                <div class="tnum mt-4 flex flex-wrap items-center gap-2 text-[11.5px] text-ink-muted">
                  <span>资源 {{ point.resource_count }}</span>
                  <span class="h-3 w-px bg-border"></span>
                  <span>概念 {{ point.concept_count }}</span>
                  <span
                    v-if="point.concept_count === 0"
                    class="chip chip-warning !py-0 !text-[10.5px]"
                  >概念待生成</span>
                </div>
              </RouterLink>
            </div>
          </div>
        </div>

        <div v-else class="state-box mt-5">
          <span class="state-icon">
            <AppIcon name="layers" :size="22" />
          </span>
          <p class="state-title">暂无知识点</p>
          <p class="state-copy">该学科的知识点尚未建立，可先浏览课程资源。</p>
        </div>
      </section>

      <section class="mt-12">
        <div class="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 class="section-title">课程资源</h2>
            <p class="mt-1.5 text-[13px] text-ink-muted">来自学科知识点关联的课程、视频与文章</p>
          </div>
          <span class="tnum text-[12px] text-ink-muted">共 {{ detail.resource_total }} 条</span>
        </div>

        <div v-if="resources.length > 0" class="mt-6 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
          <ResourceCard v-for="resource in resources" :key="resource.id" :resource="resource" />
        </div>
        <div v-else class="state-box mt-6">
          <span class="state-icon">
            <AppIcon name="book" :size="22" />
          </span>
          <p class="state-title">{{ crawling ? '正在自动抓取资源…' : '该学科暂无资源' }}</p>
          <p class="state-copy">可以重新抓取，或直接按关键词搜索相关资源。</p>
          <div class="mt-2 flex flex-wrap justify-center gap-2">
            <button v-if="!crawling" type="button" class="btn btn-secondary btn-sm" @click="autoCrawl">重新抓取</button>
            <RouterLink :to="{ name: 'search' }" class="btn btn-primary btn-sm">去搜索资源</RouterLink>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
