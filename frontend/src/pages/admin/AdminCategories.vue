<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createCategory, listCategories } from '@/api/category'
import AppIcon from '@/components/ui/AppIcon.vue'
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
  <div class="mx-auto max-w-[1000px]">
    <header>
      <h1 class="headline">分类管理</h1>
      <p class="mt-2 text-[13px] text-ink-secondary">维护教育资源分类</p>
    </header>

    <div v-if="loading" class="card mt-6 p-5">
      <div class="space-y-3">
        <div v-for="i in 5" :key="i" class="skeleton h-10 w-full"></div>
      </div>
    </div>
    <div v-else-if="error" class="state-box mt-6">
      <span class="state-icon !bg-danger-soft !text-danger">
        <AppIcon name="alert" :size="22" />
      </span>
      <p class="state-title">分类加载失败</p>
      <p class="state-copy">{{ error }}</p>
      <button type="button" class="btn btn-secondary btn-sm mt-2" @click="fetchList">重新加载</button>
    </div>
    <div v-else class="card mt-6 overflow-hidden">
      <el-table :data="items">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="description" label="描述" min-width="240" />
      </el-table>
    </div>

    <div class="card mt-6 p-5 sm:p-6">
      <div class="flex items-center gap-2.5">
        <span class="flex h-8 w-8 items-center justify-center rounded-[9px] bg-primary-soft text-primary">
          <AppIcon name="plus" :size="16" />
        </span>
        <div>
          <h2 class="text-[15px] font-bold text-ink">新建分类</h2>
          <p class="mt-0.5 text-[11.5px] text-ink-muted">分类用于资源归类与兴趣推荐</p>
        </div>
      </div>
      <div class="mt-4 flex flex-wrap items-start gap-3">
        <div class="w-full sm:w-48">
          <el-input v-model="name" aria-label="分类名称" placeholder="分类名称（必填）" clearable maxlength="64" />
        </div>
        <div class="w-full sm:w-80">
          <el-input v-model="description" aria-label="分类描述" placeholder="描述（选填）" clearable maxlength="256" />
        </div>
        <el-button type="primary" :loading="creating" @click="handleCreate">创建</el-button>
      </div>
      <p v-if="createError" class="mt-3 flex items-center gap-1.5 text-[12.5px] text-danger">
        <AppIcon name="alert" :size="14" />
        {{ createError }}
      </p>
    </div>
  </div>
</template>
