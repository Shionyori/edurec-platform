<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { adminGetRecommendationStats, adminListRecommendationRuns } from '@/api/admin'
import { formatDate } from '@/utils/format'
import type { RecommendationRun, RecommendationStats } from '@/types'

const stats = ref<RecommendationStats | null>(null)
const runs = ref<RecommendationRun[]>([])
const loading = ref(false)
const error = ref('')

function fromUnix(seconds: number): string {
  if (!seconds) return '-'
  return formatDate(new Date(seconds * 1000).toISOString())
}

function percent(rate: number): string {
  return `${(rate * 100).toFixed(2)}%`
}

onMounted(async () => {
  loading.value = true
  try {
    const [s, r] = await Promise.all([
      adminGetRecommendationStats(),
      adminListRecommendationRuns(20),
    ])
    stats.value = s
    runs.value = r.list
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="mx-auto max-w-[1200px]">
    <header>
      <h1 class="headline">推荐管理</h1>
      <p class="mt-2 text-[13px] text-ink-secondary">推荐效果看板与运行记录（可追溯）</p>
    </header>

    <div v-if="loading" class="mt-7 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
      <div v-for="i in 6" :key="i" class="card p-4">
        <div class="skeleton h-3 w-12"></div>
        <div class="skeleton mt-3 h-7 w-16"></div>
      </div>
    </div>
    <div v-else-if="error" class="state-box mt-6">
      <span class="state-icon !bg-danger-soft !text-danger">
        <AppIcon name="alert" :size="22" />
      </span>
      <p class="state-title">推荐数据加载失败</p>
      <p class="state-copy">{{ error }}</p>
    </div>
    <template v-else>
      <div class="mt-7 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        <div class="card p-4">
          <p class="text-[11.5px] font-semibold text-ink-muted">曝光</p>
          <p class="tnum mt-1.5 text-[26px] font-extrabold leading-none tracking-[-0.04em] text-ink">{{ stats?.impressions ?? 0 }}</p>
        </div>
        <div class="card p-4">
          <p class="text-[11.5px] font-semibold text-ink-muted">点击</p>
          <p class="tnum mt-1.5 text-[26px] font-extrabold leading-none tracking-[-0.04em] text-ink">{{ stats?.clicks ?? 0 }}</p>
        </div>
        <div class="card p-4">
          <p class="text-[11.5px] font-semibold text-ink-muted">点击率 CTR</p>
          <p class="tnum mt-1.5 text-[26px] font-extrabold leading-none tracking-[-0.04em] text-primary">{{ percent(stats?.click_through_rate ?? 0) }}</p>
        </div>
        <div class="card p-4">
          <p class="text-[11.5px] font-semibold text-ink-muted">收藏</p>
          <p class="tnum mt-1.5 text-[26px] font-extrabold leading-none tracking-[-0.04em] text-ink">{{ stats?.favorites ?? 0 }}</p>
        </div>
        <div class="card p-4">
          <p class="text-[11.5px] font-semibold text-ink-muted">浏览</p>
          <p class="tnum mt-1.5 text-[26px] font-extrabold leading-none tracking-[-0.04em] text-ink">{{ stats?.views ?? 0 }}</p>
        </div>
        <div class="card p-4">
          <p class="text-[11.5px] font-semibold text-ink-muted">推荐覆盖用户</p>
          <p class="tnum mt-1.5 text-[26px] font-extrabold leading-none tracking-[-0.04em] text-ink">{{ stats?.recommendation_users ?? 0 }}</p>
        </div>
      </div>

      <div class="mt-8 flex items-end justify-between gap-3">
        <div>
          <h2 class="text-[16px] font-bold text-ink">运行记录</h2>
          <p class="mt-1 text-[11.5px] text-ink-muted">每次导入 engine 推荐结果都落一条；来源信息取自旁挂信封（缺失则为空）</p>
        </div>
        <span class="chip tnum !text-[11px]">{{ runs.length }} 条</span>
      </div>

      <div class="card mt-3 overflow-hidden">
        <el-table :data="runs" empty-text="暂无运行记录">
          <el-table-column prop="run_id" label="运行 ID" min-width="150" />
          <el-table-column prop="model_name" label="模型" min-width="200" />
          <el-table-column prop="snapshot_run_id" label="数据快照" min-width="150" />
          <el-table-column label="生成时间" min-width="150">
            <template #default="{ row }">{{ fromUnix(row.generated_at) }}</template>
          </el-table-column>
          <el-table-column prop="imported_users" label="导入用户" width="100" />
          <el-table-column prop="skipped_users" label="跳过用户" width="100" />
          <el-table-column prop="imported_resources" label="导入条目" width="100" />
        </el-table>
      </div>
    </template>
  </div>
</template>
