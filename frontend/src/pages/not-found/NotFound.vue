<template>
  <div class="flex items-center justify-center py-24">
    <div class="text-center">
      <h1 class="text-4xl font-bold text-primary">404</h1>
      <p class="mt-2 text-sm text-ink-secondary">页面不存在</p>
      <RouterLink :to="{ name: 'home' }">
        <el-button type="primary" round class="mt-6">返回首页</el-button>
      </RouterLink>
    </div>
  </div>
</template>
