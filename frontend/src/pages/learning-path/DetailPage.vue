<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { completePathStep, getLearningPath, getPathResources } from '@/api/learning'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import type { LearningPath, PathStep, Resource } from '@/types'
import AppIcon from '@/components/ui/AppIcon.vue'

const route = useRoute()
const pathId = computed(() => Number(route.params.id))

const path = ref<LearningPath | null>(null)
const resources = ref<Resource[]>([])
const currentPointName = ref('')
const loading = ref(true)
const completing = ref(false)
const error = ref('')

const doneCount = computed(
  () => path.value?.steps.filter((step) => step.status === 'done').length ?? 0,
)
const progress = computed(() => {
  const total = path.value?.steps.length ?? 0
  return total === 0 ? 0 : Math.round((doneCount.value / total) * 100)
})

const statusLabels: Record<PathStep['status'], string> = {
  done: '已完成',
  current: '当前学习',
  locked: '未解锁',
}

function stepColor(status: PathStep['status']): string {
  if (status === 'done') return 'var(--color-success)'
  if (status === 'current') return 'var(--color-primary)'
  return 'var(--color-border)'
}

async function loadResources() {
  if (!path.value || path.value.status === 'completed') {
    resources.value = []
    currentPointName.value = ''
    return
  }
  try {
    const data = await getPathResources(pathId.value, 9)
    resources.value = data.list
    currentPointName.value = data.knowledge_point?.name ?? ''
  } catch {
    // 资源加载失败不阻断路径展示
    resources.value = []
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    path.value = await getLearningPath(pathId.value)
    await loadResources()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
    path.value = null
  } finally {
    loading.value = false
  }
}

async function completeCurrent() {
  if (!path.value) return
  completing.value = true
  try {
    path.value = await completePathStep(pathId.value, path.value.current_index)
    await loadResources()
    ElMessage.success(
      path.value.status === 'completed' ? '整条学习路径已完成' : '已完成本节，下一节已解锁',
    )
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  } finally {
    completing.value = false
  }
}

onMounted(load)
watch(pathId, load)
</script>
<template>
  <div class="page shell">
    <div class="mx-auto max-w-[1040px]">
      <RouterLink :to="{ name: 'knowledge' }" class="link-quiet inline-flex items-center gap-1.5 text-[12.5px]">
        <AppIcon name="arrow-left" :size="14" />
        返回知识点导航
      </RouterLink>

      <div v-if="loading" class="mt-6">
        <h1 class="sr-only">学习路径</h1>
        <div class="skeleton h-52 w-full !rounded-[16px]"></div>
        <div class="mt-6 skeleton h-72 w-full !rounded-[16px]"></div>
      </div>

      <div v-else-if="error" class="mt-6">
        <h1 class="headline">学习路径</h1>
        <div class="state-box mt-4">
        <span class="state-icon !bg-danger-soft !text-danger">
          <AppIcon name="alert" :size="22" />
        </span>
        <p class="state-title">学习路径加载失败</p>
        <p class="state-copy">{{ error }}</p>
        <button type="button" class="btn btn-secondary btn-sm mt-2" @click="load">重新加载</button>
        </div>
      </div>

      <template v-else-if="path">
        <header class="panel-tint mt-4 px-6 py-7 sm:px-8 sm:py-8">
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
            <div class="max-w-2xl">
              <span
                class="chip !text-[11px]"
                :class="path.status === 'completed' ? 'chip-success' : 'chip-primary'"
              >{{ path.status === 'completed' ? '已全部完成' : '进行中' }}</span>
              <h1 class="mt-3 text-[24px] font-bold leading-tight tracking-[-0.025em] text-ink sm:text-[30px]">
                {{ path.target_name }} · 学习路径
              </h1>
              <p class="mt-2 text-[13px] leading-relaxed text-ink-secondary">
                按先修关系逐节解锁；已掌握的内容会跳过，只推荐路径上还没掌握的部分。
              </p>
            </div>
            <div class="tnum flex w-full items-end justify-between gap-4 sm:block sm:w-auto sm:text-right">
              <p class="text-[40px] font-bold leading-none tracking-[-0.04em] text-ink">
                {{ progress }}<span class="text-[16px] font-semibold text-ink-muted">%</span>
              </p>
              <p class="text-[12px] text-ink-muted sm:mt-2">已完成 {{ doneCount }} / {{ path.steps.length }} 节</p>
            </div>
          </div>
          <div class="relative mt-6 h-1.5 overflow-hidden rounded-full bg-primary-soft" role="progressbar" aria-label="学习路径完成进度" :aria-valuenow="progress" aria-valuemin="0" aria-valuemax="100">
            <div class="h-full rounded-full bg-primary transition-[width] duration-700 ease-[cubic-bezier(.22,1,.36,1)]" :style="{ width: `${progress}%` }"></div>
          </div>
        </header>

        <section class="card mt-6 p-5 sm:p-6">
          <div class="flex flex-wrap items-end justify-between gap-2">
            <div>
              <h2 class="text-[18px] font-bold text-ink">学习路径</h2>
              <p class="mt-1 text-[12.5px] text-ink-muted">按先修顺序排列，逐节解锁</p>
            </div>
            <span class="tnum text-[12px] text-ink-muted">共 {{ path.steps.length }} 节</span>
          </div>

          <ol class="relative mt-6">
            <span class="absolute bottom-4 left-[15px] top-4 w-px bg-border" aria-hidden="true"></span>

            <li v-for="(step, index) in path.steps" :key="step.code + index" class="relative pb-3 pl-11 last:pb-0">
              <span
                class="absolute left-0 top-3 flex h-8 w-8 items-center justify-center rounded-full border-2 bg-surface"
                :class="step.status === 'done'
                  ? 'border-success text-success'
                  : step.status === 'current'
                    ? 'border-primary text-primary ring-4 ring-primary-soft'
                    : 'border-border-strong text-ink-muted'"
                aria-hidden="true"
              >
                <AppIcon
                  :name="step.status === 'done' ? 'check' : step.status === 'current' ? 'play' : 'lock'"
                  :size="14"
                  :stroke-width="2.2"
                />
              </span>

              <div
                class="rounded-[12px] border px-4 py-3.5"
                :class="step.status === 'current'
                  ? 'border-primary/35 bg-primary-softer'
                  : step.status === 'done'
                    ? 'border-border bg-surface-2'
                    : 'border-border bg-surface'"
              >
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div class="flex min-w-0 items-center gap-3">
                    <span class="tnum w-6 shrink-0 text-[11.5px] font-bold text-ink-muted">
                      {{ String(index + 1).padStart(2, '0') }}
                    </span>
                    <div class="min-w-0">
                      <RouterLink
                        v-if="step.status !== 'locked'"
                        :to="{ name: 'knowledge-detail', params: { code: step.code } }"
                        class="text-[14px] font-bold text-ink underline-offset-4 transition-colors hover:text-primary hover:underline"
                      >{{ step.name }}</RouterLink>
                      <span v-else class="text-[14px] font-semibold text-ink-muted">{{ step.name }}</span>
                      <span class="ml-2 text-[11.5px] text-ink-muted">第 {{ step.level }} 层</span>
                    </div>
                  </div>

                  <div class="flex items-center gap-3">
                    <span class="tnum hidden text-[11.5px] text-ink-muted sm:inline">掌握度 {{ Math.round(step.mastery * 100) }}%</span>
                    <span class="tnum hidden text-[11.5px] text-ink-muted md:inline">资源 {{ step.resource_count }}</span>
                    <span
                      class="chip !text-[10.5px]"
                      :class="step.status === 'done'
                        ? 'chip-success'
                        : step.status === 'current'
                          ? 'chip-primary'
                          : ''"
                    >{{ statusLabels[step.status] }}</span>
                  </div>
                </div>

                <div class="mt-2.5 flex items-center gap-3 pl-9">
                  <div class="h-1 w-full max-w-44 overflow-hidden rounded-full bg-sunken">
                    <div
                      class="h-full rounded-full transition-[width] duration-500"
                      :style="{ width: `${Math.round(step.mastery * 100)}%`, background: stepColor(step.status) }"
                    ></div>
                  </div>
                  <span class="tnum text-[11px] text-ink-muted sm:hidden">掌握度 {{ Math.round(step.mastery * 100) }}%</span>
                </div>
              </div>
            </li>
          </ol>
        </section>

        <div
          v-if="path.status === 'completed'"
          class="card mt-6 flex flex-wrap items-center justify-between gap-4 p-5 sm:p-6"
        >
          <div class="flex items-center gap-3">
            <span class="flex h-10 w-10 items-center justify-center rounded-[11px] bg-success-soft text-success">
              <AppIcon name="check-circle" :size="20" />
            </span>
            <div>
              <p class="text-[14px] font-bold text-ink">这条学习路径已经学完</p>
              <p class="mt-0.5 text-[12px] text-ink-muted">可以回到知识点导航，选择一个更深入的目标继续生成路径。</p>
            </div>
          </div>
          <RouterLink :to="{ name: 'knowledge' }" class="btn btn-primary btn-sm">选择下一个目标</RouterLink>
        </div>

        <section v-else class="mt-10">
          <div class="flex flex-wrap items-end justify-between gap-3">
            <div>
              <h2 class="section-title">当前一节</h2>
              <p v-if="currentPointName" class="mt-1.5 text-[13px] text-ink-secondary">{{ currentPointName }}</p>
            </div>
            <el-button type="primary" :loading="completing" @click="completeCurrent">
              <AppIcon name="check" :size="15" class="mr-1.5" />
              学完本节，解锁下一节
            </el-button>
          </div>

          <div v-if="resources.length > 0" class="mt-6 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
            <ResourceCard v-for="resource in resources" :key="resource.id" :resource="resource" />
          </div>
          <div v-else class="state-box mt-6">
            <span class="state-icon">
              <AppIcon name="book" :size="22" />
            </span>
            <p class="state-title">本节暂无课程资源</p>
            <p class="state-copy">可以先到搜索页按关键词采集，或直接用上方的「学完本节，解锁下一节」继续。</p>
            <RouterLink :to="{ name: 'search' }" class="btn btn-primary btn-sm mt-2">去搜索资源</RouterLink>
          </div>
        </section>
      </template>
    </div>
  </div>
</template>
