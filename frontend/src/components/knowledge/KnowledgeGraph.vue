<script setup lang="ts">
import { computed } from 'vue'
import type { KnowledgeEdge, KnowledgeNode } from '@/types'
import { layoutKnowledgeGraph } from '@/utils/knowledgeGraph'

const props = withDefaults(
  defineProps<{
    nodes: KnowledgeNode[]
    edges: KnowledgeEdge[]
    currentId?: number
    compact?: boolean
  }>(),
  { currentId: 0, compact: false },
)

const emit = defineEmits<{ (e: 'select', node: KnowledgeNode): void }>()

const layout = computed(() =>
  layoutKnowledgeGraph(props.nodes, props.edges, {
    dx: props.compact ? 220 : 260,
    dy: props.compact ? 78 : 100,
    nodeWidth: props.compact ? 150 : 180,
    nodeHeight: props.compact ? 48 : 56,
  }),
)

const nodeWidth = computed(() => (props.compact ? 150 : 180))
const nodeHeight = computed(() => (props.compact ? 48 : 56))
const arrowId = `kg-arrow-${Math.random().toString(36).slice(2, 8)}`

// 线路色按层级循环：层级是真实语义，不是装饰
const LINE_COLORS = ['#0E7C7B', '#1D4ED8', '#B45309', '#BE123C']
function levelColor(level: number): string {
  return LINE_COLORS[(Math.max(1, level) - 1) % LINE_COLORS.length]
}

const levelTicks = computed(() => {
  if (layout.value.nodes.length === 0) return []
  return Array.from({ length: layout.value.levels }, (_, index) => ({
    level: index + 1,
    x: 24 + index * (props.compact ? 220 : 260),
  }))
})

function edgePath(fromX: number, fromY: number, toX: number, toY: number): string {
  const startX = fromX + nodeWidth.value
  const midX = (startX + toX) / 2
  return `M ${startX} ${fromY} C ${midX} ${fromY}, ${midX} ${toY}, ${toX} ${toY}`
}

function labelOf(node: KnowledgeNode): string {
  return node.name.length > 11 ? `${node.name.slice(0, 11)}…` : node.name
}

function onSelect(node: KnowledgeNode) {
  emit('select', node)
}
</script>

<template>
  <div class="overflow-x-auto">
    <svg
      v-if="layout.nodes.length > 0"
      :viewBox="`0 0 ${layout.width} ${layout.height + 14}`"
      :width="layout.width"
      :height="layout.height + 14"
      role="img"
      aria-label="知识点先修关系图"
      class="min-w-full"
    >
      <defs>
        <marker
          :id="arrowId"
          markerWidth="8"
          markerHeight="8"
          refX="7"
          refY="4"
          orient="auto"
        >
          <path d="M 0 0 L 8 4 L 0 8 z" fill="#B9B5A8" />
        </marker>
        <marker
          :id="`${arrowId}-active`"
          markerWidth="8"
          markerHeight="8"
          refX="7"
          refY="4"
          orient="auto"
        >
          <path d="M 0 0 L 8 4 L 0 8 z" fill="#E4572E" />
        </marker>
      </defs>

      <!-- 层级刻度：线路的“站序” -->
      <g aria-hidden="true">
        <text
          v-for="tick in levelTicks"
          :key="tick.level"
          :x="tick.x"
          y="10"
          font-size="10"
          font-weight="700"
          letter-spacing="1.5"
          fill="#77828C"
        >L{{ tick.level }}</text>
      </g>

      <path
        v-for="(edge, index) in layout.edges"
        :key="index"
        :d="edgePath(edge.from.x, edge.from.y + nodeHeight / 2, edge.to.x, edge.to.y + nodeHeight / 2)"
        fill="none"
        :stroke="edge.to.id === currentId ? '#E4572E' : '#C9C5B8'"
        :stroke-width="edge.to.id === currentId ? 2 : 2"
        :opacity="edge.to.id === currentId ? 1 : 0.75"
        :marker-end="`url(#${edge.to.id === currentId ? `${arrowId}-active` : arrowId})`"
      />

      <g
        v-for="node in layout.nodes"
        :key="node.id"
        :transform="`translate(${node.x}, ${node.y + 14})`"
        role="button"
        tabindex="0"
        :aria-label="`${node.name}，第 ${node.level} 层，资源 ${node.resource_count}，概念 ${node.concept_count}`"
        class="kg-node cursor-pointer"
        @click="onSelect(node)"
        @keydown.enter="onSelect(node)"
        @keydown.space.prevent="onSelect(node)"
      >
        <rect
          :width="nodeWidth"
          :height="nodeHeight"
          rx="3"
          :fill="node.id === currentId ? '#0B1B2B' : '#FFFFFF'"
          :stroke="node.id === currentId ? '#E4572E' : '#DCD9CF'"
          :stroke-width="node.id === currentId ? 2 : 1.5"
          class="transition-colors duration-150"
        />
        <circle
          cx="14"
          :cy="nodeHeight / 2"
          r="4"
          :fill="node.id === currentId ? '#E4572E' : levelColor(node.level)"
        />
        <text
          x="26"
          :y="compact ? 20 : 23"
          font-size="13"
          font-weight="600"
          :fill="node.id === currentId ? '#FFFFFF' : '#0B1B2B'"
        >
          {{ labelOf(node) }}
        </text>
        <text
          x="26"
          :y="compact ? 37 : 42"
          font-size="11"
          :fill="node.id === currentId ? 'rgba(255,255,255,0.62)' : '#77828C'"
        >
          资源 {{ node.resource_count }} · 概念 {{ node.concept_count }}
        </text>
      </g>
    </svg>
  </div>
</template>

<style scoped>
.kg-node {
  outline: none;
}
.kg-node:focus-visible rect,
.kg-node:hover rect {
  stroke: #0b1b2b;
}
.kg-node:focus-visible rect {
  stroke-width: 2.5;
}
</style>
