<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getRecommendations } from '@/api/recommendation'
import { recordImpressions } from '@/api/impression'
import { useAuthStore } from '@/stores/auth'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import ResourceCardSkeleton from '@/components/resource/ResourceCardSkeleton.vue'
import InterestPicker from '@/components/profile/InterestPicker.vue'
import type { Resource } from '@/types'

const auth = useAuthStore()
const resources = ref<Resource[]>([])
const loading = ref(false)
const error = ref('')

// 无兴趣且无历史的新用户，展示冷启动引导
const showInterestPicker = computed(
  () => !!auth.user && (auth.user.interests?.length ?? 0) === 0,
)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await getRecommendations(12)
    resources.value = data.list
    // 列表渲染即曝光：记录本屏展示的资源（CTR 的分母）。失败静默，埋点不影响页面。
    recordImpressions('home', data.list.map((r) => r.id)).catch(() => {})
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
    <section class="relative overflow-hidden rounded-[4px] border border-border bg-surface px-6 py-8 sm:px-10 sm:py-12">
      <!-- 线路网络底纹：只做世界肌理，不承载数据 -->
      <svg
        class="pointer-events-none absolute -right-24 -top-24 hidden h-[140%] w-[62%] opacity-[0.14] lg:block"
        viewBox="0 0 640 420"
        fill="none"
        aria-hidden="true"
      >
        <path d="M-20 300H180Q230 300 230 250V90Q230 50 270 50H660" stroke="var(--color-line-teal)" stroke-width="3" />
        <path d="M660 340H430Q380 340 380 300V150Q380 110 340 110H-20" stroke="var(--color-line-blue)" stroke-width="3" />
        <path d="M120 -20V160Q120 200 160 200H660" stroke="var(--color-line-amber)" stroke-width="3" />
        <path d="M520 -20V90Q520 130 480 130H-20" stroke="var(--color-line-rose)" stroke-width="3" />
        <circle cx="230" cy="250" r="7" fill="var(--color-surface)" stroke="var(--color-line-teal)" stroke-width="3" />
        <circle cx="380" cy="300" r="7" fill="var(--color-surface)" stroke="var(--color-line-blue)" stroke-width="3" />
        <circle cx="160" cy="200" r="7" fill="var(--color-surface)" stroke="var(--color-line-amber)" stroke-width="3" />
        <circle cx="480" cy="130" r="7" fill="var(--color-surface)" stroke="var(--color-line-rose)" stroke-width="3" />
      </svg>

      <div class="relative z-10 flex flex-col gap-8 lg:flex-row lg:items-end lg:justify-between">
        <div class="max-w-2xl">
          <h1 class="text-3xl font-bold leading-[1.1] tracking-tight text-ink sm:text-5xl">
            下一站，<span class="text-primary">学什么？</span>
          </h1>
          <p class="mt-4 max-w-xl text-sm leading-relaxed text-ink-secondary sm:text-base">
            从知识点与先修关系出发，跳过已经掌握的，只推荐你当前该学的一节。
          </p>
          <div class="mt-6 flex flex-wrap items-center gap-3">
            <RouterLink :to="{ name: 'knowledge' }">
              <el-button type="primary" size="large">选一个学习目标</el-button>
            </RouterLink>
            <RouterLink :to="{ name: 'questionnaire' }">
              <el-button size="large">调整学习问卷</el-button>
            </RouterLink>
          </div>
        </div>

        <div class="w-full max-w-sm border-t border-border pt-4 lg:w-80 lg:border-l lg:border-t-0 lg:pl-8 lg:pt-0">
          <p class="text-sm font-semibold text-ink">一条线路，三步走完</p>
          <ol class="route-strip mt-5" style="--track: var(--color-ink)" aria-label="学习流程示意">
            <li class="route-strip__stop">
              <span class="route-strip__dot"></span>
              <span class="route-strip__label">选目标</span>
            </li>
            <li class="route-strip__stop">
              <span class="route-strip__dot"></span>
              <span class="route-strip__label">生成路径</span>
            </li>
            <li class="route-strip__stop">
              <span class="route-strip__dot"></span>
              <span class="route-strip__label">完成本节</span>
            </li>
          </ol>
          <p class="mt-4 text-xs leading-relaxed text-ink-muted">
            路径会按先修关系跳过已掌握内容，学完一节才解锁下一节。
          </p>
        </div>
      </div>

      <!-- 线路母题：品牌动作，不承载数据 -->
      <div class="pointer-events-none absolute bottom-0 left-0 right-0 hidden items-center gap-0 lg:flex" aria-hidden="true">
        <span class="h-[2px] flex-1 bg-line-teal/40"></span>
        <span class="mx-2 h-2.5 w-2.5 rounded-full border-[3px] border-line-teal"></span>
        <span class="h-[2px] w-24 bg-line-blue/40"></span>
        <span class="mx-2 h-2.5 w-2.5 rounded-full border-[3px] border-line-blue"></span>
        <span class="h-[2px] w-16 bg-line-amber/40"></span>
        <span class="mx-2 h-2.5 w-2.5 rounded-full border-[3px] border-primary bg-primary"></span>
        <span class="h-[2px] flex-1 bg-line-rose/40"></span>
      </div>
    </section>

    <InterestPicker v-if="showInterestPicker" class="mt-6" @saved="load" />

    <section class="mt-10">
      <div class="flex flex-wrap items-end justify-between gap-3 border-b-2 pb-3" style="border-color: var(--color-line-teal)">
        <div class="flex items-center gap-3">
          <span class="route-node" style="--node: var(--color-line-teal)" aria-hidden="true"></span>
          <h2 class="text-xl font-bold text-ink">本站推荐</h2>
          <span v-if="resources.length" class="data-num text-xs text-ink-muted">
            为你挑选 {{ resources.length }} 条
          </span>
        </div>
        <RouterLink :to="{ name: 'search' }" class="text-sm font-medium text-ink-secondary underline-offset-4 hover:text-primary hover:underline">
          查看全部资源 →
        </RouterLink>
      </div>

      <div v-if="loading" class="mt-6 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
        <ResourceCardSkeleton v-for="i in 6" :key="i" />
      </div>

      <div v-else-if="error" class="mt-6 flex flex-col items-center gap-3 rounded-[4px] border border-border bg-surface px-6 py-16 text-center">
        <p class="text-sm text-danger">{{ error }}</p>
        <el-button @click="load">重新加载</el-button>
      </div>

      <div v-else-if="resources.length === 0" class="mt-6 flex flex-col items-center gap-4 rounded-[4px] border border-dashed border-border px-6 py-16 text-center">
        <p class="text-sm font-medium text-ink">还没有可推荐的内容</p>
        <p class="text-xs text-ink-muted">先到搜索页按关键词采集资源，或浏览知识点导航。</p>
        <div class="flex gap-3">
          <RouterLink :to="{ name: 'search' }"><el-button type="primary">去搜索资源</el-button></RouterLink>
          <RouterLink :to="{ name: 'knowledge' }"><el-button>浏览知识点</el-button></RouterLink>
        </div>
      </div>

      <div v-else class="mt-6 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
        <ResourceCard
          v-for="(r, i) in resources"
          :key="r.id"
          :resource="r"
          show-reason-fallback
          :style="{ animationDelay: `${Math.min(i, 8) * 30}ms` }"
        />
      </div>
    </section>
  </div>
</template>
