<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { adminListResources } from '@/api/admin'
import { deleteResource } from '@/api/resource'
import { formatDate } from '@/utils/format'
import type { Resource } from '@/types'

const items = ref<Resource[]>([])
const loading = ref(false)
const error = ref('')
const total = ref(0)
const keyword = ref('')
const type = ref('')
const page = ref(1)
const pageSize = 20

async function fetchList() {
  loading.value = true
  error.value = ''
  try {
    const data = await adminListResources({
      keyword: keyword.value.trim() || undefined,
      type: type.value || undefined,
      page: page.value,
      page_size: pageSize,
    })
    items.value = data.list
    total.value = data.total
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchList()
}

function handlePageChange(p: number) {
  page.value = p
  fetchList()
}

async function handleDelete(id: number) {
  try {
    await deleteResource(id)
    fetchList()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '删除失败'
  }
}

onMounted(fetchList)
</script>

<template>
  <div>
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-bold">资源管理</h1>
        <p class="mt-1 text-sm text-ink-secondary">维护平台教育资源</p>
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <el-input v-model="keyword" placeholder="按标题搜索" clearable class="w-52" @keyup.enter="handleSearch" />
        <el-select v-model="type" placeholder="全部类型" clearable class="w-32" @change="handleSearch">
          <el-option label="课程" value="course" />
          <el-option label="文章" value="article" />
          <el-option label="视频" value="video" />
        </el-select>
        <RouterLink :to="{ name: 'resource-upload' }">
          <el-button type="primary">新增资源</el-button>
        </RouterLink>
      </div>
    </div>

    <div v-if="loading" class="py-16 text-center text-sm text-ink-muted">加载中…</div>
    <div v-else-if="error" class="py-16 text-center text-sm text-red-500">{{ error }}</div>
    <div v-else class="mt-4 rounded-lg border border-border bg-surface">
      <el-table :data="items">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
        <el-table-column label="类型" width="90">
          <template #default="{ row }">
            <el-tag
              size="small"
              :type="row.type === 'course' ? 'primary' : row.type === 'video' ? 'success' : 'warning'"
            >
              {{ row.type }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="分类" min-width="110">
          <template #default="{ row }">{{ row.category?.name ?? '-' }}</template>
        </el-table-column>
        <el-table-column prop="avg_rating" label="评分" width="80" />
        <el-table-column prop="view_count" label="浏览" width="90" />
        <el-table-column label="创建时间" min-width="150">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-popconfirm
              title="确认删除该资源？"
              confirm-button-text="删除"
              cancel-button-text="取消"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <el-button size="small" type="danger" text>删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="total > pageSize" class="flex justify-center p-4">
        <el-pagination
          background
          layout="prev, pager, next"
          :total="total"
          :page-size="pageSize"
          :current-page="page"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>
