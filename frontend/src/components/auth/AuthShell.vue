<script setup lang="ts">
import AppIcon from '@/components/ui/AppIcon.vue'
import BrandMark from '@/components/ui/BrandMark.vue'

const features = [
  { icon: 'route', title: '先修关系排序', copy: '每一步都建立在已掌握的知识之上。' },
  { icon: 'target', title: '跳过已掌握', copy: '按你的真实水平推荐，不重复学。' },
  { icon: 'spark', title: '推荐理由可读', copy: '知道为什么是这一节，而不是一堆资源。' },
]
</script>

<template>
  <div class="min-h-screen bg-bg lg:grid lg:grid-cols-[0.88fr_1.12fr]">
    <aside class="relative hidden overflow-hidden border-r border-border bg-primary-softer px-12 py-12 lg:flex lg:flex-col xl:px-16">
      <svg
        class="pointer-events-none absolute -bottom-44 -left-32 h-[560px] w-[560px] text-primary/10"
        viewBox="0 0 560 560"
        fill="none"
        aria-hidden="true"
      >
        <circle cx="280" cy="280" r="180" stroke="currentColor" stroke-width="1.2" />
        <circle cx="280" cy="280" r="250" stroke="currentColor" stroke-width="1.2" stroke-dasharray="2 8" />
        <circle cx="280" cy="280" r="90" stroke="currentColor" stroke-width="1.2" />
        <path d="M280 100v360M100 280h360" stroke="currentColor" stroke-width="1.2" stroke-dasharray="2 10" />
        <circle cx="460" cy="280" r="7" fill="var(--color-primary)" />
        <circle cx="280" cy="100" r="4" fill="currentColor" />
      </svg>

      <div class="relative z-10 flex flex-1 flex-col">
        <RouterLink to="/" class="flex items-center gap-2.5" aria-label="edurec 首页">
          <BrandMark :size="36" />
          <span class="text-[19px] font-extrabold tracking-[-0.045em] text-ink">edurec</span>
        </RouterLink>

        <div class="mt-auto max-w-sm">
          <p class="text-[34px] font-bold leading-[1.18] tracking-[-0.03em] text-ink xl:text-[38px]">
            把下一步，<br />学明白。
          </p>
          <p class="mt-4 max-w-xs text-[14px] leading-relaxed text-ink-secondary">
            知识点、先修关系与学习路径放在一起，告诉你现在该学哪一节，以及为什么。
          </p>

          <ul class="mt-10 space-y-5">
            <li v-for="feature in features" :key="feature.title" class="flex items-start gap-3.5">
              <span class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-surface text-primary">
                <AppIcon :name="feature.icon" :size="17" />
              </span>
              <div>
                <p class="text-[13.5px] font-semibold text-ink">{{ feature.title }}</p>
                <p class="mt-0.5 text-[12.5px] leading-relaxed text-ink-muted">{{ feature.copy }}</p>
              </div>
            </li>
          </ul>
        </div>

        <p class="mt-auto pt-12 text-[11.5px] text-ink-muted">
          用知识点与先修关系组织学习 · edurec
        </p>
      </div>
    </aside>

    <main class="flex min-h-screen flex-col items-center justify-center px-5 py-10 sm:px-8">
      <div class="w-full max-w-[420px]">
        <RouterLink to="/" class="mb-8 inline-flex items-center gap-2.5 lg:hidden" aria-label="edurec 首页">
          <BrandMark :size="36" />
          <span class="text-[19px] font-extrabold tracking-[-0.045em] text-ink">edurec</span>
        </RouterLink>

        <div class="card p-6 sm:p-8">
          <slot />
        </div>

        <p class="mt-6 text-center text-[11.5px] text-ink-muted">© 2026 edurec · 教育资源推荐平台</p>
      </div>
    </main>
  </div>
</template>
