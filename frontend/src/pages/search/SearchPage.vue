<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listCategories } from '@/api/category'
import { listResources } from '@/api/resource'
import { startGenerationByKeyword, getConceptGenerationStatus } from '@/api/knowledge'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import ResourceCardSkeleton from '@/components/resource/ResourceCardSkeleton.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
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
  <div class="page shell">
    <header class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 class="headline">{{ pageTitle }}</h1>
        <p class="mt-2 text-[13.5px] text-ink-secondary">搜索并筛选全部课程、文章与视频</p>
      </div>
      <span v-if="total > 0" class="chip tnum">
        <AppIcon name="layers" :size="13" />
        共 {{ total }} 条
      </span>
    </header>

    <div
      v-if="generation"
      class="mt-5 flex flex-wrap items-center gap-2.5 rounded-[12px] border border-border bg-surface px-4 py-3 text-[13px]"
    >
      <template v-if="generation.status.status === 'running'">
        <span class="spinner-dots" aria-hidden="true"><span></span><span></span><span></span></span>
        <span class="text-ink-secondary">
          正在为「{{ generation.pointName }}」自动构建概念图谱…
        </span>
      </template>
      <template v-else-if="generation.status.status === 'done'">
        <span class="flex h-6 w-6 items-center justify-center rounded-md bg-success-soft text-success">
          <AppIcon name="check" :size="14" />
        </span>
        <span class="text-ink-secondary">
          已为「{{ generation.pointName }}」补充 {{ generation.status.generated }} 个概念（共
          {{ generation.status.total }} 个）。
        </span>
        <RouterLink
          :to="{ name: 'knowledge-detail', params: { code: generation.pointCode } }"
          class="link-arrow !text-[13px]"
        >查看知识点 <AppIcon name="arrow-right" :size="13" /></RouterLink>
      </template>
      <template v-else-if="generation.status.status === 'failed'">
        <span class="flex h-6 w-6 items-center justify-center rounded-md bg-danger-soft text-danger">
          <AppIcon name="alert" :size="14" />
        </span>
        <span class="text-danger">
          概念生成失败：{{ generation.status.error || 'engine 未启动' }}（搜索不受影响）
        </span>
      </template>
    </div>

    <!-- Filter toolbar: sticky under the header, one row on desktop -->
    <div class="panel mt-5 p-4 md:sticky md:top-[73px] md:z-30 md:shadow-[0_1px_2px_rgba(11,27,43,0.05)]">
      <div class="flex flex-wrap items-center gap-2.5">
        <div class="w-full sm:w-60">
          <el-input
            v-model="keyword"
            aria-label="关键词"
            placeholder="输入关键词…"
            clearable
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <AppIcon name="search" :size="16" />
            </template>
          </el-input>
        </div>
        <div class="w-[calc(50%-5px)] sm:w-36">
          <el-select v-model="categoryId" aria-label="按分类筛选" placeholder="全部分类" clearable @change="handleSearch">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </div>
        <div class="w-[calc(50%-5px)] sm:w-32">
          <el-select v-model="type" aria-label="按类型筛选" placeholder="全部类型" clearable @change="handleSearch">
            <el-option label="课程" value="course" />
            <el-option label="文章" value="article" />
            <el-option label="视频" value="video" />
          </el-select>
        </div>
        <div class="w-[calc(50%-5px)] sm:w-32">
          <el-select v-model="sort" aria-label="排序方式" @change="handleSearch">
            <el-option label="最新" value="latest" />
            <el-option label="热门" value="popular" />
            <el-option label="评分" value="rating" />
          </el-select>
        </div>
        <div class="w-[calc(50%-5px)] sm:w-44">
          <el-input v-model="tags" aria-label="标签筛选" placeholder="标签（逗号分隔）" clearable @keyup.enter="handleSearch" />
        </div>
        <button type="button" class="btn btn-primary shrink-0" @click="handleSearch">
          <AppIcon name="search" :size="15" />
          搜索
        </button>
        <button
          v-if="hasFilters"
          type="button"
          class="ml-auto text-[12.5px] font-semibold text-ink-muted underline-offset-4 hover:text-primary hover:underline"
          @click="clearFilters"
        >清除筛选</button>
      </div>
    </div>

    <div class="mt-7 flex flex-wrap items-end justify-between gap-2">
      <h2 class="text-[18px] font-bold text-ink">资源列表</h2>
      <span v-if="total > 0" class="tnum text-[12px] text-ink-muted">共 {{ total }} 条</span>
    </div>

    <div v-if="loading" class="mt-4">
      <p class="flex items-center justify-center gap-2 py-3 text-[13px] text-ink-muted">
        <span class="spinner-dots" aria-hidden="true"><span></span><span></span><span></span></span>
        {{ loadingText }}
      </p>
      <div class="mt-5 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
        <ResourceCardSkeleton v-for="i in 6" :key="i" />
      </div>
    </div>

    <div v-else-if="error && resources.length === 0" class="state-box mt-4">
      <span class="state-icon !bg-danger-soft !text-danger">
        <AppIcon name="alert" :size="22" />
      </span>
      <p class="state-title">搜索失败</p>
      <p class="state-copy">{{ error }}</p>
      <button type="button" class="btn btn-secondary btn-sm mt-2" @click="handleSearch">重新搜索</button>
    </div>

    <div v-else-if="resources.length === 0 && !loadingMore" class="state-box mt-4">
      <span class="state-icon">
        <AppIcon name="search" :size="22" />
      </span>
      <p class="state-title">没有符合条件的资源</p>
      <p class="state-copy">
        {{ hasFilters ? '换个关键词，或清除筛选后再试。' : '资源准备好后会出现在这里。' }}
      </p>
      <button v-if="hasFilters" type="button" class="btn btn-secondary btn-sm mt-2" @click="clearFilters">
        清除筛选
      </button>
    </div>

    <div v-else class="mt-4 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <ResourceCard
        v-for="(r, i) in resources"
        :key="r.id"
        :resource="r"
        class="fade-up"
        :style="{ animationDelay: `${Math.min(i, 8) * 25}ms` }"
      />
    </div>

    <div v-if="loadingMore" class="flex items-center justify-center gap-2.5 py-10 text-[13px] text-ink-muted">
      <span class="spinner-dots" aria-hidden="true"><span></span><span></span><span></span></span>
      正在加载更多
    </div>
    <div v-else-if="exhausted && resources.length > 0" class="flex items-center justify-center gap-2 py-10 text-[12.5px] text-ink-muted">
      <span class="h-px w-12 bg-border"></span>
      没有更多内容
      <span class="h-px w-12 bg-border"></span>
    </div>
    <div ref="sentinel" class="h-px"></div>
  </div>
</template>
