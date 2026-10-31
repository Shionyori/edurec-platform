<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getKnowledgePoint, startConceptGeneration, getConceptGenerationStatus } from '@/api/knowledge'
import { createLearningPath } from '@/api/learning'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import KnowledgeGraph from '@/components/knowledge/KnowledgeGraph.vue'
import type { KnowledgeDetail, KnowledgeNode } from '@/types'

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

onMounted(load)
watch(code, load)
</script>

<template>
  <div class="mx-auto max-w-7xl px-6 py-10">
    <RouterLink
      :to="detail?.point.discipline_code
        ? { name: 'discipline', params: { code: detail.point.discipline_code } }
        : { name: 'knowledge' }"
      class="text-sm text-primary hover:underline"
    >
      ← {{ detail?.point.discipline_code ? '返回所属学科' : '返回学科导航' }}
    </RouterLink>

    <div v-if="loading" class="mt-8">
      <el-skeleton :rows="8" animated />
    </div>
    <el-empty v-else-if="error" :description="error" class="mt-16" />

    <template v-else-if="detail">
      <header class="mt-4">
        <div class="flex flex-wrap items-center gap-3">
          <h1 class="text-2xl font-bold text-ink">{{ detail.point.name }}</h1>
          <el-tag size="small" type="primary">第 {{ detail.point.level }} 层</el-tag>
          <span class="text-xs text-ink-muted">
            课程资源 {{ detail.point.resource_count }} · 概念 {{ detail.point.concept_count }}
          </span>
        </div>
        <p class="mt-2 max-w-3xl text-sm leading-relaxed text-ink-secondary">
          {{ detail.point.description || '暂无描述' }}
        </p>
        <div class="mt-4 flex flex-wrap items-center gap-3">
          <el-button type="primary" round :loading="creatingPath" @click="startLearningPath">
            生成我的学习路径
          </el-button>
          <RouterLink :to="{ name: 'questionnaire' }">
            <el-button round>调整学习问卷</el-button>
          </RouterLink>
          <span class="text-xs text-ink-muted">
            路径会按先修关系跳过你已掌握的内容，学完一节才解锁下一节
          </span>
        </div>
      </header>

      <section class="mt-8 rounded-lg border border-border bg-surface p-5">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h2 class="text-base font-semibold text-ink">先修关系</h2>
          <div class="flex items-center gap-3">
            <span class="text-xs text-ink-muted">箭头表示「先学 → 后学」，点击节点可切换</span>
            <el-switch v-model="showAllNodes" size="small" active-text="显示全部学科" />
          </div>
        </div>
        <KnowledgeGraph
          class="mt-4"
          :nodes="goalGraph.nodes"
          :edges="goalGraph.edges"
          :current-id="detail.point.id"
          @select="openNode"
        />
      </section>

      <section class="mt-6 rounded-lg border border-border bg-surface p-5">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h2 class="text-base font-semibold text-ink">概念图谱</h2>
          <div class="flex items-center gap-3">
            <span class="text-xs text-ink-muted">
              来自 EMBEDD-ER{{ concepts.length ? ` · 共 ${concepts.length} 个概念` : '' }}
            </span>
            <el-button size="small" round :loading="regenerating" @click="regenerateConcepts">
              重新生成概念
            </el-button>
          </div>
        </div>
        <p v-if="generationNotice" class="mt-2 text-xs text-primary">{{ generationNotice }}</p>

        <template v-if="visibleConcepts.length > 0">
          <div class="mt-4 flex flex-wrap gap-2">
            <el-tooltip
              v-for="concept in visibleConcepts"
              :key="concept.id"
              :content="concept.explanation || `${concept.name}（${concept.field}）`"
              placement="top"
            >
              <el-tag :type="concept.source === 'generated' ? 'success' : 'info'" effect="plain">
                {{ concept.name }}
                <span class="ml-1 text-[11px] text-ink-muted">
                  {{ concept.source === 'generated' ? '生成' : `L${concept.level}` }}
                </span>
              </el-tag>
            </el-tooltip>
          </div>
          <p v-if="hiddenConceptCount > 0" class="mt-3 text-xs text-ink-muted">
            另有 {{ hiddenConceptCount }} 个概念未展示
          </p>
          <p class="mt-3 text-xs text-ink-muted">
            概念先修边 {{ conceptEdges.length }} 条；同一层级的概念可以并行学习。
          </p>
        </template>
        <el-empty
          v-else
          description="该学科暂未匹配到概念层（MOOCCube 未覆盖）；可点击「重新生成概念」尝试补齐，或先按学科先修顺序学习"
          :image-size="80"
          class="mt-2"
        />
      </section>

      <section class="mt-6">
        <div class="flex items-end justify-between">
          <h2 class="text-lg font-semibold text-ink">课程资源</h2>
          <span class="text-xs text-ink-muted">共 {{ detail.resource_total }} 条</span>
        </div>

        <div
          v-if="resources.length > 0"
          class="mt-4 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3"
        >
          <ResourceCard
            v-for="resource in resources"
            :key="resource.id"
            :resource="resource"
          />
        </div>
        <el-empty
          v-else
          description="该知识点下暂无资源，可先到搜索页按关键词采集"
          class="mt-4"
        />
      </section>
    </template>
  </div>
</template>
