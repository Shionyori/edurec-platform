<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    name: string
    size?: number | string
    strokeWidth?: number
  }>(),
  { size: 20, strokeWidth: 1.7 },
)

// One authored icon set: 24px grid, round caps, consistent optical weight.
// Icons are stroke-first; the few filled marks are listed explicitly.
const FILLED = new Set(['star-filled', 'heart-filled', 'spark-filled'])

const ICONS: Record<string, string> = {
  home: 'M3.5 10.8 12 3.5l8.5 7.3|M5.5 9.5V20a.8.8 0 0 0 .8.8h11.4a.8.8 0 0 0 .8-.8V9.5',
  compass:
    'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18Z|M15.6 8.4l-2.1 5.1-5.1 2.1 2.1-5.1 5.1-2.1Z',
  search: 'M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14Z|m16.2 16.2 4.3 4.3',
  user: 'M20 21a8 8 0 0 0-16 0|M12 3.5a4.5 4.5 0 1 0 0 9 4.5 4.5 0 0 0 0-9Z',
  spark:
    'M12 3l2.1 6.9L21 12l-6.9 2.1L12 21l-2.1-6.9L3 12l6.9-2.1L12 3Z',
  'spark-filled':
    'M12 3l2.1 6.9L21 12l-6.9 2.1L12 21l-2.1-6.9L3 12l6.9-2.1L12 3Z',
  'arrow-right': 'M4 12h15|m13 6 6 6-6 6',
  'arrow-left': 'M20 12H5|m11 6-6 6 6 6',
  'arrow-up-right': 'M7 17 17 7|M9 7h8v8',
  'chevron-right': 'm9 6 6 6-6 6',
  'chevron-down': 'm6 9 6 6 6-6',
  'chevron-up': 'm6 15 6-6 6 6',
  book: 'M4 5.5A2.5 2.5 0 0 1 6.5 3H20v15H6.5A2.5 2.5 0 0 0 4 20.5V5.5Z|M4 20.5A2.5 2.5 0 0 1 6.5 18H20v3H6.5A2.5 2.5 0 0 1 4 20.5Z',
  'book-open':
    'M12 6.5C10.5 4.8 8.2 4 5 4H3v15h3c2.6 0 4.7.7 6 2 1.3-1.3 3.4-2 6-2h3V4h-2c-3.2 0-5.5.8-7 2.5Z|M12 6.5V21',
  play: 'M8.5 5.2v13.6L19 12 8.5 5.2Z',
  'play-circle': 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18Z|m10 8.5 5.5 3.5-5.5 3.5v-7Z',
  star: 'M12 3.5l2.6 5.3 5.9.9-4.3 4.1 1 5.8-5.2-2.7-5.2 2.7 1-5.8-4.3-4.1 5.9-.9L12 3.5Z',
  'star-filled':
    'M12 3.4l2.6 5.4 5.9.9-4.3 4.2 1 5.8-5.2-2.8-5.2 2.8 1-5.8-4.3-4.2 5.9-.9L12 3.4Z',
  eye: 'M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12Z|M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6Z',
  heart:
    'M12 20.5S3.5 15.5 3.5 9.5A4.5 4.5 0 0 1 12 6.8a4.5 4.5 0 0 1 8.5 2.7c0 6-8.5 11-8.5 11Z',
  'heart-filled':
    'M12 20.5S3.5 15.5 3.5 9.5A4.5 4.5 0 0 1 12 6.8a4.5 4.5 0 0 1 8.5 2.7c0 6-8.5 11-8.5 11Z',
  clock: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18Z|M12 7.2V12l3.3 2',
  layers: 'm12 3 9 5-9 5-9-5 9-5Z|m3 12.5 9 5 9-5|m3 16.5 9 5 9-5',
  route: 'M6.5 19.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z|M17.5 9.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z|M9 17h3.5a3 3 0 0 0 3-3V9',
  check: 'm5 12.5 4.5 4.5L19 7.5',
  'check-circle': 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18Z|m8 12 2.7 2.7L16.5 9',
  lock: 'M6.5 10V7.2a5.5 5.5 0 0 1 11 0V10|M5 10h14v10.2H5V10Z|M12 14v2.5',
  plus: 'M12 5v14|M5 12h14',
  minus: 'M5 12h14',
  dashboard: 'M4 4h7v7H4V4Z|M13 4h7v4h-7V4Z|M13 10h7v10h-7V10Z|M4 13h7v7H4v-7Z',
  users:
    'M15.5 20a6.5 6.5 0 0 0-13 0|M9 3.5a3.6 3.6 0 1 0 0 7.2 3.6 3.6 0 0 0 0-7.2Z|M17 4.5a3.2 3.2 0 0 1 0 6.2|M18 13.4a6.5 6.5 0 0 1 3.5 5.8',
  folder: 'M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z',
  chart: 'M4 20V4|M4 20h16|M8 16v-4|M12.5 16V8|M17 16v-7',
  logout: 'M9 4H5a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h4|M15 8l4 4-4 4|M19 12H9',
  external: 'M14 4h6v6|M20 4l-9 9|M19 13v5a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h5',
  trash: 'M4 7h16|M9 7V4h6v3|M6.2 7l1 13h9.6l1-13|M10 11v6|M14 11v6',
  edit: 'M4 20h4l11-11a2.1 2.1 0 0 0-3-3L5 17v3Z|M13.5 6.5l3 3',
  mail: 'M3 6h18v12H3V6Z|m3 7 9 6 9-6',
  key: 'M15 8.5a4 4 0 1 0-3.7 4L9 14.8H7v2H5v2H2.8v-3.2l7.5-7.5A4 4 0 0 1 15 8.5Z',
  filter: 'M4 5h16l-6 7v7l-4 2v-9L4 5Z',
  refresh: 'M20 11a8 8 0 1 0-2.3 6.4|M20 4v7h-7',
  menu: 'M4 7h16|M4 12h16|M4 17h16',
  close: 'm6 6 12 12|m18 6-12 12',
  target: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18Z|M12 7a5 5 0 1 0 0 10 5 5 0 0 0 0-10Z|M12 10.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3Z',
  cap: 'm12 4 10 5-10 5L2 9l10-5Z|M6 11.5V16c0 1.6 2.7 3 6 3s6-1.4 6-3v-4.5',
  info: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18Z|M12 11v5|M12 7.6v.1',
  alert: 'M12 3.2 21 19H3L12 3.2Z|M12 9v4.5|M12 16.5v.1',
  calendar: 'M4 6h16v14H4V6Z|M8 3v4|M16 3v4|M4 10h16',
  tag: 'M3.5 4.5h7l10 10-6 6-10-10v-6Z|M8 8h.1',
  link: 'M10 13a5 5 0 0 0 7 0l2-2a5 5 0 0 0-7-7l-1 1|M14 11a5 5 0 0 0-7 0l-2 2a5 5 0 0 0 7 7l1-1',
  'more-horizontal': 'M5 12h.1|M12 12h.1|M19 12h.1',
  sort: 'M8 5v14|m4 15 4 4 4-4|M16 19V5|m12 9 4-4 4 4',
  grid: 'M4 4h7v7H4V4Z|M13 4h7v7h-7V4Z|M4 13h7v7H4v-7Z|M13 13h7v7h-7v-7Z',
  list: 'M9 6h11|M9 12h11|M9 18h11|M4.5 6h.1|M4.5 12h.1|M4.5 18h.1',
  bolt: 'M13 2 4.5 13.5H11L10 22l8.5-11.5H12L13 2Z',
  'arrow-down': 'M12 4v15|m6 13 6 6 6-6',
}

const paths = computed(() => (ICONS[props.name] ?? ICONS.spark).split('|'))
const filled = computed(() => FILLED.has(props.name))
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    :fill="filled ? 'currentColor' : 'none'"
    :stroke="filled ? 'none' : 'currentColor'"
    :stroke-width="strokeWidth"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    class="shrink-0"
  >
    <path v-for="(d, index) in paths" :key="index" :d="d" />
  </svg>
</template>
