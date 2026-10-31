<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import ProfileCard from '@/components/profile/ProfileCard.vue'
import FavoriteList from '@/components/profile/FavoriteList.vue'
import BehaviorHistory from '@/components/profile/BehaviorHistory.vue'
import type { User } from '@/types'

const auth = useAuthStore()
const user = ref<User | null>(auth.user)

onMounted(async () => {
  try {
    user.value = await auth.fetchMe()
  } catch {
    // 拉取失败时保留当前缓存用户
  }
})

async function handleUpdated() {
  try {
    user.value = await auth.fetchMe()
  } catch {
    // 忽略刷新失败
  }
}
</script>

<template>
  <div class="mx-auto max-w-5xl px-4 py-8 sm:px-6">
    <div class="border-b border-border pb-4">
      <h1 class="text-2xl font-bold tracking-tight text-ink sm:text-3xl">个人中心</h1>
      <p class="mt-2 text-sm text-ink-secondary">管理资料、收藏与行为历史；这些数据也会影响推荐。</p>
    </div>
    <div class="mt-6 space-y-5">
      <ProfileCard v-if="user" :user="user" @updated="handleUpdated" />
      <FavoriteList />
      <BehaviorHistory />
    </div>
  </div>
</template>
