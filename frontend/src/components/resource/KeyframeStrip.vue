<script setup lang="ts">
import { computed } from 'vue'
import AppIcon from '@/components/ui/AppIcon.vue'

export interface VideoFrames {
  bvid?: string
  sprites: string[]
  x_len: number
  y_len: number
  x_size?: number
  y_size?: number
  count?: number
  source?: string
}

const props = withDefaults(
  defineProps<{
    frames: VideoFrames
    max?: number
  }>(),
  { max: 8 },
)

const cols = computed(() => Math.max(1, props.frames.x_len || 10))
const rows = computed(() => Math.max(1, props.frames.y_len || 10))
const cellWidth = 160
const cellHeight = computed(() => {
  const width = props.frames.x_size || 480
  const height = props.frames.y_size || 270
  return Math.round((cellWidth * height) / width)
})

// B 站把关键帧拼成雪碧图：每张图按 cols × rows 网格排布。
// 这里只记录每格是哪张图的第几列第几行，用 CSS 的 background-size/position 裁切，
// 不下载、不转存图片；一张雪碧图只加载一次。
const cells = computed(() => {
  const perSprite = cols.value * rows.value
  const total = props.frames.sprites.length * perSprite
  const count = Math.min(props.max, total)
  const out: Array<{ key: string; sprite: string; style: Record<string, string> }> = []
  for (let i = 0; i < count; i++) {
    const spriteIndex = Math.floor(i / perSprite)
    const offset = i % perSprite
    const col = offset % cols.value
    const row = Math.floor(offset / cols.value)
    out.push({
      key: `${spriteIndex}-${col}-${row}`,
      sprite: props.frames.sprites[spriteIndex],
      style: {
        backgroundImage: `url("${props.frames.sprites[spriteIndex]}")`,
        backgroundSize: `${cols.value * 100}% ${rows.value * 100}%`,
        backgroundPosition: `${position(col, cols.value)}% ${position(row, rows.value)}%`,
        width: `${cellWidth}px`,
        height: `${cellHeight.value}px`,
      },
    })
  }
  return out
})

function position(index: number, length: number): number {
  return length > 1 ? (index / (length - 1)) * 100 : 0
}
</script>

<template>
  <div v-if="cells.length > 0" class="card mt-4 p-5 sm:p-6">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2.5">
        <span class="flex h-8 w-8 items-center justify-center rounded-[9px] bg-sunken text-ink-secondary">
          <AppIcon name="play" :size="15" />
        </span>
        <h2 class="text-[16px] font-bold text-ink">关键帧</h2>
      </div>
      <span class="tnum text-[11.5px] text-ink-muted">
        共 {{ frames.count ?? cells.length }} 帧 · 来自 B 站视频预览
      </span>
    </div>
    <div class="mt-4 flex gap-2 overflow-x-auto pb-1">
      <div
        v-for="cell in cells"
        :key="cell.key"
        class="shrink-0 rounded-[9px] border border-border bg-bg bg-no-repeat"
        :style="cell.style"
        :title="`关键帧：${cell.sprite}`"
      />
    </div>
    <p class="mt-3 text-[11.5px] leading-relaxed text-ink-muted">
      关键帧用于快速预览课程节奏，不额外存储图片；点击封面或前往原站可观看完整视频。
    </p>
  </div>
</template>
