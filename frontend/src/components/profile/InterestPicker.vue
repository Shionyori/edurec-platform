<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listCategories } from '@/api/category'
import { useAuthStore } from '@/stores/auth'
import type { Category } from '@/types'

const emit = defineEmits<{ saved: [] }>()

const auth = useAuthStore()

const categories = ref<Category[]>([])
const selected = ref<number[]>([])
const loading = ref(false)
const saving = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    categories.value = await listCategories()
  } catch {
    // 分类加载失败不阻塞页面主体
  } finally {
    loading.value = false
  }
})

async function save() {
  if (selected.value.length === 0) {
    ElMessage.warning('请至少选择一个方向')
    return
  }
  saving.value = true
  try {
    await auth.setInterests(selected.value)
    ElMessage.success('已保存兴趣方向')
    emit('saved')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败，请重试')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="mt-6 rounded-lg border border-primary/30 bg-primary/5 p-5">
    <h2 class="text-base font-semibold text-ink">选择你感兴趣的方向</h2>
    <p class="mt-1 text-sm text-ink-secondary">
      新用户还没有学习记录，先告诉我们你的兴趣，推荐会更贴近你的需求（冷启动）
    </p>
    <div v-if="loading" class="mt-3 text-sm text-ink-muted">加载中…</div>
    <el-checkbox-group v-else v-model="selected" class="mt-3 flex flex-wrap gap-2">
      <el-checkbox-button v-for="c in categories" :key="c.id" :value="c.id">
        {{ c.name }}
      </el-checkbox-button>
    </el-checkbox-group>
    <div class="mt-4">
      <el-button type="primary" :loading="saving" @click="save">保存并刷新推荐</el-button>
    </div>
  </div>
</template>
