<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getKnowledgePoint, startConceptGeneration, getConceptGenerationStatus } from '@/api/knowledge'
import { createLearningPath } from '@/api/learning'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import KnowledgeGraph from '@/components/knowledge/KnowledgeGraph.vue'
import type { KnowledgeConcept, KnowledgeDetail, KnowledgeNode } from '@/types'
import AppIcon from '@/components/ui/AppIcon.vue'

const route = useRoute()
const router = useRouter()
const code = computed(() => String(route.params.code ?? ''))

const detail = ref<KnowledgeDetail | null>(null)
const loading = ref(true)
const error = ref('')
const creatingPath = ref(false)
const regenerating = ref(false)
const generationNotice = ref('')
// 列表字段兜底为空数组：后端历史版本在无数据时可能返回 null，直接 .length 会让整页渲染崩溃。
const nodes = computed(() => detail.value?.nodes ?? [])
const edges = computed(() => detail.value?.edges ?? [])
const resources = computed(() => detail.value?.resources ?? [])
const concepts = computed(() => detail.value?.concepts ?? [])
const conceptEdges = computed(() => detail.value?.concept_edges ?? [])
// 默认只显示「当前目标」的先修子图：目标必须是唯一末端；
// 打开后展示全部学科（含旁支），用于浏览整张图。
const showAllNodes = ref(false)

// 窄屏用紧凑节点尺寸，配合 SVG 自适应缩放到卡片宽度
const isNarrow = ref(false)
let narrowQuery: MediaQueryList | null = null
function syncNarrow(event?: MediaQueryListEvent) {
  isNarrow.value = event ? event.matches : (narrowQuery?.matches ?? false)
}

const LINE_COLORS = ['#0E7C7B', '#1D4ED8', '#B45309', '#BE123C']
function levelColor(level: number): string {
  return LINE_COLORS[(Math.max(1, level) - 1) % LINE_COLORS.length]
}

function conceptColor(concept: KnowledgeConcept): string {
  return concept.source === 'generated' ? 'var(--color-line-teal)' : levelColor(concept.level)
}

const VISIBLE_CONCEPTS = 30
const visibleConcepts = computed(() => concepts.value.slice(0, VISIBLE_CONCEPTS))
const hiddenConceptCount = computed(() =>
  Math.max(0, concepts.value.length - visibleConcepts.value.length),
)

// 目标导向子图：从当前知识点沿先修边反向取闭包，只保留「先修 → 目标」的节点。
// 这样图的最右端（末端）只有当前这一个学习目标，而不是整张图里的多个同级学科。
const goalGraph = computed(() => {
  const allNodes = nodes.value
  const allEdges = edges.value
  const targetId = detail.value?.point.id
  if (showAllNodes.value || !targetId) {
    return { nodes: allNodes, edges: allEdges }
  }

  const prereqs = new Map<number, number[]>()
  for (const edge of allEdges) {
    const list = prereqs.get(edge.to) ?? []
    list.push(edge.from)
    prereqs.set(edge.to, list)
  }

  const closure = new Set<number>()
  const stack = [targetId]
  while (stack.length > 0) {
    const current = stack.pop()!
    if (closure.has(current)) continue
    closure.add(current)
    for (const parent of prereqs.get(current) ?? []) stack.push(parent)
  }

  return {
    nodes: allNodes.filter((node) => closure.has(node.id)),
    edges: allEdges.filter((edge) => closure.has(edge.from) && closure.has(edge.to)),
  }
})

async function load() {
  if (!code.value) return
  loading.value = true
  error.value = ''
  try {
    detail.value = await getKnowledgePoint(code.value, 12)
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
    detail.value = null
  } finally {
    loading.value = false
  }
}

function openNode(node: KnowledgeNode) {
  if (node.code === code.value) return
  router.push({ name: 'knowledge-detail', params: { code: node.code } })
}

async function startLearningPath() {
  if (!detail.value) return
  creatingPath.value = true
  try {
    const path = await createLearningPath(detail.value.point.code)
    router.push({ name: 'learning-path', params: { id: path.id } })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '生成学习路径失败')
  } finally {
    creatingPath.value = false
  }
}

// 重新生成概念：调用 engine（平台只发 HTTP），轮询到完成后刷新详情
async function regenerateConcepts() {
  if (!detail.value) return
  const pointCode = detail.value.point.code
  regenerating.value = true
  generationNotice.value = ''
  try {
    let status = await startConceptGeneration(pointCode)
    let attempts = 0
    while (status.status === 'running' && attempts < 60) {
      await new Promise((resolve) => setTimeout(resolve, 2000))
      status = await getConceptGenerationStatus(pointCode)
      attempts += 1
    }
    if (status.status === 'done') {
      generationNotice.value = `已生成 ${status.generated} 个概念（共 ${status.total} 个）`
      await load()
    } else if (status.status === 'failed') {
      generationNotice.value = `生成失败：${status.error || 'engine 未启动'}`
    }
  } catch (e) {
    generationNotice.value = e instanceof Error ? e.message : '生成失败'
  } finally {
    regenerating.value = false
  }
}

onMounted(() => {
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    narrowQuery = window.matchMedia('(max-width: 640px)')
    syncNarrow()
    narrowQuery.addEventListener('change', syncNarrow)
  }
  load()
})

onBeforeUnmount(() => {
  narrowQuery?.removeEventListener('change', syncNarrow)
})

watch(code, load)
</script>
<template>
  <div class="page shell">
    <RouterLink
      :to="detail?.point.discipline_code
        ? { name: 'discipline', params: { code: detail.point.discipline_code } }
        : { name: 'knowledge' }"
      class="link-quiet inline-flex items-center gap-1.5 text-[12.5px]"
    >
      <AppIcon name="arrow-left" :size="14" />
      {{ detail?.point.discipline_code ? '返回所属学科' : '返回学科导航' }}
    </RouterLink>

    <div v-if="loading" class="mt-6">
      <h1 class="sr-only">知识点详情</h1>
      <div class="skeleton h-52 w-full !rounded-[16px]"></div>
      <div class="mt-6 skeleton h-72 w-full !rounded-[16px]"></div>
    </div>

    <div v-else-if="error" class="mt-6">
      <h1 class="headline">知识点详情</h1>
      <div class="state-box mt-4">
      <span class="state-icon !bg-danger-soft !text-danger">
        <AppIcon name="alert" :size="22" />
      </span>
      <p class="state-title">知识点加载失败</p>
      <p class="state-copy">{{ error }}</p>
      <button type="button" class="btn btn-secondary btn-sm mt-2" @click="load">重新加载</button>
      </div>
    </div>

    <template v-else-if="detail">
      <header class="dark-panel mt-4 px-6 py-7 sm:px-8 sm:py-9">
        <svg
          class="pointer-events-none absolute -right-20 -top-28 h-[340px] w-[340px] text-white/10"
          viewBox="0 0 340 340"
          fill="none"
          aria-hidden="true"
        >
          <circle cx="170" cy="170" r="126" stroke="currentColor" stroke-width="1" />
          <circle cx="170" cy="170" r="158" stroke="currentColor" stroke-width="1" stroke-dasharray="2 8" />
          <circle cx="296" cy="170" r="5" fill="#c9451f" />
        </svg>

        <div class="relative flex flex-wrap items-end justify-between gap-6">
          <div class="max-w-3xl">
            <div class="flex flex-wrap items-center gap-2">
              <span
                class="flex h-6 min-w-6 items-center justify-center rounded-full px-2 text-[11px] font-bold"
                :style="{ backgroundColor: `color-mix(in srgb, ${levelColor(detail.point.level)} 22%, transparent)`, color: '#fff' }"
              >L{{ detail.point.level }}</span>
              <span class="text-[12px] text-white/55">第 {{ detail.point.level }} 层</span>
            </div>
            <h1 class="mt-3 text-[26px] font-extrabold leading-tight tracking-[-0.03em] text-white sm:text-[34px]">
              {{ detail.point.name }}
            </h1>
            <p class="mt-2 max-w-2xl text-[13.5px] leading-relaxed text-white/65">
              {{ detail.point.description || '暂无描述' }}
            </p>
            <div class="tnum mt-5 flex flex-wrap gap-x-6 gap-y-2 text-[12px] text-white/60">
              <span>课程资源 <strong class="font-semibold text-white">{{ detail.point.resource_count }}</strong></span>
              <span>概念 <strong class="font-semibold text-white">{{ detail.point.concept_count }}</strong></span>
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-2.5">
            <el-button type="primary" :loading="creatingPath" @click="startLearningPath">
              <AppIcon name="route" :size="15" class="mr-1.5" />
              生成我的学习路径
            </el-button>
            <RouterLink :to="{ name: 'questionnaire' }">
              <el-button>调整学习问卷</el-button>
            </RouterLink>
          </div>
        </div>
      </header>

      <p class="mt-3 text-[12px] text-ink-muted">
        路径会按先修关系跳过你已掌握的内容，学完一节才解锁下一节。
      </p>

      <section class="card mt-6 p-5 sm:p-6">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex items-center gap-2.5">
            <span class="flex h-8 w-8 items-center justify-center rounded-[9px] bg-primary-soft text-primary">
              <AppIcon name="route" :size="16" />
            </span>
            <div>
              <h2 class="text-[16px] font-bold text-ink">先修关系</h2>
              <p class="mt-0.5 text-[11.5px] text-ink-muted">箭头表示「先学 → 后学」，点击节点可切换</p>
            </div>
          </div>
          <el-switch v-model="showAllNodes" size="small" aria-label="显示全部学科" active-text="显示全部学科" />
        </div>

        <KnowledgeGraph
          class="mt-5"
          :nodes="goalGraph.nodes"
          :edges="goalGraph.edges"
          :current-id="detail.point.id"
          :compact="isNarrow"
          @select="openNode"
        />
        <p v-if="goalGraph.nodes.length === 0" class="state-box mt-4 !border-0 !bg-surface-2 !py-10">
          <span class="state-icon !h-10 !w-10">
            <AppIcon name="route" :size="18" />
          </span>
          <span class="state-title">暂无先修关系</span>
          <span class="state-copy">这条知识点是独立起点，可以直接开始学习。</span>
        </p>
      </section>

      <section class="card mt-4 p-5 sm:p-6">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-2.5">
            <span class="flex h-8 w-8 items-center justify-center rounded-[9px] bg-sunken text-ink-secondary">
              <AppIcon name="layers" :size="16" />
            </span>
            <div>
              <h2 class="text-[16px] font-bold text-ink">概念图谱</h2>
              <p class="mt-0.5 text-[11.5px] text-ink-muted">
                来自 EMBEDD-ER{{ concepts.length ? ` · 共 ${concepts.length} 个概念` : '' }}
              </p>
            </div>
          </div>
          <el-button size="small" :loading="regenerating" @click="regenerateConcepts">
            <AppIcon name="refresh" :size="14" class="mr-1.5" />
            重新生成概念
          </el-button>
        </div>

        <p v-if="generationNotice" class="mt-3 text-[12.5px] font-semibold text-primary">{{ generationNotice }}</p>

        <template v-if="visibleConcepts.length > 0">
          <div class="mt-4 flex flex-wrap gap-2">
            <el-tooltip
              v-for="concept in visibleConcepts"
              :key="concept.id"
              :content="concept.explanation || `${concept.name}（${concept.field}）`"
              placement="top"
            >
              <span class="inline-flex cursor-default items-center gap-1.5 rounded-full border border-border bg-surface-2 px-2.5 py-1 text-[12px] font-medium text-ink">
                <span class="h-1.5 w-1.5 rounded-full" :style="{ background: conceptColor(concept) }" aria-hidden="true"></span>
                {{ concept.name }}
                <span class="tnum text-[10px] text-ink-muted">
                  {{ concept.source === 'generated' ? '生成' : `L${concept.level}` }}
                </span>
              </span>
            </el-tooltip>
          </div>
          <p v-if="hiddenConceptCount > 0" class="mt-3 text-[12px] text-ink-muted">
            另有 {{ hiddenConceptCount }} 个概念未展示
          </p>
          <p class="mt-3 text-[12px] text-ink-muted">
            概念先修边 {{ conceptEdges.length }} 条；同一层级的概念可以并行学习。
          </p>
        </template>
        <div v-else class="state-box mt-4 !py-10">
          <span class="state-icon !h-10 !w-10">
            <AppIcon name="layers" :size="18" />
          </span>
          <p class="state-title">暂未匹配到概念层</p>
          <p class="state-copy">
            该学科暂未匹配到概念层（MOOCCube 未覆盖）；可点击「重新生成概念」尝试补齐，或先按学科先修顺序学习。
          </p>
        </div>
      </section>

      <section class="mt-10">
        <div class="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 class="section-title">课程资源</h2>
            <p class="mt-1.5 text-[13px] text-ink-muted">与这个知识点关联的课程、视频与文章</p>
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
          <p class="state-title">暂无资源</p>
          <p class="state-copy">该知识点下暂无资源，可先到搜索页按关键词采集。</p>
          <RouterLink :to="{ name: 'search' }" class="btn btn-primary btn-sm mt-2">去搜索资源</RouterLink>
        </div>
      </section>
    </template>
  </div>
</template>
