<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listCategories } from '@/api/category'
import { listDisciplines } from '@/api/discipline'
import { useAuthStore } from '@/stores/auth'
import type { Category, Discipline } from '@/types'

const emit = defineEmits<{ saved: [] }>()

const auth = useAuthStore()

const categories = ref<Category[]>([])
// 分类数据可能很少；学科是另一条真实的入口，给新用户更多可选方向
const disciplines = ref<Discipline[]>([])
const selected = ref<number[]>([])
const loading = ref(false)
const saving = ref(false)

onMounted(async () => {
  loading.value = true
  // 分类与学科互不依赖：任一失败都不阻塞页面主体
  const [cats, discs] = await Promise.allSettled([listCategories(), listDisciplines()])
  if (cats.status === 'fulfilled') categories.value = cats.value
  if (discs.status === 'fulfilled') disciplines.value = discs.value
  loading.value = false
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
  <div class="mt-6 overflow-hidden rounded-[4px] border border-border bg-surface">
    <div class="flex flex-wrap items-start justify-between gap-3 border-b border-border bg-bg/60 px-5 py-4">
      <div>
        <h2 class="text-base font-semibold text-ink">选择你感兴趣的方向</h2>
        <p class="mt-1 text-sm text-ink-secondary">
          新用户还没有学习记录，先告诉我们你的兴趣，推荐会更贴近你的需求
        </p>
      </div>
      <span class="line-chip" style="--chip: var(--color-line-amber)">冷启动</span>
    </div>
    <div class="px-5 py-4">
      <div v-if="loading" class="text-sm text-ink-muted">加载中…</div>
      <el-checkbox-group v-else v-model="selected" class="flex flex-wrap gap-2">
        <el-checkbox-button v-for="c in categories" :key="c.id" :value="c.id">
          {{ c.name }}
        </el-checkbox-button>
      </el-checkbox-group>
      <div class="mt-4 flex flex-wrap items-center gap-3">
        <el-button type="primary" :loading="saving" @click="save">保存并刷新推荐</el-button>
        <span class="text-xs text-ink-muted">至少选择一个方向</span>
      </div>

      <div v-if="disciplines.length > 0" class="mt-4 border-t border-border pt-3">
        <p class="text-xs text-ink-muted">也可以直接按学科开始：</p>
        <div class="mt-2 flex flex-wrap gap-1.5">
          <RouterLink
            v-for="discipline in disciplines"
            :key="discipline.code"
            :to="{ name: 'discipline', params: { code: discipline.code } }"
            class="rounded-[3px] border border-border px-2 py-1 text-xs text-ink-secondary transition-colors hover:border-ink/30 hover:text-primary"
          >{{ discipline.name }}</RouterLink>
        </div>
      </div>
    </div>
  </div>
</template>
