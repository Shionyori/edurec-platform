<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { listCategories } from '@/api/category'
import { listResources } from '@/api/resource'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import type { Category, Resource } from '@/types'

const PAGE_SIZE = 12

const categories = ref<Category[]>([])
const resources = ref<Resource[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const error = ref('')
const total = ref(0)
const hasMore = ref(true)
const exhausted = ref(false)

const keyword = ref('')
const categoryId = ref<number | undefined>(undefined)
const type = ref('')
const sort = ref<'latest' | 'popular' | 'rating'>('latest')
const tags = ref('')

const page = ref(1)
const onlinePage = ref(1)

// 仅纯关键词搜索（无分类/类型/标签筛选）才会在本地耗尽后爬 B 站
const canCrawlOnline = computed(
  () => !!keyword.value.trim() && !categoryId.value && !type.value && !tags.value.trim(),
)

// 带关键词的搜索在本地无结果时会同步触发 B 站爬取，耗时数秒，用文案提示
const loadingText = computed(() =>
  keyword.value.trim() ? '正在搜索，本地无结果时会自动检索 B 站，请稍候…' : '加载中…',
)

const sentinel = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null

function buildQuery() {
  return {
    page_size: PAGE_SIZE,
    keyword: keyword.value.trim() || undefined,
    category_id: categoryId.value || undefined,
    type: type.value || undefined,
    sort: sort.value,
    tags: tags.value.trim() || undefined,
  }
}

async function fetchLocal() {
  loading.value = true
  error.value = ''
  try {
    const data = await listResources({ page: page.value, ...buildQuery() })
    resources.value = data.list
    total.value = data.total
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

async function fetchMoreLocal() {
  loadingMore.value = true
  try {
    const data = await listResources({ page: page.value, ...buildQuery() })
    resources.value = resources.value.concat(data.list)
    total.value = data.total
    if (data.list.length === 0) exhausted.value = true
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loadingMore.value = false
  }
}

async function fetchOnline() {
  loadingMore.value = true
  try {
    const data = await listResources({ online_page: onlinePage.value, ...buildQuery() })
    resources.value = resources.value.concat(data.list)
    hasMore.value = !!data.has_more
    if (data.list.length === 0) exhausted.value = true
  } catch {
    // 在线爬取失败（风控/超时）不阻塞已加载内容，直接到底
    exhausted.value = true
  } finally {
    loadingMore.value = false
  }
}

async function loadMore() {
  if (loading.value || loadingMore.value || exhausted.value) return
  // 本地还有下一页
  if (resources.value.length < total.value) {
    page.value += 1
    await fetchMoreLocal()
    return
  }
  // 本地耗尽：纯关键词搜索且 B 站可能还有 → 逐页爬取
  if (canCrawlOnline.value && hasMore.value) {
    await fetchOnline()
    onlinePage.value += 1
    return
  }
  exhausted.value = true
}

function resetAndSearch() {
  page.value = 1
  onlinePage.value = 1
  resources.value = []
  total.value = 0
  hasMore.value = true
  exhausted.value = false
  fetchLocal()
}

function handleSearch() {
  resetAndSearch()
}

function setupObserver() {
  if (!sentinel.value || typeof IntersectionObserver === 'undefined') return
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((e) => e.isIntersecting)) loadMore()
    },
    { rootMargin: '200px' },
  )
  observer.observe(sentinel.value)
}

onMounted(async () => {
  try {
    categories.value = await listCategories()
  } catch {
    // 分类加载失败不阻塞资源列表
  }
  fetchLocal()
  setupObserver()
})

onBeforeUnmount(() => {
  observer?.disconnect()
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
    <div v-else-if="error && resources.length === 0" class="py-24 text-center text-sm text-red-500">{{ error }}</div>
    <div v-else-if="resources.length === 0 && !loadingMore" class="py-24 text-center text-sm text-ink-muted">
      没有符合条件的资源
    </div>
    <div v-else class="mt-6 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
      <ResourceCard v-for="r in resources" :key="r.id" :resource="r" />
    </div>

    <div v-if="loadingMore" class="py-8 text-center text-sm text-ink-muted">正在加载…</div>
    <div v-else-if="exhausted && resources.length > 0" class="py-8 text-center text-sm text-ink-muted">
      没有更多内容
    </div>
    <div ref="sentinel" class="h-px"></div>
  </div>
</template>
