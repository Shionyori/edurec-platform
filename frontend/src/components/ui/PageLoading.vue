<script setup lang="ts">
import { onScopeDispose, watch } from 'vue'
import { pageLoading } from '@/composables/pageLoading'
import AppIcon from './AppIcon.vue'

const { visible } = pageLoading

// The overlay covers the viewport; lock the document so a stray wheel/touch
// cannot move the page that is about to be replaced.
watch(
  visible,
  (active) => {
    document.documentElement.style.overflow = active ? 'hidden' : ''
    document.body.style.overflow = active ? 'hidden' : ''
  },
  { immediate: true },
)
onScopeDispose(() => {
  document.documentElement.style.overflow = ''
  document.body.style.overflow = ''
})
</script>

<template>
  <Transition name="page-loader">
    <div
      v-if="visible"
      class="page-loader"
      role="status"
      aria-live="polite"
      aria-label="页面加载中"
    >
      <div class="page-loader__inner">
        <span class="page-loader__mark" aria-hidden="true">
          <AppIcon name="spark-filled" :size="21" />
        </span>
        <p class="page-loader__label">正在打开…</p>
        <span class="page-loader__bar" aria-hidden="true"><span></span></span>
      </div>
    </div>
  </Transition>
</template>
