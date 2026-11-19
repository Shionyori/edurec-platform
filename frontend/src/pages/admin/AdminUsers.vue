<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { adminListUsers } from '@/api/admin'
import { formatDate } from '@/utils/format'
import AppIcon from '@/components/ui/AppIcon.vue'
import type { User } from '@/types'

const items = ref<User[]>([])
const loading = ref(false)
const error = ref('')
const total = ref(0)
const keyword = ref('')
const page = ref(1)
const pageSize = 20

async function fetchList() {
  loading.value = true
  error.value = ''
  try {
    const data = await adminListUsers({
      keyword: keyword.value.trim() || undefined,
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

onMounted(fetchList)
</script>

<template>
  <div class="mx-auto max-w-[1200px]">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 class="headline">用户管理</h1>
        <p class="mt-2 text-[13px] text-ink-secondary">查看平台注册用户</p>
      </div>
      <el-input
        v-model="keyword"
        aria-label="按用户名或邮箱搜索"
        placeholder="按用户名 / 邮箱搜索"
        clearable
        class="w-full sm:w-72"
        @keyup.enter="handleSearch"
      >
        <template #prefix>
          <AppIcon name="search" :size="15" />
        </template>
        <template #append>
          <el-button @click="handleSearch">搜索</el-button>
        </template>
      </el-input>
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
      <p class="state-title">用户加载失败</p>
      <p class="state-copy">{{ error }}</p>
      <button type="button" class="btn btn-secondary btn-sm mt-2" @click="fetchList">重新加载</button>
    </div>
    <div v-else class="card mt-6 overflow-hidden">
      <el-table :data="items">
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="username" label="用户名" min-width="120" />
        <el-table-column prop="email" label="邮箱" min-width="180" />
        <el-table-column prop="display_name" label="昵称" min-width="120" />
        <el-table-column label="角色" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.is_admin" type="warning" size="small">管理员</el-tag>
            <el-tag v-else type="info" size="small">普通用户</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="注册时间" min-width="160">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
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
