<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { listCategories } from '@/api/category'
import { listResources } from '@/api/resource'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import type { Category, Resource } from '@/types'

const categories = ref<Category[]>([])
const resources = ref<Resource[]>([])
const loading = ref(false)
const error = ref('')
const total = ref(0)

const keyword = ref('')
const categoryId = ref<number | undefined>(undefined)
const type = ref('')
const sort = ref<'latest' | 'popular' | 'rating'>('latest')
const tags = ref('')

const page = ref(1)
const pageSize = 12

// 带关键词的搜索在本地无结果时会同步触发 B 站爬取，耗时数秒，用文案提示
const loadingText = computed(() =>
  keyword.value.trim() ? '正在搜索，本地无结果时会自动检索 B 站，请稍候…' : '加载中…',
)

async function fetchList() {
  loading.value = true
  error.value = ''
  try {
    const data = await listResources({
      page: page.value,
      page_size: pageSize,
      keyword: keyword.value.trim() || undefined,
      category_id: categoryId.value || undefined,
      type: type.value || undefined,
      sort: sort.value,
      tags: tags.value.trim() || undefined,
    })
    resources.value = data.list
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

onMounted(async () => {
  try {
    categories.value = await listCategories()
  } catch {
    // 分类加载失败不阻塞资源列表
  }
  fetchList()
})
</script>

<template>
  <div class="mx-auto max-w-7xl px-6 py-8">
    <h1 class="text-2xl font-bold">资源库</h1>
    <p class="mt-2 text-sm text-ink-secondary">搜索并筛选全部课程、文章与视频</p>

    <div class="mt-6 flex flex-wrap items-center gap-3 rounded-lg border border-border bg-surface p-4">
      <el-input v-model="keyword" placeholder="输入关键词…" clearable class="w-56" @keyup.enter="handleSearch" />
      <el-select v-model="categoryId" placeholder="全部分类" clearable class="w-40" @change="handleSearch">
        <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
      </el-select>
      <el-select v-model="type" placeholder="全部类型" clearable class="w-32" @change="handleSearch">
        <el-option label="课程" value="course" />
        <el-option label="文章" value="article" />
        <el-option label="视频" value="video" />
      </el-select>
      <el-select v-model="sort" class="w-32" @change="handleSearch">
        <el-option label="最新" value="latest" />
        <el-option label="热门" value="popular" />
        <el-option label="评分" value="rating" />
      </el-select>
      <el-input v-model="tags" placeholder="标签（逗号分隔）" clearable class="w-48" @keyup.enter="handleSearch" />
      <el-button type="primary" @click="handleSearch">搜索</el-button>
    </div>

    <div v-if="loading" class="py-24 text-center text-sm text-ink-muted">{{ loadingText }}</div>
    <div v-else-if="error" class="py-24 text-center text-sm text-red-500">{{ error }}</div>
    <div v-else-if="resources.length === 0" class="py-24 text-center text-sm text-ink-muted">没有符合条件的资源</div>
    <div v-else class="mt-6 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <ResourceCard v-for="r in resources" :key="r.id" :resource="r" />
    </div>

    <div v-if="total > pageSize" class="mt-8 flex justify-center">
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
</template>
