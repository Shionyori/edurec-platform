<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  initialScore?: number
  initialComment?: string
  submitting?: boolean
}>()

const emit = defineEmits<{
  submit: [payload: { score: number; comment: string }]
}>()

const score = ref(props.initialScore ?? 0)
const comment = ref(props.initialComment ?? '')

watch(
  () => [props.initialScore, props.initialComment] as const,
  ([s, c]) => {
    score.value = s ?? 0
    comment.value = c ?? ''
  },
)

function onSubmit() {
  if (score.value < 1) {
    ElMessage.warning('请先选择评分')
    return
  }
  emit('submit', { score: score.value, comment: comment.value.trim() })
}
</script>

<template>
  <div class="space-y-4">
    <div>
      <div class="mb-1 text-sm text-ink-secondary">你的评分</div>
      <el-rate v-model="score" :max="5" />
    </div>
    <el-input
      v-model="comment"
      type="textarea"
      :rows="3"
      maxlength="500"
      show-word-limit
      placeholder="说说你的学习感受（选填）"
    />
    <el-button type="primary" class="w-full" :loading="submitting" @click="onSubmit">
      提交评价
    </el-button>
  </div>
</template>
