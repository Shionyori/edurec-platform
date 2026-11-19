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
      d="M16 4l2.8 9.2L28 16l-9.2 2.8L16 28l-2.8-9.2L4 16l9.2-2.8L16 4Z"
      fill="#FFFFFF"
    />
    <circle cx="24.6" cy="7.4" r="1.7" fill="#B7E4D8" />
  </svg>
</template>
