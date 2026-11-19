<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { adminListResources } from '@/api/admin'
import { deleteResource } from '@/api/resource'
import { formatDate } from '@/utils/format'
import AppIcon from '@/components/ui/AppIcon.vue'
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
  <div class="mx-auto max-w-[1200px]">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 class="headline">资源管理</h1>
        <p class="mt-2 text-[13px] text-ink-secondary">维护平台教育资源</p>
      </div>
      <div class="flex flex-wrap items-center gap-2.5">
        <div class="w-52">
          <el-input v-model="keyword" aria-label="按标题搜索" placeholder="按标题搜索" clearable @keyup.enter="handleSearch">
            <template #prefix>
              <AppIcon name="search" :size="15" />
            </template>
          </el-input>
        </div>
        <div class="w-32">
          <el-select v-model="type" aria-label="按类型筛选" placeholder="全部类型" clearable @change="handleSearch">
            <el-option label="课程" value="course" />
            <el-option label="文章" value="article" />
            <el-option label="视频" value="video" />
          </el-select>
        </div>
        <RouterLink :to="{ name: 'resource-upload' }" class="btn btn-primary">
          <AppIcon name="plus" :size="15" />
          新增资源
        </RouterLink>
      </div>
    </div>

    <div v-if="loading" class="card mt-6 overflow-hidden p-5">
      <div class="space-y-3">
        <div v-for="i in 6" :key="i" class="skeleton h-10 w-full"></div>
      </div>
    </div>
    <div v-else-if="error" class="state-box mt-6">
      <span class="state-icon !bg-danger-soft !text-danger">
        <AppIcon name="alert" :size="22" />
      </span>
      <p class="state-title">资源加载失败</p>
      <p class="state-copy">{{ error }}</p>
      <button type="button" class="btn btn-secondary btn-sm mt-2" @click="fetchList">重新加载</button>
    </div>
    <div v-else class="card mt-6 overflow-hidden">
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
      <div v-if="total > pageSize" class="flex justify-center border-t border-border p-4">
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
