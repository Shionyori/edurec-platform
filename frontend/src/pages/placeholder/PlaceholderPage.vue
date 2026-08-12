<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<template>
  <div class="flex items-center justify-center py-24">
    <div class="text-center">
      <h2 class="text-xl font-semibold">功能开发中</h2>
      <p class="mt-2 text-sm text-ink-muted">路由：{{ route.name }}（后续子项目实现）</p>
    </div>
  </div>
</template>
