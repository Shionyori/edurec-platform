<script setup lang="ts">
withDefaults(defineProps<{ size?: number | string }>(), { size: 36 })
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 32 32"
    fill="none"
    aria-hidden="true"
    class="shrink-0 text-primary"
  >
    <rect width="32" height="32" rx="9" fill="currentColor" />
    <path
      d="M16 5.2l2.1 6.7 6.7 2.1-6.7 2.1L16 22.8l-2.1-6.7-6.7-2.1 6.7-2.1L16 5.2Z"
      fill="#FFFFFF"
    />
    <circle cx="24.2" cy="7.8" r="2.1" fill="#B7E4D8" />
  </svg>
</template>
