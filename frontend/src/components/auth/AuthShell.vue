<script setup lang="ts">
import { themeTokens } from '@/styles/theme'
</script>

<template>
  <div
    class="flex min-h-screen items-center justify-center bg-bg px-4"
    :style="{
      backgroundImage: `radial-gradient(ellipse at top left, ${themeTokens.color.primary}14, transparent 50%), radial-gradient(ellipse at bottom right, ${themeTokens.color.primary}0f, transparent 50%)`,
    }"
  >
    <div class="w-full max-w-md">
      <div class="mb-8 text-center">
        <RouterLink to="/" class="inline-flex items-center gap-2 text-2xl font-bold text-primary">
          <span class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary text-white">E</span>
          edurec
        </RouterLink>
        <p class="mt-2 text-sm text-ink-secondary">发现优质教育资源</p>
      </div>
      <div class="rounded-xl border border-border bg-surface p-8 shadow-sm">
        <slot />
      </div>
      <p class="mt-6 text-center text-xs text-ink-muted">© 2026 edurec · 教育资源推荐平台</p>
    </div>
  </div>
</template>
