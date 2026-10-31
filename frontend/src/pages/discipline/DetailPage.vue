<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getDiscipline, getDisciplineCrawlStatus, startDisciplineCrawl } from '@/api/discipline'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import type { DisciplineDetail, DisciplineCrawlStatus } from '@/types'

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
  <div class="mx-auto max-w-7xl px-6 py-10">
    <RouterLink :to="{ name: 'knowledge' }" class="text-sm text-primary hover:underline">
      ← 返回学科导航
    </RouterLink>

    <div v-if="loading" class="mt-8">
      <el-skeleton :rows="8" animated />
    </div>
    <el-empty v-else-if="error" :description="error" class="mt-16" />

    <template v-else-if="detail">
      <header class="mt-4 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 class="text-2xl font-bold text-ink">{{ detail.discipline.name }}</h1>
          <p class="mt-2 max-w-3xl text-sm text-ink-secondary">{{ detail.discipline.description }}</p>
          <div class="mt-2 flex gap-4 text-xs text-ink-muted">
            <span>知识点 {{ detail.discipline.point_count }}</span>
            <span>课程资源 {{ detail.resource_total }}</span>
          </div>
        </div>
        <el-button
          v-if="detail.resource_total === 0"
          round
          :loading="crawling"
          @click="autoCrawl"
        >重新抓取资源</el-button>
      </header>

      <el-alert
        v-if="crawlNotice"
        class="mt-4"
        :type="crawling ? 'info' : detail.resource_total > 0 ? 'success' : 'warning'"
        :closable="false"
        :title="crawlNotice"
      />

      <section class="mt-8">
        <h2 class="text-lg font-semibold text-ink">知识点</h2>
        <p v-if="points.some((point) => point.concept_count === 0)" class="mt-1 text-xs text-ink-muted">
          标有「概念待生成」的知识点尚无概念图谱，进入详情页点击「重新生成概念」即可尝试补齐。
        </p>
        <div v-if="points.length > 0" class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <RouterLink
            v-for="point in points"
            :key="point.id"
            :to="{ name: 'knowledge-detail', params: { code: point.code } }"
            class="rounded-lg border border-border bg-surface p-4 transition hover:border-primary hover:shadow-sm"
          >
            <h3 class="text-sm font-semibold text-ink">{{ point.name }}</h3>
            <p class="mt-1 line-clamp-2 text-xs text-ink-secondary">{{ point.description }}</p>
            <div class="mt-3 flex flex-wrap items-center gap-3 text-xs text-ink-muted">
              <span>资源 {{ point.resource_count }}</span>
              <span>概念 {{ point.concept_count }}</span>
              <el-tag
                v-if="point.concept_count === 0"
                size="small"
                type="info"
                effect="plain"
              >概念待生成</el-tag>
            </div>
          </RouterLink>
        </div>
        <el-empty v-else description="暂无知识点" class="mt-4" />
      </section>

      <section class="mt-8">
        <div class="flex items-end justify-between">
          <h2 class="text-lg font-semibold text-ink">课程资源</h2>
          <span class="text-xs text-ink-muted">共 {{ detail.resource_total }} 条</span>
        </div>
        <div v-if="resources.length > 0" class="mt-4 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          <ResourceCard v-for="resource in resources" :key="resource.id" :resource="resource" />
        </div>
        <el-empty
          v-else
          :description="crawling ? '正在自动抓取资源…' : '该学科暂无资源，可点击上方按钮抓取'"
          class="mt-4"
        />
      </section>
    </template>
  </div>
</template>
