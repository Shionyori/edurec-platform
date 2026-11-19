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
      d="M16 6l2.33 7.67L26 16l-7.67 2.33L16 26l-2.33-7.67L6 16l7.67-2.33L16 6Z"
      fill="#FFFFFF"
    />
    <circle cx="24.6" cy="7.4" r="1.7" fill="#B7E4D8" />
  </svg>
</template>
