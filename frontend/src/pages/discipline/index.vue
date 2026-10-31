<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listDisciplines } from '@/api/discipline'
import type { Discipline } from '@/types'

const items = ref<Discipline[]>([])
const loading = ref(true)
const error = ref('')

const LINE_COLORS = ['#0E7C7B', '#1D4ED8', '#B45309', '#BE123C']

function lineColor(index: number): string {
  return LINE_COLORS[index % LINE_COLORS.length]
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    items.value = await listDisciplines()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 sm:py-10">
    <div class="flex flex-wrap items-end justify-between gap-4 border-b border-border pb-5">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-ink sm:text-3xl">学科导航</h1>
        <p class="mt-2 max-w-2xl text-sm leading-relaxed text-ink-secondary">
          先选学科，再进入学科页查看知识点与课程资源；没有资源的学科会自动去 B 站抓取。
        </p>
      </div>
      <RouterLink :to="{ name: 'questionnaire' }">
        <el-button>学习问卷</el-button>
      </RouterLink>
    </div>

    <div v-if="loading" class="mt-7 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <div v-for="i in 9" :key="i" class="rounded-[4px] border border-border bg-surface p-5">
        <el-skeleton :rows="3" animated />
      </div>
    </div>

    <div v-else-if="error" class="mt-7 flex flex-col items-center gap-3 rounded-[4px] border border-border bg-surface px-6 py-16 text-center">
      <p class="text-sm text-danger">{{ error }}</p>
      <el-button @click="load">重新加载</el-button>
    </div>

    <div v-else-if="items.length === 0" class="mt-7 rounded-[4px] border border-dashed border-border px-6 py-16 text-center">
      <p class="text-sm font-medium text-ink">暂无学科</p>
      <p class="mt-1 text-xs text-ink-muted">学科数据准备好后会显示在这里。</p>
    </div>

    <div v-else class="mt-7 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <RouterLink
        v-for="(item, index) in items"
        :key="item.code"
        :to="{ name: 'discipline', params: { code: item.code } }"
        class="group flex flex-col rounded-[4px] border border-border bg-surface p-5 transition-all duration-200 hover:-translate-y-0.5 hover:border-ink/25 hover:shadow-[0_10px_30px_-18px_rgba(11,27,43,0.45)]"
        :aria-label="`进入${item.name}学科，${item.point_count} 个知识点`"
      >
        <div class="flex items-start justify-between gap-3">
          <span class="line-chip" :style="{ '--chip': lineColor(index) }">{{ item.code.toUpperCase() }}</span>
          <span
            v-if="item.resource_count === 0"
            class="rounded-[3px] bg-warning/10 px-2 py-0.5 text-[11px] font-semibold text-warning"
          >自动抓取</span>
        </div>

        <h2 class="mt-3 text-lg font-semibold text-ink transition-colors group-hover:text-primary">
          {{ item.name }}
        </h2>
        <p class="mt-1.5 line-clamp-2 text-sm leading-relaxed text-ink-secondary">
          {{ item.description }}
        </p>

        <div class="mt-auto flex items-end justify-between gap-3 border-t border-border pt-4">
          <div class="data-num flex gap-4 text-xs text-ink-muted">
            <span>知识点 <strong class="font-semibold text-ink">{{ item.point_count }}</strong></span>
            <span>资源 <strong class="font-semibold text-ink">{{ item.resource_count }}</strong></span>
          </div>
          <span class="flex items-center gap-1.5 text-xs font-semibold text-ink-secondary transition-colors group-hover:text-primary">
            进入线路
            <svg viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="none" aria-hidden="true">
              <path d="M5 12h13M13 6l6 6-6 6" stroke="currentColor" stroke-width="2" stroke-linecap="square" />
            </svg>
          </span>
        </div>
      </RouterLink>
    </div>
  </div>
</template>
