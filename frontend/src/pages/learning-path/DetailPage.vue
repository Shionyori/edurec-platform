<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { completePathStep, getLearningPath, getPathResources } from '@/api/learning'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import type { LearningPath, PathStep, Resource } from '@/types'

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
  <div class="mx-auto max-w-5xl px-4 py-8 sm:px-6 sm:py-10">
    <RouterLink :to="{ name: 'knowledge' }" class="text-sm text-ink-secondary underline-offset-4 hover:text-primary hover:underline">
      ← 返回知识点导航
    </RouterLink>

    <div v-if="loading" class="mt-6">
      <el-skeleton :rows="8" animated />
    </div>

    <div v-else-if="error" class="mt-6 flex flex-col items-center gap-3 rounded-[4px] border border-border bg-surface px-6 py-16 text-center">
      <p class="text-sm text-danger">{{ error }}</p>
      <el-button @click="load">重新加载</el-button>
    </div>

    <template v-else-if="path">
      <header class="mt-4 rounded-[4px] bg-rail px-6 py-7 text-white">
        <div class="flex flex-wrap items-end justify-between gap-6">
          <div class="max-w-2xl">
            <span
              class="line-chip"
              :style="{ '--chip': path.status === 'completed' ? 'var(--color-success)' : 'var(--color-primary)' }"
            >{{ path.status === 'completed' ? '已全部完成' : '进行中' }}</span>
            <h1 class="mt-3 text-2xl font-bold tracking-tight sm:text-3xl">
              {{ path.target_name }} · 学习路径
            </h1>
            <p class="mt-2 text-sm leading-relaxed text-white/70">
              按先修关系逐节解锁；已掌握的内容会跳过，只推荐路径上还没掌握的部分。
            </p>
          </div>
          <div class="data-num flex w-full items-end justify-between gap-4 sm:block sm:w-auto sm:text-right">
            <p class="text-4xl font-bold leading-none">{{ progress }}<span class="text-lg font-semibold text-white/60">%</span></p>
            <p class="text-xs text-white/60 sm:mt-2">已完成 {{ doneCount }} / {{ path.steps.length }} 节</p>
          </div>
        </div>
        <div class="mt-5 h-1.5 overflow-hidden rounded-full bg-white/15" role="progressbar" :aria-valuenow="progress" aria-valuemin="0" aria-valuemax="100">
          <div class="h-full rounded-full bg-primary transition-[width] duration-500" :style="{ width: `${progress}%` }"></div>
        </div>
      </header>

      <section class="mt-6 rounded-[4px] border border-border bg-surface p-5">
        <div class="flex flex-wrap items-end justify-between gap-2 border-b-2 pb-3" style="border-color: var(--color-line-teal)">
          <div class="flex items-center gap-3">
            <span class="route-node" style="--node: var(--color-line-teal)" aria-hidden="true"></span>
            <h2 class="text-lg font-bold text-ink">线路站点</h2>
          </div>
          <span class="text-xs text-ink-muted">按先修顺序排列，逐节解锁</span>
        </div>

        <ol class="relative mt-4 pl-7">
          <span class="absolute bottom-3 left-[5px] top-3 w-[2px] bg-border" aria-hidden="true"></span>

          <li v-for="(step, index) in path.steps" :key="step.code + index" class="relative pb-3 last:pb-0">
            <span
              class="route-node absolute -left-7 top-4"
              :class="{ 'route-node--filled': step.status !== 'locked' }"
              :style="{ '--node': stepColor(step.status) }"
              aria-hidden="true"
            ></span>

            <div
              class="rounded-[4px] border px-4 py-3 transition-colors"
              :class="step.status === 'current'
                ? 'border-primary/40 bg-primary/[0.04]'
                : 'border-border'"
            >
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div class="flex min-w-0 items-center gap-3">
                  <span class="data-num w-6 shrink-0 text-xs font-semibold text-ink-muted">
                    {{ String(index + 1).padStart(2, '0') }}
                  </span>
                  <div class="min-w-0">
                    <RouterLink
                      v-if="step.status !== 'locked'"
                      :to="{ name: 'knowledge-detail', params: { code: step.code } }"
                      class="text-sm font-semibold text-ink underline-offset-4 hover:text-primary hover:underline"
                    >{{ step.name }}</RouterLink>
                    <span v-else class="text-sm font-medium text-ink-muted">{{ step.name }}</span>
                    <span class="ml-2 text-xs text-ink-muted">第 {{ step.level }} 层</span>
                  </div>
                </div>

                <div class="flex items-center gap-3">
                  <span class="data-num hidden text-xs text-ink-muted sm:inline">掌握度 {{ Math.round(step.mastery * 100) }}%</span>
                  <span class="data-num hidden text-xs text-ink-muted md:inline">资源 {{ step.resource_count }}</span>
                  <span
                    class="rounded-[3px] px-2 py-0.5 text-[11px] font-semibold"
                    :class="step.status === 'done'
                      ? 'bg-success/10 text-success'
                      : step.status === 'current'
                        ? 'bg-primary/10 text-primary-active'
                        : 'bg-bg text-ink-muted'"
                  >{{ statusLabels[step.status] }}</span>
                </div>
              </div>

              <div class="mt-2 flex items-center gap-3 pl-9">
                <div class="h-1 w-full max-w-40 overflow-hidden rounded-full bg-border">
                  <div
                    class="h-full rounded-full"
                    :style="{ width: `${Math.round(step.mastery * 100)}%`, background: stepColor(step.status) }"
                  ></div>
                </div>
                <span class="data-num text-[11px] text-ink-muted sm:hidden">掌握度 {{ Math.round(step.mastery * 100) }}%</span>
              </div>
            </div>
          </li>
        </ol>
      </section>

      <div
        v-if="path.status === 'completed'"
        class="mt-6 flex flex-wrap items-center justify-between gap-4 rounded-[4px] border border-border bg-surface px-6 py-5"
      >
        <div>
          <p class="text-sm font-semibold text-ink">这条学习路径已经学完</p>
          <p class="mt-1 text-xs text-ink-muted">可以回到知识点导航，选择一个更深入的目标继续生成路径。</p>
        </div>
        <RouterLink :to="{ name: 'knowledge' }"><el-button type="primary">选择下一个目标</el-button></RouterLink>
      </div>

      <section v-else class="mt-8">
        <div class="flex flex-wrap items-end justify-between gap-3 border-b-2 pb-3" style="border-color: var(--color-primary)">
          <div>
            <div class="flex items-center gap-3">
              <span class="route-node" style="--node: var(--color-primary)" aria-hidden="true"></span>
              <h2 class="text-xl font-bold text-ink">当前一节</h2>
            </div>
            <p v-if="currentPointName" class="mt-1 text-sm text-ink-secondary">{{ currentPointName }}</p>
          </div>
          <el-button type="primary" :loading="completing" @click="completeCurrent">
            学完本节，解锁下一节
          </el-button>
        </div>

        <div v-if="resources.length > 0" class="mt-5 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
          <ResourceCard v-for="resource in resources" :key="resource.id" :resource="resource" />
        </div>
        <div v-else class="mt-5 flex flex-col items-center gap-3 rounded-[4px] border border-dashed border-border px-6 py-12 text-center">
          <p class="text-sm text-ink-secondary">
            本节暂无课程资源，可以先到搜索页按关键词采集，或直接用上方的「学完本节，解锁下一节」继续。
          </p>
          <RouterLink :to="{ name: 'search' }"><el-button>去搜索资源</el-button></RouterLink>
        </div>
      </section>
    </template>
  </div>
</template>
