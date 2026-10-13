<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  initialGoal?: string
  submitLabel?: string
  loading?: boolean
}>()

const emit = defineEmits<{ submit: [goal: string] }>()

// 与后端 service 的 minGoalRunes / maxGoalRunes 对齐
const MIN_GOAL_LENGTH = 2
const MAX_GOAL_LENGTH = 100

const goal = ref(props.initialGoal ?? '')

// 目标同步到 URL 后，浏览器前进/后退或直接带 ?goal= 打开时要回填输入框
watch(
  () => props.initialGoal,
  (next) => {
    goal.value = next ?? ''
  },
)

function submit() {
  const trimmed = goal.value.trim()
  if (trimmed.length < MIN_GOAL_LENGTH) {
    ElMessage.warning('请输入学习目标')
    return
  }
  if (trimmed.length > MAX_GOAL_LENGTH) {
    ElMessage.warning(`学习目标请精简到 ${MAX_GOAL_LENGTH} 字以内`)
    return
  }
  emit('submit', trimmed)
}
</script>

<template>
  <form class="flex flex-col gap-3 sm:flex-row" @submit.prevent="submit">
    <el-input
      v-model="goal"
      size="large"
      maxlength="100"
      placeholder="例如：我想入门机器学习"
      aria-label="学习目标"
    />
    <el-button type="primary" size="large" native-type="submit" :loading="props.loading">
      {{ props.submitLabel ?? '开始规划' }}
    </el-button>
  </form>
</template>
