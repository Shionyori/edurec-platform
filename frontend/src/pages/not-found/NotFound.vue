<script setup lang="ts">
import AppIcon from '@/components/ui/AppIcon.vue'
import BrandMark from '@/components/ui/BrandMark.vue'
</script>

<template>
  <main class="page shell">
    <div class="mx-auto flex max-w-xl flex-col items-center py-16 text-center sm:py-24">
      <RouterLink to="/" class="mb-12 inline-flex items-center gap-2.5" aria-label="edurec 首页">
        <BrandMark :size="36" />
        <span class="text-[19px] font-extrabold tracking-[-0.045em] text-ink">edurec</span>
      </RouterLink>
      <div class="relative">
        <svg class="pointer-events-none absolute -inset-12 text-ink/[0.07]" viewBox="0 0 300 300" fill="none" aria-hidden="true">
          <circle cx="150" cy="150" r="118" stroke="currentColor" stroke-width="1.2" />
          <circle cx="150" cy="150" r="146" stroke="currentColor" stroke-width="1.2" stroke-dasharray="2 8" />
          <circle cx="268" cy="150" r="5" fill="var(--color-primary)" />
        </svg>
        <p class="relative text-[86px] font-extrabold leading-none tracking-[-0.07em] text-ink sm:text-[112px]">404</p>
      </div>
      <h1 class="mt-4 text-[22px] font-bold tracking-[-0.03em] text-ink">页面不存在</h1>
      <p class="mt-2 max-w-sm text-[13.5px] leading-relaxed text-ink-muted">
        这个地址可能已经失效，或者你点到了一个不存在的链接。
      </p>
      <div class="mt-7 flex flex-wrap justify-center gap-3">
        <RouterLink :to="{ name: 'home' }" class="btn btn-primary">
          <AppIcon name="home" :size="16" />
          返回首页
        </RouterLink>
        <RouterLink :to="{ name: 'knowledge' }" class="btn btn-secondary">
          浏览知识点
        </RouterLink>
      </div>
    </div>
  </main>
</template>
