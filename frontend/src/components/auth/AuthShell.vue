<template>
  <div class="flex min-h-screen items-center justify-center bg-bg px-4 py-10">
    <div class="w-full max-w-md">
      <div class="mb-7 text-center">
        <RouterLink to="/" class="inline-flex items-center gap-2.5" aria-label="edurec 首页">
          <span class="flex h-9 w-9 items-center justify-center rounded-[3px] bg-rail" aria-hidden="true">
            <svg viewBox="0 0 24 24" class="h-5 w-5" fill="none">
              <path d="M8 5v6.5a3 3 0 0 0 3 3h5" stroke="#FFFFFF" stroke-width="2" stroke-linecap="square" />
              <circle cx="8" cy="4" r="2" fill="#C9451F" />
              <circle cx="17" cy="14.5" r="2" fill="#0E7C7B" />
            </svg>
          </span>
          <span class="text-2xl font-bold tracking-tight text-ink">edurec</span>
        </RouterLink>

        <div class="mt-4 flex items-center justify-center gap-0" aria-hidden="true">
          <span class="h-[2px] w-10 bg-border"></span>
          <span class="route-node mx-1.5" style="--node: var(--color-line-teal)"></span>
          <span class="h-[2px] w-16 bg-border"></span>
          <span class="route-node mx-1.5" style="--node: var(--color-line-blue)"></span>
          <span class="h-[2px] w-10 bg-border"></span>
        </div>

        <p class="mt-3 text-sm text-ink-secondary">发现优质教育资源</p>
      </div>

      <div class="rounded-[4px] border border-border bg-surface p-7 sm:p-8">
        <slot />
      </div>

      <p class="mt-6 text-center text-xs text-ink-muted">© 2026 edurec · 教育资源推荐平台</p>
    </div>
  </div>
</template>
