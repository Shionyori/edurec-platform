<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import ProfileCard from '@/components/profile/ProfileCard.vue'
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
  <div class="mx-auto max-w-5xl px-6 py-8">
    <h1 class="text-2xl font-bold">个人中心</h1>
    <div class="mt-6 space-y-6">
      <ProfileCard v-if="user" :user="user" @updated="handleUpdated" />
      <BehaviorHistory />
    </div>
  </div>
</template>
