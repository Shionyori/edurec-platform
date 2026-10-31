<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getLearningProfile, submitQuestionnaire } from '@/api/learning'
import type { LearningProfileItem } from '@/types'

const router = useRouter()

const items = ref<LearningProfileItem[]>([])
const answers = ref<Record<string, number>>({})
const loading = ref(true)
const saving = ref(false)
const error = ref('')

const levelLabels = ['没接触过', '了解概念', '学过并练习', '熟练']
const levelOptions = levelLabels.map((label, value) => ({ label, value }))

const LINE_COLORS = ['#0E7C7B', '#1D4ED8', '#B45309', '#BE123C']
function levelColor(level: number): string {
  return LINE_COLORS[(Math.max(1, level) - 1) % LINE_COLORS.length]
}

const grouped = computed(() => {
  const map = new Map<number, LearningProfileItem[]>()
  for (const item of items.value) {
    const level = Math.max(1, item.level || 1)
    const group = map.get(level) ?? []
    group.push(item)
    map.set(level, group)
  }
  return [...map.entries()]
    .sort(([a], [b]) => a - b)
    .map(([level, list]) => ({ level, items: list }))
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    items.value = await getLearningProfile()
    const initial: Record<string, number> = {}
    for (const item of items.value) {
      initial[item.point_code] = Math.min(3, Math.max(0, Math.round(item.mastery * 3)))
    }
    answers.value = initial
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await submitQuestionnaire(answers.value)
    ElMessage.success('已保存，学习路径会按你的水平跳过已掌握内容')
    router.push({ name: 'knowledge' })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-4xl px-4 py-8 sm:px-6 sm:py-10">
    <header class="flex flex-wrap items-start justify-between gap-4 border-b border-border pb-5">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-ink sm:text-3xl">初始学习问卷</h1>
        <p class="mt-2 max-w-2xl text-sm leading-relaxed text-ink-secondary">
          按你的真实水平自评每个知识点，系统会据此跳过已掌握内容，只推荐路径上还没掌握的部分。以后可以随时回来重做。
        </p>
      </div>
      <RouterLink :to="{ name: 'knowledge' }">
        <el-button>先去看知识点</el-button>
      </RouterLink>
    </header>

    <div v-if="loading" class="mt-6">
      <el-skeleton :rows="8" animated />
    </div>
    <div v-else-if="error" class="mt-6 flex flex-col items-center gap-3 rounded-[4px] border border-border bg-surface px-6 py-16 text-center">
      <p class="text-sm text-danger">{{ error }}</p>
      <el-button @click="load">重新加载</el-button>
    </div>

    <template v-else>
      <p class="mt-4 text-xs text-ink-muted">0 = 没接触，3 = 熟练；掌握度 ≥ 80% 的节点会从路径中跳过</p>

      <section
        v-for="group in grouped"
        :key="group.level"
        class="mt-5 overflow-hidden rounded-[4px] border border-border bg-surface"
      >
        <div class="flex flex-wrap items-center gap-3 border-b border-border bg-bg/60 px-5 py-3">
          <span class="line-chip" :style="{ '--chip': levelColor(group.level) }">L{{ group.level }}</span>
          <h2 class="text-sm font-semibold text-ink">第 {{ group.level }} 层</h2>
          <span class="data-num text-xs text-ink-muted">{{ group.items.length }} 个知识点</span>
        </div>

        <div class="divide-y divide-border">
          <div
            v-for="item in group.items"
            :key="item.point_code"
            class="flex flex-wrap items-center justify-between gap-3 px-5 py-3 transition-colors hover:bg-bg/40"
          >
            <div class="min-w-40">
              <span class="text-sm font-medium text-ink">{{ item.name }}</span>
              <span v-if="item.source" class="ml-2 text-xs text-ink-muted">
                （当前：{{ item.source === 'behavior' ? '行为估计' : '已填写' }}）
              </span>
            </div>
            <el-radio-group v-model="answers[item.point_code]" size="small">
              <el-radio-button
                v-for="option in levelOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </el-radio-button>
            </el-radio-group>
          </div>
        </div>
      </section>

      <div class="sticky bottom-4 mt-6 flex flex-wrap items-center justify-between gap-3 rounded-[4px] border border-border bg-surface px-5 py-3 shadow-[0_10px_30px_-18px_rgba(11,27,43,0.45)]">
        <span class="data-num text-xs text-ink-muted">共 {{ items.length }} 个知识点</span>
        <el-button type="primary" :loading="saving" @click="save">保存并返回知识点</el-button>
      </div>
    </template>
  </div>
</template>
