<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getRecommendations } from '@/api/recommendation'
import { recordImpressions } from '@/api/impression'
import { useAuthStore } from '@/stores/auth'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import ResourceCardSkeleton from '@/components/resource/ResourceCardSkeleton.vue'
import InterestPicker from '@/components/profile/InterestPicker.vue'
import GoalInput from './GoalInput.vue'
import LearningPlanView from './LearningPlanView.vue'
import type { Resource } from '@/types'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const resources = ref<Resource[]>([])
const loading = ref(false)
const error = ref('')

// 目标存在即进入「学习计划」态，否则保持原有推荐流。
// 目标是唯一的状态来源，刷新与前进/后退都能恢复。
const goal = computed(() => (typeof route.query.goal === 'string' ? route.query.goal : ''))

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

function submitGoal(next: string) {
  router.push({ name: 'home', query: { goal: next } })
}

function resetGoal() {
  router.push({ name: 'home' })
}

// 未提交目标时才有推荐流——提交后整页切换为知识点面板（决策 #7 / #8）。
//
// 用 watch + immediate 而不是 onMounted：只在挂载时拉一次的话，带 ?goal= 进入
// （或刷新时正处于计划态）再点「换一个目标」，goal 变空但 resources 仍是 []，
// 页面会渲染成「暂无推荐内容」并且不再发请求——对已有兴趣的用户没有恢复入口。
watch(
  () => goal.value,
  (current) => {
    if (!current && resources.value.length === 0 && !loading.value) load()
  },
  { immediate: true },
)
</script>

<template>
  <div class="mx-auto max-w-7xl px-6 py-10">
    <div class="mx-auto max-w-3xl text-center">
      <h1 class="text-2xl font-bold">想学点什么？</h1>
      <p class="mt-2 text-sm text-ink-secondary">
        说出你的学习目标，我们把它拆成知识点，每个知识点配一组资源
      </p>
      <div class="mt-6">
        <GoalInput :initial-goal="goal" @submit="submitGoal" />
      </div>
    </div>

    <div v-if="goal" class="mt-12">
      <h2 class="text-xl font-bold">为你规划的学习路径</h2>
      <LearningPlanView :goal="goal" @reset="resetGoal" />
    </div>

    <template v-else>
      <div class="mt-12 flex items-end justify-between">
        <div>
          <h2 class="text-xl font-bold">发现优质教育资源</h2>
          <p class="mt-2 text-sm text-ink-secondary">为你推荐的精选课程、文章与视频</p>
        </div>
        <RouterLink :to="{ name: 'search' }">
          <el-button round>查看全部</el-button>
        </RouterLink>
      </div>

      <InterestPicker v-if="showInterestPicker" @saved="load" />

      <div v-if="loading" class="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <ResourceCardSkeleton v-for="i in 6" :key="i" />
      </div>
      <div v-else-if="error" class="py-24 text-center text-sm text-red-500">{{ error }}</div>
      <el-empty v-else-if="resources.length === 0" description="暂无推荐内容" class="mt-8" />
      <div v-else class="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <ResourceCard
          v-for="(r, i) in resources"
          :key="r.id"
          :resource="r"
          :style="{ animationDelay: `${Math.min(i, 8) * 30}ms` }"
        />
      </div>
    </template>
  </div>
</template>
