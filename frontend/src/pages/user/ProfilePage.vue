<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import ProfileCard from '@/components/profile/ProfileCard.vue'
import FavoriteList from '@/components/profile/FavoriteList.vue'
import BehaviorHistory from '@/components/profile/BehaviorHistory.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
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
  <div class="page shell">
    <div class="mx-auto max-w-[920px]">
      <header class="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 class="headline">个人中心</h1>
          <p class="lede mt-2.5">管理资料、收藏与行为历史；这些数据也会影响推荐。</p>
        </div>
        <RouterLink :to="{ name: 'questionnaire' }" class="btn btn-secondary">
          <AppIcon name="target" :size="16" />
          学习问卷
        </RouterLink>
      </header>

      <div class="mt-7 space-y-5">
        <ProfileCard v-if="user" :user="user" @updated="handleUpdated" />
        <FavoriteList />
        <BehaviorHistory />
      </div>
    </div>
  </div>
</template>
