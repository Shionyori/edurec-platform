<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getLearningProfile, submitQuestionnaire } from '@/api/learning'
import type { LearningProfileItem } from '@/types'

const router = useRouter()

const items = ref<LearningProfileItem[]>([])
const answers = ref<Record<string, number>>({})
// 原始值用于区分「未改动」与「已改动」：朱红只留给真正的改动与保存动作
const baseline = ref<Record<string, number>>({})
const openLevels = ref<Record<number, boolean>>({})
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

const changedCount = computed(
  () => items.value.filter((item) => answers.value[item.point_code] !== baseline.value[item.point_code]).length,
)

const allOpen = computed(
  () => grouped.value.length > 0 && grouped.value.every((group) => openLevels.value[group.level]),
)

function changedInGroup(level: number): number {
  const group = grouped.value.find((entry) => entry.level === level)
  if (!group) return 0
  return group.items.filter((item) => answers.value[item.point_code] !== baseline.value[item.point_code]).length
}

function toggleLevel(level: number) {
  openLevels.value[level] = !openLevels.value[level]
}

function toggleAll() {
  const next = !allOpen.value
  for (const group of grouped.value) openLevels.value[group.level] = next
}

function setAllUnseen() {
  for (const item of items.value) answers.value[item.point_code] = 0
}

function answer(item: LearningProfileItem, value: number) {
  answers.value[item.point_code] = value
}

function isChanged(item: LearningProfileItem): boolean {
  return answers.value[item.point_code] !== baseline.value[item.point_code]
}

function answerClass(item: LearningProfileItem, value: number): string {
  if (answers.value[item.point_code] !== value) {
    return 'bg-surface text-ink-secondary hover:bg-bg'
  }
  // 未改动：安静的中性选中；已改动：信号朱红（与保存动作同一语义）
  return isChanged(item) ? 'bg-primary font-semibold text-white' : 'bg-ink/10 font-semibold text-ink'
}

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
    baseline.value = { ...initial }

    // 默认只展开第一层，其余层折叠，避免 72 行一次铺满
    openLevels.value = {}
    const firstLevel = grouped.value[0]?.level
    if (firstLevel !== undefined) openLevels.value[firstLevel] = true
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
      <div class="mt-4 flex flex-wrap items-center gap-3">
        <p class="text-xs text-ink-muted">0 = 没接触，3 = 熟练；掌握度 ≥ 80% 的节点会从路径中跳过</p>
        <div class="ml-auto flex flex-wrap items-center gap-2">
          <el-button size="small" @click="setAllUnseen">全部按「没接触过」</el-button>
          <button
            type="button"
            class="text-xs text-ink-secondary underline-offset-4 hover:text-primary hover:underline"
            @click="toggleAll"
          >{{ allOpen ? '全部收起' : '全部展开' }}</button>
        </div>
      </div>

      <section
        v-for="group in grouped"
        :key="group.level"
        class="mt-4 overflow-hidden rounded-[4px] border border-border bg-surface"
      >
        <button
          type="button"
          class="flex w-full flex-wrap items-center gap-3 border-b border-border bg-bg/60 px-5 py-3 text-left transition-colors hover:bg-bg"
          :aria-expanded="!!openLevels[group.level]"
          @click="toggleLevel(group.level)"
        >
          <span class="line-chip" :style="{ '--chip': levelColor(group.level) }">L{{ group.level }}</span>
          <span class="text-sm font-semibold text-ink">第 {{ group.level }} 层</span>
          <span class="data-num text-xs text-ink-muted">{{ group.items.length }} 个知识点</span>
          <span
            v-if="changedInGroup(group.level) > 0"
            class="data-num rounded-[3px] bg-primary/10 px-1.5 py-0.5 text-[11px] font-semibold text-primary-active"
          >已改 {{ changedInGroup(group.level) }}</span>
          <svg
            viewBox="0 0 24 24"
            class="ml-auto h-4 w-4 text-ink-muted transition-transform duration-200"
            :class="openLevels[group.level] ? 'rotate-180' : ''"
            fill="none"
            aria-hidden="true"
          >
            <path d="M6 9l6 6 6-6" stroke="currentColor" stroke-width="2" stroke-linecap="square" />
          </svg>
        </button>

        <div v-show="openLevels[group.level]" class="divide-y divide-border">
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
            <div class="flex overflow-hidden rounded-[3px] border border-control" role="radiogroup" :aria-label="item.name">
              <button
                v-for="option in levelOptions"
                :key="option.value"
                type="button"
                role="radio"
                :aria-checked="answers[item.point_code] === option.value"
                class="px-2.5 py-1 text-xs font-medium transition-colors"
                :class="answerClass(item, option.value)"
                @click="answer(item, option.value)"
              >{{ option.label }}</button>
            </div>
          </div>
        </div>
      </section>

      <div class="sticky bottom-4 mt-6 flex flex-wrap items-center justify-between gap-3 rounded-[4px] border border-border bg-surface px-5 py-3 shadow-[0_10px_30px_-18px_rgba(11,27,43,0.45)]">
        <span class="data-num text-xs text-ink-muted">
          共 {{ items.length }} 个知识点<template v-if="changedCount > 0"> · 已修改 {{ changedCount }} 项</template>
        </span>
        <el-button type="primary" :loading="saving" @click="save">保存并返回知识点</el-button>
      </div>
    </template>
  </div>
</template>
