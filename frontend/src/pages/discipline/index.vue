<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listDisciplines } from '@/api/discipline'
import type { Discipline } from '@/types'
import AppIcon from '@/components/ui/AppIcon.vue'

const items = ref<Discipline[]>([])
const loading = ref(true)
const error = ref('')

const LINE_COLORS = ['#0F7A66', '#2F62C4', '#A26800', '#B84A62']
const LINE_INKS = ['#0B6354', '#2757B5', '#7A4D00', '#913B4D']

function lineColor(index: number): string {
  return LINE_COLORS[index % LINE_COLORS.length]
}

function lineInk(index: number): string {
  return LINE_INKS[index % LINE_INKS.length]
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
  <div class="page shell">
    <header class="flex flex-wrap items-end justify-between gap-6">
      <div class="max-w-2xl">
        <h1 class="headline">学科导航</h1>
        <p class="lede mt-2.5 max-w-xl">
          先选学科，再进入学科页查看知识点与课程资源；没有资源的学科会自动去 B 站抓取。
        </p>
      </div>
      <RouterLink :to="{ name: 'questionnaire' }" class="btn btn-secondary">
        <AppIcon name="target" :size="16" />
        学习问卷
      </RouterLink>
    </header>

    <div v-if="loading" class="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <div v-for="i in 9" :key="i" class="card p-5">
        <div class="skeleton h-10 w-10 !rounded-[10px]"></div>
        <div class="skeleton mt-4 h-5 w-1/2"></div>
        <div class="skeleton mt-3 h-3.5 w-full"></div>
        <div class="skeleton mt-2 h-3.5 w-4/5"></div>
        <div class="skeleton mt-5 h-3.5 w-2/3"></div>
      </div>
    </div>

    <div v-else-if="error" class="state-box mt-8">
      <span class="state-icon !bg-danger-soft !text-danger">
        <AppIcon name="alert" :size="22" />
      </span>
      <p class="state-title">学科加载失败</p>
      <p class="state-copy">{{ error }}</p>
      <button type="button" class="btn btn-secondary btn-sm mt-2" @click="load">重新加载</button>
    </div>

    <div v-else-if="items.length === 0" class="state-box mt-8">
      <span class="state-icon">
        <AppIcon name="book" :size="22" />
      </span>
      <p class="state-title">暂无学科</p>
      <p class="state-copy">学科数据准备好后会显示在这里。</p>
    </div>

    <div v-else class="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <RouterLink
        v-for="(item, index) in items"
        :key="item.code"
        :to="{ name: 'discipline', params: { code: item.code } }"
        class="group card interactive flex flex-col overflow-hidden p-5"
        :style="{ '--accent': lineColor(index) }"
        :aria-label="`进入${item.name}学科，${item.point_count} 个知识点`"
      >
        <div class="flex items-start justify-between gap-3">
          <span
            class="flex h-11 w-11 items-center justify-center rounded-[11px] text-[11px] font-extrabold tracking-[0.02em]"
            :style="{ backgroundColor: `color-mix(in srgb, ${lineColor(index)} 10%, white)`, color: lineInk(index) }"
            aria-hidden="true"
          >
            {{ item.code.toUpperCase() }}
          </span>
          <span
            v-if="item.resource_count === 0"
            class="chip !text-[11px]"
          >
            <span class="dot text-warning" aria-hidden="true"></span>
            自动抓取
          </span>
        </div>

        <h2 class="mt-4 text-[18px] font-bold tracking-[-0.02em] text-ink transition-colors group-hover:text-primary">
          {{ item.name }}
        </h2>
        <p class="mt-1.5 line-clamp-2 text-[13px] leading-relaxed text-ink-secondary">
          {{ item.description }}
        </p>

        <div class="mt-auto flex items-end justify-between gap-3 pt-5">
          <div class="tnum flex gap-4 text-[12px] text-ink-muted">
            <span>知识点 <strong class="font-semibold text-ink">{{ item.point_count }}</strong></span>
            <span>资源 <strong class="font-semibold text-ink">{{ item.resource_count }}</strong></span>
          </div>
          <span class="flex h-8 w-8 items-center justify-center rounded-full bg-surface-2 text-ink-muted transition-all duration-200 group-hover:bg-primary-soft group-hover:text-primary">
            <AppIcon name="arrow-right" :size="15" />
          </span>
        </div>
      </RouterLink>
    </div>
  </div>
</template>
