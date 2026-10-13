<script setup lang="ts">
import { ref, watch } from 'vue'
import { getLearningPlan } from '@/api/learningPlan'
import { recordImpressions } from '@/api/impression'
import KnowledgePointPanel from './KnowledgePointPanel.vue'
import type { LearningPlan, LearningPlanGroup } from '@/types'

const props = defineProps<{ goal: string }>()
const emit = defineEmits<{ reset: [] }>()

const plan = ref<LearningPlan | null>(null)
const loading = ref(false)
const error = ref('')

// 请求代次：目标快速变更时，先发的慢响应不得覆盖后发的结果
let generation = 0

async function load(goal: string) {
  const current = ++generation
  loading.value = true
  error.value = ''
  try {
    const data = await getLearningPlan(goal)
    if (current !== generation) return // 已被更新的请求取代，丢弃
    plan.value = data
    // 面板展开即曝光：记录本屏知识点下的资源。用独立场景区分入口。
    // 注意：这里**不能**说「不影响首页 CTR」——后台看板的 CTR 分母不过滤 scene
    // （repository/stats.go 统计全部 resource_impressions），新增本场景会让看板 CTR 下降。
    // 失败静默，埋点不影响页面。
    const shown = data.groups.flatMap((g) => g.resources.map((r) => r.id))
    // 无可见资源时跳过：避免一次空曝光调用（服务端没有任何资源可记录）
    if (shown.length > 0) recordImpressions('learning-plan', shown).catch(() => {})
  } catch (e) {
    if (current !== generation) return
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    if (current === generation) loading.value = false
  }
}

watch(
  () => props.goal,
  (goal) => {
    if (goal) load(goal)
  },
  { immediate: true },
)

// 与 KnowledgePointPanel 的面板 name 保持一致（id 为 0 的兜底组用名字）
function panelKey(group: LearningPlanGroup): string {
  return group.knowledge_point.id ? String(group.knowledge_point.id) : group.knowledge_point.name
}
</script>

<template>
  <div>
    <div v-if="loading" class="mt-8 space-y-3">
      <el-skeleton :rows="4" animated />
    </div>

    <div v-else-if="error" class="py-24 text-center text-sm">
      <p class="text-red-500">{{ error }}</p>
      <el-button class="mt-4" @click="load(props.goal)">重试</el-button>
    </div>

    <template v-else-if="plan">
      <p class="mt-6 text-sm text-ink-secondary">
        <template v-if="plan.matched">
          为「{{ plan.goal }}」规划了 {{ plan.groups.length }} 个知识点，展开可查看推荐资源
        </template>
        <template v-else>没有找到对应知识点，以下是按目标搜索到的资源</template>
      </p>

      <!-- 组内资源为空时仍要展示知识点本身（这是本功能的核心产出），
           面板内各自提示无资源；只有整份计划没有任何分组才走空态 -->
      <el-empty v-if="plan.groups.length === 0" description="暂无相关资源" class="mt-8" />
      <!-- el-collapse 的 model-value 是「展开项 name 的数组」，默认展开第一个知识点 -->
      <el-collapse v-else class="mt-4" :model-value="plan.groups.length ? [panelKey(plan.groups[0])] : []">
        <KnowledgePointPanel v-for="g in plan.groups" :key="panelKey(g)" :group="g" />
      </el-collapse>

      <div class="mt-6">
        <el-button text @click="emit('reset')">换一个目标</el-button>
      </div>
    </template>
  </div>
</template>
