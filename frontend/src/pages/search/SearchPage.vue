<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listCategories } from '@/api/category'
import { listResources } from '@/api/resource'
import { startGenerationByKeyword, getConceptGenerationStatus } from '@/api/knowledge'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import ResourceCardSkeleton from '@/components/resource/ResourceCardSkeleton.vue'
import type { Category, GenerationStatus, Resource } from '@/types'

const PAGE_SIZE = 12

const route = useRoute()
const router = useRouter()

const categories = ref<Category[]>([])
const resources = ref<Resource[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const error = ref('')
const total = ref(0)
const hasMore = ref(true)
const exhausted = ref(false)

// 支持从顶栏搜索框跳转：/search?keyword=xxx（query 可能是 string | string[] | null）
function keywordFromQuery(): string {
  const raw = route.query.keyword
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === 'string' ? value.trim() : ''
}

// 支持从顶栏导航跳转：/search?type=video（course / article / video）
function typeFromQuery(): string {
  const raw = route.query.type
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === 'string' ? value.trim() : ''
}

const keyword = ref(keywordFromQuery())
const categoryId = ref<number | undefined>(undefined)
const type = ref(typeFromQuery())
const sort = ref<'latest' | 'popular' | 'rating'>('latest')
const tags = ref('')

const page = ref(1)
const onlinePage = ref(1)

// 搜索即构建：关键词命中知识点时，后台调用 engine 生成概念，前端轮询状态。
// 失败（无匹配 / engine 未启动）静默忽略，绝不阻塞搜索结果。
const generation = ref<{
  pointCode: string
  pointName: string
  status: GenerationStatus
} | null>(null)
let generationTimer: ReturnType<typeof setInterval> | null = null
let generationToken = 0

function stopGenerationPolling() {
  if (generationTimer) {
    clearInterval(generationTimer)
    generationTimer = null
  }
}

async function triggerGeneration(rawKeyword: string) {
  stopGenerationPolling()
  const token = ++generationToken
  generation.value = null
  const kw = rawKeyword.trim()
  if (!kw) return

  try {
    const result = await startGenerationByKeyword(kw)
    if (token !== generationToken) return
    generation.value = {
      pointCode: result.point_code,
      pointName: result.point_name,
      status: result.status,
    }
    if (result.status.status === 'running') {
      pollGeneration(result.point_code, token)
    }
  } catch {
    // 没有匹配的知识点或 engine 未启动：保持静默
  }
}

function pollGeneration(pointCode: string, token: number) {
  stopGenerationPolling()
  generationTimer = setInterval(async () => {
    if (token !== generationToken) {
      stopGenerationPolling()
      return
    }
    try {
      const status = await getConceptGenerationStatus(pointCode)
      if (token !== generationToken) return
      if (generation.value) generation.value.status = status
      if (status.status !== 'running') stopGenerationPolling()
    } catch {
      stopGenerationPolling()
    }
  }, 2000)
}

const TYPE_LABELS: Record<string, string> = { course: '课程', article: '文章', video: '视频' }
// 标题跟随顶栏导航的类型，明确当前在看哪一类
const pageTitle = computed(() => TYPE_LABELS[type.value] || '资源库')

// 仅纯关键词搜索（无分类/类型/标签筛选）才会在本地耗尽后爬 B 站
const canCrawlOnline = computed(
  () => !!keyword.value.trim() && !categoryId.value && !type.value && !tags.value.trim(),
)

// 带关键词的搜索在本地无结果时会同步触发外部来源抓取（目前是 B 站），耗时数秒，用文案提示。
// 不写死具体来源：将来接入别的在线来源时这里不用跟着改。
const loadingText = computed(() =>
  keyword.value.trim() ? '正在搜索，本地无结果时会自动检索外部内容，请稍候…' : '加载中…',
)

const sentinel = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null
// 上一次加载是否失败：失败时不自动续拉，否则哨兵仍在视口内会立即重试同一页而成环
let loadFailed = false
// 请求代次：重置搜索时自增。B 站爬取可能长达数十秒，期间用户改了关键词的话，
// 旧请求的返回不能再追加进新列表，也不能推进页码
let requestGeneration = 0
const isStale = (generation: number) => generation !== requestGeneration

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
  const generation = requestGeneration
  loading.value = true
  error.value = ''
  loadFailed = false
  try {
    const data = await listResources({ page: page.value, ...buildQuery() })
    if (isStale(generation)) return
    resources.value = data.list
    total.value = data.total
  } catch (e) {
    if (isStale(generation)) return
    error.value = e instanceof Error ? e.message : '加载失败'
    loadFailed = true
  } finally {
    // 旧请求不清理加载标志，复位由 resetAndSearch 负责，避免覆盖新搜索的状态
    if (!isStale(generation)) {
      loading.value = false
      refreshObservation()
    }
  }
}

// 追加结果按 id 去重：在线爬取会把判重命中的本地已有行一并回传，
// 直接 concat 会产生重复 id，触发 v-for 重复 key 并把同一张卡片渲染两次。
// 返回本次真正新增的条数，供调用方判断是否还有进展
function appendResources(list: Resource[]) {
  const seen = new Set(resources.value.map((r) => r.id))
  const fresh = list.filter((r) => !seen.has(r.id))
  resources.value = resources.value.concat(fresh)
  return fresh.length
}

async function fetchMoreLocal() {
  const generation = requestGeneration
  const nextPage = page.value + 1
  loadingMore.value = true
  loadFailed = false
  try {
    const data = await listResources({ page: nextPage, ...buildQuery() })
    if (isStale(generation)) return
    // 页码只在请求成功后推进：失败时保持原页码，否则下次滚动会跳过这一页的结果
    page.value = nextPage
    appendResources(data.list)
    total.value = data.total
    if (data.list.length === 0) exhausted.value = true
  } catch (e) {
    if (isStale(generation)) return
    error.value = e instanceof Error ? e.message : '加载失败'
    loadFailed = true
  } finally {
    if (!isStale(generation)) {
      loadingMore.value = false
      refreshObservation()
    }
  }
}

async function fetchOnline() {
  const generation = requestGeneration
  loadingMore.value = true
  loadFailed = false
  try {
    const data = await listResources({ online_page: onlinePage.value, ...buildQuery() })
    if (isStale(generation)) return
    const added = appendResources(data.list)
    hasMore.value = !!data.has_more
    // 只有本次请求仍然有效时才推进页码，否则会把新关键词的 B 站第 1 页跳过去
    onlinePage.value += 1
    // 没有新增（空页，或整页都是本地已有的判重命中）说明翻不出新内容，停止续拉
    if (added === 0) exhausted.value = true
  } catch {
    if (isStale(generation)) return
    // 在线爬取失败（风控/超时）不阻塞已加载内容，直接到底
    exhausted.value = true
  } finally {
    if (!isStale(generation)) {
      loadingMore.value = false
      refreshObservation()
    }
  }
}

async function loadMore() {
  if (loading.value || loadingMore.value || exhausted.value) return
  // 本地还有下一页
  if (resources.value.length < total.value) {
    await fetchMoreLocal()
    return
  }
  // 本地耗尽：纯关键词搜索且 B 站可能还有 → 逐页爬取
  if (canCrawlOnline.value && hasMore.value) {
    await fetchOnline()
    return
  }
  exhausted.value = true
}

function resetAndSearch() {
  // 旧请求的 finally 会因代次不符而跳过清理，故此处显式复位加载标志
  requestGeneration += 1
  page.value = 1
  onlinePage.value = 1
  resources.value = []
  total.value = 0
  hasMore.value = true
  exhausted.value = false
  loadingMore.value = false
  fetchLocal()
  // 输入即构建：每次重置搜索都尝试为关键词匹配知识点并生成概念
  triggerGeneration(keyword.value)
}

// 顶栏搜索框再次提交（改写 query）时，同步关键词并重新搜索；
// resetAndSearch 内部会自增 requestGeneration，在飞的旧请求返回后会被丢弃。
watch(
  () => route.query.keyword,
  () => {
    const kw = keywordFromQuery()
    if (kw === keyword.value) return
    keyword.value = kw
    resetAndSearch()
  },
)

// 顶栏导航切换类型（课程/视频/文章）时，同步筛选并重新搜索
watch(
  () => route.query.type,
  () => {
    const next = typeFromQuery()
    if (next === type.value) return
    type.value = next
    resetAndSearch()
  },
)

function handleSearch() {
  resetAndSearch()
}

const hasFilters = computed(
  () => !!keyword.value.trim() || !!categoryId.value || !!type.value || !!tags.value.trim(),
)

function clearFilters() {
  keyword.value = ''
  categoryId.value = undefined
  type.value = ''
  tags.value = ''
  sort.value = 'latest'
  // 同步清掉 URL 上的筛选，刷新/分享时不会又回到旧筛选
  if (Object.keys(route.query).length > 0) {
    router.replace({ name: 'search', query: {} })
  }
  resetAndSearch()
}

// IntersectionObserver 只在交叉状态发生变化时回调：哨兵一直停在视口内（内容不足一屏）
// 时不会再次触发，且 observe() 后的初始通知会被 loading 守卫挡掉。每次加载结束后重新
// observe，让浏览器按当前状态重新投递一次通知，无限滚动与在线爬取才能自启动并继续续拉。
function refreshObservation() {
  if (!observer || !sentinel.value || loadFailed || exhausted.value) return
  observer.unobserve(sentinel.value)
  observer.observe(sentinel.value)
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
  // 首次进入（含从顶栏带 keyword 跳转）也要触发生成，不能只在重置搜索时触发
  triggerGeneration(keyword.value)
})

onBeforeUnmount(() => {
  observer?.disconnect()
  stopGenerationPolling()
  generationToken += 1
})
</script>

<template>
  <div class="mx-auto max-w-7xl px-4 py-8 sm:px-6">
    <div class="flex flex-wrap items-end justify-between gap-3 border-b border-border pb-4">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-ink sm:text-3xl">{{ pageTitle }}</h1>
        <p class="mt-2 text-sm text-ink-secondary">搜索并筛选全部课程、文章与视频</p>
      </div>
      <span v-if="total > 0" class="data-num text-xs text-ink-muted">共 {{ total }} 条</span>
    </div>

    <div
      v-if="generation"
      class="mt-4 flex flex-wrap items-center gap-2 rounded-[4px] border border-border bg-surface px-4 py-3 text-sm"
    >
      <template v-if="generation.status.status === 'running'">
        <span class="route-node" style="--node: var(--color-line-amber)" aria-hidden="true"></span>
        <span class="text-ink-secondary">
          正在为「{{ generation.pointName }}」自动构建概念图谱…
        </span>
      </template>
      <template v-else-if="generation.status.status === 'done'">
        <span class="route-node" style="--node: var(--color-success)" aria-hidden="true"></span>
        <span class="text-ink-secondary">
          已为「{{ generation.pointName }}」补充 {{ generation.status.generated }} 个概念（共
          {{ generation.status.total }} 个）。
        </span>
        <RouterLink
          :to="{ name: 'knowledge-detail', params: { code: generation.pointCode } }"
          class="font-medium text-primary underline-offset-4 hover:underline"
        >查看知识点 →</RouterLink>
      </template>
      <template v-else-if="generation.status.status === 'failed'">
        <span class="route-node" style="--node: var(--color-danger)" aria-hidden="true"></span>
        <span class="text-danger">
          概念生成失败：{{ generation.status.error || 'engine 未启动' }}（搜索不受影响）
        </span>
      </template>
    </div>

    <div class="relative mt-5 overflow-hidden rounded-[4px] border border-border bg-surface p-4 md:sticky md:top-16 md:z-30">
      <span class="absolute inset-x-0 top-0 h-[2px]" style="background: var(--color-line-teal)" aria-hidden="true"></span>
      <div class="flex flex-wrap items-center gap-2.5">
        <div class="w-full sm:w-56">
          <el-input
            v-model="keyword"
            placeholder="输入关键词…"
            clearable
            @keyup.enter="handleSearch"
          />
        </div>
        <div class="w-36">
          <el-select v-model="categoryId" placeholder="全部分类" clearable @change="handleSearch">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </div>
        <div class="w-32">
          <el-select v-model="type" placeholder="全部类型" clearable @change="handleSearch">
            <el-option label="课程" value="course" />
            <el-option label="文章" value="article" />
            <el-option label="视频" value="video" />
          </el-select>
        </div>
        <div class="w-32">
          <el-select v-model="sort" @change="handleSearch">
            <el-option label="最新" value="latest" />
            <el-option label="热门" value="popular" />
            <el-option label="评分" value="rating" />
          </el-select>
        </div>
        <div class="w-44">
          <el-input v-model="tags" placeholder="标签（逗号分隔）" clearable @keyup.enter="handleSearch" />
        </div>
        <el-button type="primary" @click="handleSearch">搜索</el-button>
        <button
          v-if="hasFilters"
          type="button"
          class="ml-auto text-xs text-ink-muted underline-offset-4 hover:text-primary hover:underline"
          @click="clearFilters"
        >清除筛选</button>
      </div>
    </div>

    <div v-if="loading" class="mt-5">
      <p class="py-3 text-center text-sm text-ink-muted">{{ loadingText }}</p>
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
        <ResourceCardSkeleton v-for="i in 6" :key="i" />
      </div>
    </div>

    <div v-else-if="error && resources.length === 0" class="mt-5 flex flex-col items-center gap-3 rounded-[4px] border border-border bg-surface px-6 py-16 text-center">
      <p class="text-sm text-danger">{{ error }}</p>
      <el-button @click="handleSearch">重新搜索</el-button>
    </div>

    <div v-else-if="resources.length === 0 && !loadingMore" class="mt-5 flex flex-col items-center gap-3 rounded-[4px] border border-dashed border-border px-6 py-16 text-center">
      <p class="text-sm font-medium text-ink">没有符合条件的资源</p>
      <p class="text-xs text-ink-muted">
        {{ hasFilters ? '换个关键词，或清除筛选后再试。' : '资源准备好后会出现在这里。' }}
      </p>
      <el-button v-if="hasFilters" @click="clearFilters">清除筛选</el-button>
    </div>

    <div v-else class="mt-5 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <ResourceCard
        v-for="(r, i) in resources"
        :key="r.id"
        :resource="r"
        :style="{ animationDelay: `${Math.min(i, 8) * 25}ms` }"
      />
    </div>

    <div v-if="loadingMore" class="flex items-center justify-center gap-2 py-8 text-sm text-ink-muted">
      <span class="route-node" style="--node: var(--color-line-teal)" aria-hidden="true"></span>
      正在加载…
    </div>
    <div v-else-if="exhausted && resources.length > 0" class="py-8 text-center text-sm text-ink-muted">
      没有更多内容
    </div>
    <div ref="sentinel" class="h-px"></div>
  </div>
</template>
