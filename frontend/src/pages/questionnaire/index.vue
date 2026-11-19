<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getLearningProfile, submitQuestionnaire } from '@/api/learning'
import type { LearningProfileItem } from '@/types'
import AppIcon from '@/components/ui/AppIcon.vue'

const router = useRouter()

const items = ref<LearningProfileItem[]>([])
const answers = ref<Record<string, number>>({})
// 原始值用于区分「未改动」与「已改动」：朱红只留给真正的改动与保存动作
const baseline = ref<Record<string, number>>({})
const openLevels = ref<Record<number, boolean>>({})
// 批量“全部按没接触过”的撤销快照：避免一键覆盖已填/行为估计的答案
const bulkUndo = ref<Record<string, number> | null>(null)
const loading = ref(true)
const saving = ref(false)
const error = ref('')

const levelLabels = ['没接触过', '了解概念', '学过并练习', '熟练']
const levelOptions = levelLabels.map((label, value) => ({ label, value }))

const LINE_COLORS = ['#0F7A66', '#2F62C4', '#A26800', '#B84A62']
function levelColor(level: number): string {
  return LINE_COLORS[(Math.max(1, level) - 1) % LINE_COLORS.length]
}
const LEVEL_INKS = ['#0B6354', '#2757B5', '#7A4D00', '#913B4D']
function levelInk(level: number): string {
  return LEVEL_INKS[(Math.max(1, level) - 1) % LEVEL_INKS.length]
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
  if (!bulkUndo.value) bulkUndo.value = { ...answers.value }
  for (const item of items.value) answers.value[item.point_code] = 0
}

function undoBulk() {
  if (!bulkUndo.value) return
  answers.value = { ...bulkUndo.value }
  bulkUndo.value = null
}

function answer(item: LearningProfileItem, value: number) {
  answers.value[item.point_code] = value
  bulkUndo.value = null
}

function isChanged(item: LearningProfileItem): boolean {
  return answers.value[item.point_code] !== baseline.value[item.point_code]
}

function answerClass(item: LearningProfileItem, value: number): string {
  if (answers.value[item.point_code] !== value) {
    return 'text-ink-secondary hover:bg-sunken'
  }
  // 未改动：安静的中性选中；已改动：信号朱红（与保存动作同一语义）
  return isChanged(item) ? 'bg-primary text-white shadow-sm' : 'bg-ink text-white'
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
  <div class="page shell">
    <div class="mx-auto max-w-[920px]">
      <header class="flex flex-wrap items-start justify-between gap-4">
        <div class="max-w-2xl">
          <h1 class="headline">初始学习问卷</h1>
          <p class="lede mt-2.5">
            按你的真实水平自评每个知识点，系统会据此跳过已掌握内容，只推荐路径上还没掌握的部分。以后可以随时回来重做。
          </p>
        </div>
        <RouterLink :to="{ name: 'knowledge' }" class="btn btn-secondary">
          <AppIcon name="compass" :size="16" />
          先去看知识点
        </RouterLink>
      </header>

      <div v-if="loading" class="mt-6 space-y-4">
        <div v-for="i in 4" :key="i" class="skeleton h-16 !rounded-[14px]"></div>
      </div>

      <div v-else-if="error" class="state-box mt-6">
        <span class="state-icon !bg-danger-soft !text-danger">
          <AppIcon name="alert" :size="22" />
        </span>
        <p class="state-title">问卷加载失败</p>
        <p class="state-copy">{{ error }}</p>
        <button type="button" class="btn btn-secondary btn-sm mt-2" @click="load">重新加载</button>
      </div>

      <template v-else>
        <div class="mt-6 flex flex-wrap items-center gap-3 rounded-[12px] border border-border bg-surface px-4 py-3">
          <p class="text-[12px] text-ink-muted">0 = 没接触，3 = 熟练；掌握度 ≥ 80% 的节点会从路径中跳过</p>
          <div class="ml-auto flex flex-wrap items-center gap-2">
            <button type="button" class="btn btn-secondary btn-sm" @click="setAllUnseen">全部按「没接触过」</button>
            <button
              v-if="bulkUndo"
              type="button"
              class="text-[12px] font-semibold text-primary underline-offset-4 hover:underline"
              @click="undoBulk"
            >撤销全设</button>
            <button
              type="button"
              class="text-[12px] font-semibold text-ink-secondary underline-offset-4 hover:text-primary hover:underline"
              @click="toggleAll"
            >{{ allOpen ? '全部收起' : '全部展开' }}</button>
          </div>
        </div>

        <section
          v-for="group in grouped"
          :key="group.level"
          class="card mt-4 overflow-hidden"
        >
          <button
            type="button"
            class="flex w-full flex-wrap items-center gap-3 px-5 py-3.5 text-left transition-colors hover:bg-surface-2"
            :aria-expanded="!!openLevels[group.level]"
            @click="toggleLevel(group.level)"
          >
            <span
              class="flex h-6 min-w-6 items-center justify-center rounded-full px-2 text-[11px] font-bold"
              :style="{ backgroundColor: `color-mix(in srgb, ${levelColor(group.level)} 12%, white)`, color: levelInk(group.level) }"
            >L{{ group.level }}</span>
            <span class="text-[14px] font-bold text-ink">第 {{ group.level }} 层</span>
            <span class="tnum text-[12px] text-ink-muted">{{ group.items.length }} 个知识点</span>
            <span
              v-if="changedInGroup(group.level) > 0"
              class="tnum chip chip-primary !py-0 !text-[10.5px]"
            >已改 {{ changedInGroup(group.level) }}</span>
            <span class="ml-auto flex h-7 w-7 items-center justify-center rounded-full text-ink-muted transition-transform duration-200" :class="openLevels[group.level] ? 'rotate-180' : ''">
              <AppIcon name="chevron-down" :size="16" />
            </span>
          </button>

          <div v-show="openLevels[group.level]" class="divide-y divide-border border-t border-border">
            <div
              v-for="item in group.items"
              :key="item.point_code"
              class="flex flex-wrap items-center justify-between gap-3 px-5 py-3 transition-colors hover:bg-surface-2/70"
            >
              <div class="min-w-40">
                <span class="text-[13.5px] font-semibold text-ink">{{ item.name }}</span>
                <span v-if="item.source" class="ml-2 text-[11.5px] text-ink-muted">
                  （当前：{{ item.source === 'behavior' ? '行为估计' : '已填写' }}）
                </span>
              </div>
              <div
                class="flex flex-wrap overflow-hidden rounded-[9px] border border-border-strong bg-surface p-0.5"
                role="radiogroup"
                :aria-label="item.name"
              >
                <button
                  v-for="option in levelOptions"
                  :key="option.value"
                  type="button"
                  role="radio"
                  :aria-checked="answers[item.point_code] === option.value"
                  class="rounded-[7px] px-2.5 py-1.5 text-[11.5px] font-semibold transition-colors duration-150"
                  :class="answerClass(item, option.value)"
                  @click="answer(item, option.value)"
                >{{ option.label }}</button>
              </div>
            </div>
          </div>
        </section>

        <div class="sticky bottom-[calc(80px+env(safe-area-inset-bottom))] mt-6 flex flex-wrap items-center justify-between gap-3 rounded-[14px] border border-border bg-surface px-5 py-3.5 shadow-[var(--shadow-lift)] md:bottom-6">
          <span class="tnum text-[12px] text-ink-muted">
            共 {{ items.length }} 个知识点<template v-if="changedCount > 0"> · 已修改 {{ changedCount }} 项</template>
          </span>
          <el-button type="primary" :loading="saving" @click="save">保存并返回知识点</el-button>
        </div>
      </template>
    </div>
  </div>
</template>
