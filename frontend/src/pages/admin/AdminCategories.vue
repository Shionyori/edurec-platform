<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createCategory, listCategories } from '@/api/category'
import type { Category } from '@/types'

const items = ref<Category[]>([])
const loading = ref(false)
const error = ref('')
const creating = ref(false)
const createError = ref('')
const name = ref('')
const description = ref('')

async function fetchList() {
  loading.value = true
  error.value = ''
  try {
    items.value = await listCategories()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!name.value.trim()) {
    createError.value = '分类名称不能为空'
    return
  }
  creating.value = true
  createError.value = ''
  try {
    await createCategory({
      name: name.value.trim(),
      description: description.value.trim() || undefined,
    })
    name.value = ''
    description.value = ''
    fetchList()
  } catch (e) {
    createError.value = e instanceof Error ? e.message : '创建失败'
  } finally {
    creating.value = false
  }
}

onMounted(fetchList)
</script>

<template>
  <div>
    <h1 class="text-xl font-bold">分类管理</h1>
    <p class="mt-1 text-sm text-ink-secondary">维护教育资源分类</p>

    <div v-if="loading" class="py-16 text-center text-sm text-ink-muted">加载中…</div>
    <div v-else-if="error" class="py-16 text-center text-sm text-red-500">{{ error }}</div>
    <div v-else class="mt-4 rounded-lg border border-border bg-surface">
      <el-table :data="items">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="description" label="描述" min-width="240" />
      </el-table>
    </div>

    <div class="mt-6 rounded-lg border border-border bg-surface p-5">
      <h2 class="text-sm font-semibold text-ink">新建分类</h2>
      <div class="mt-3 flex flex-wrap items-start gap-3">
        <el-input v-model="name" placeholder="分类名称（必填）" clearable class="w-48" maxlength="64" />
        <el-input v-model="description" placeholder="描述（选填）" clearable class="w-80" maxlength="256" />
        <el-button type="primary" :loading="creating" @click="handleCreate">创建</el-button>
      </div>
      <p v-if="createError" class="mt-2 text-sm text-red-500">{{ createError }}</p>
    </div>
  </div>
</template>
