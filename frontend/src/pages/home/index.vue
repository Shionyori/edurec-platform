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
  <div class="mx-auto max-w-7xl px-6 py-10">
    <div class="flex items-end justify-between">
      <div>
        <h1 class="text-2xl font-bold">发现优质教育资源</h1>
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
  </div>
</template>
