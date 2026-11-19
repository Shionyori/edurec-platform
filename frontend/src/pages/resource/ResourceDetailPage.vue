<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getResource } from '@/api/resource'
import { listRatings, upsertRating } from '@/api/rating'
import { listComments } from '@/api/comment'
import { recordBehavior, setFavorite } from '@/api/behavior'
import { useAuthStore } from '@/stores/auth'
import { formatDate, formatUnixDate, difficultyLabel, formatDuration } from '@/utils/format'
import type { BilibiliComment, Rating, Resource } from '@/types'
import RatingForm from '@/components/resource/RatingForm.vue'
import RatingList from '@/components/resource/RatingList.vue'
import CommentList from '@/components/resource/CommentList.vue'
import KeyframeStrip, { type VideoFrames } from '@/components/resource/KeyframeStrip.vue'
import AppIcon from '@/components/ui/AppIcon.vue'

const PAGE_SIZE = 10

const route = useRoute()
const auth = useAuthStore()
const resourceId = computed(() => Number(route.params.id))

const resource = ref<Resource | null>(null)
const resourceLoading = ref(true)
const resourceError = ref('')

// 外链封面（如 B 站）可能失效：no-referrer 已规避防盗链，但仍需对加载失败做降级
const coverFailed = ref(false)

// 收藏状态：以服务端 favorite 行为为准，进页面时由详情接口回填；
// 切换即时反馈（乐观更新），失败则回滚。
const favorited = ref(false)
const favoritePending = ref(false)

const ratings = ref<Rating[]>([])
const total = ref(0)
const page = ref(1)
const ratingsLoading = ref(false)

const comments = ref<BilibiliComment[]>([])
const commentsLoading = ref(false)

const submitting = ref(false)
const initialScore = ref(0)
const initialComment = ref('')
let prefilled = false

// 只有 B 站来源才有可爬取的评论。判据是「来源是 B 站」而非「类型是 video」：
// 长合集会被归入 course（见 docs/guides/bilibili-import.md），若绑在 type 上这些课程就没了评论区。
const isBilibiliResource = computed(
  () => !!resource.value?.source_url?.includes('bilibili.com'),
)

// B 站关键帧（雪碧图 + 网格参数）：fetch_frames 会写进 metadata.frames
const frames = computed<VideoFrames | null>(() => {
  const raw = resource.value?.metadata?.frames
  if (!raw || typeof raw !== 'object') return null
  const candidate = raw as Partial<VideoFrames>
  if (!Array.isArray(candidate.sprites) || candidate.sprites.length === 0) return null
  return {
    bvid: candidate.bvid,
    sprites: candidate.sprites,
    x_len: candidate.x_len || 10,
    y_len: candidate.y_len || 10,
    x_size: candidate.x_size,
    y_size: candidate.y_size,
    count: candidate.count,
    source: candidate.source,
  }
})

// 关键帧由独立区块展示，元信息表里不再重复渲染 frames
const metaEntries = computed(() => {
  const metadata = resource.value?.metadata ?? {}
  return Object.entries(metadata).filter(([key]) => key !== 'frames')
})

async function loadResource() {
  resourceLoading.value = true
  resourceError.value = ''
  try {
    resource.value = await getResource(resourceId.value)
    favorited.value = resource.value.favorited ?? false
    // 资源加载成功才上报 view（404 时不产生行为记录）
    recordBehavior(resourceId.value, 'view').catch(() => {})
    if (isBilibiliResource.value) {
      loadComments()
    }
  } catch (e) {
    resourceError.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    resourceLoading.value = false
  }
}

async function loadComments() {
  comments.value = []
  commentsLoading.value = true
  try {
    const data = await listComments(resourceId.value)
    comments.value = data.list
  } catch {
    // 评论抓取失败不阻塞页面主体，展示空态即可
  } finally {
    commentsLoading.value = false
  }
}

// 仅初始加载回填一次"我的评分"；提交后不再回填（走重置）
function prefillMyRating() {
  if (prefilled) return
  const mine = ratings.value.find((r) => r.user.id === auth.user?.id)
  if (mine) {
    initialScore.value = mine.score
    initialComment.value = mine.comment ?? ''
    prefilled = true
  }
}

async function loadRatings(targetPage = page.value) {
  ratingsLoading.value = true
  try {
    const data = await listRatings(resourceId.value, { page: targetPage, page_size: PAGE_SIZE })
    ratings.value = data.list
    total.value = data.total
    page.value = data.page
    prefillMyRating()
  } finally {
    ratingsLoading.value = false
  }
}

async function handleSubmit(payload: { score: number; comment: string }) {
  submitting.value = true
  try {
    await upsertRating(resourceId.value, payload)
    ElMessage.success('评价成功')
    await Promise.all([loadResource(), loadRatings()])
    // 提交成功后表单重置（RatingForm 通过 props watch 同步）
    initialScore.value = 0
    initialComment.value = ''
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '提交失败，请重试')
  } finally {
    submitting.value = false
  }
}

function handlePageChange(p: number) {
  loadRatings(p)
}

function openSource() {
  if (resource.value?.source_url) {
    window.open(resource.value.source_url, '_blank')
  }
}

// 收藏是强正反馈信号，会随行为导出供推荐模型训练。乐观更新 + 失败回滚。
async function toggleFavorite() {
  if (favoritePending.value) return
  const next = !favorited.value
  favorited.value = next
  favoritePending.value = true
  try {
    await setFavorite(resourceId.value, next)
    ElMessage.success(next ? '已加入收藏' : '已取消收藏')
  } catch (e) {
    favorited.value = !next
    ElMessage.error(e instanceof Error ? e.message : '操作失败，请重试')
  } finally {
    favoritePending.value = false
  }
}

// metadata 是自由键值表：B 站视频的发布时间存的是 Unix 秒，直接渲染会是一长串数字
function formatMetadataValue(key: string, value: unknown): string {
  if (key === 'pubdate' && typeof value === 'number') {
    return formatUnixDate(value)
  }
  return String(value)
}

// 同一路由记录（resources/:id）下切换 id 时，Vue Router 会复用组件实例而不重新挂载，
// 因此必须手动重置并重拉，否则页面会一直停留在上一个资源的数据上
watch(resourceId, () => {
  coverFailed.value = false
  favorited.value = false
  prefilled = false
  initialScore.value = 0
  initialComment.value = ''
  ratings.value = []
  total.value = 0
  page.value = 1
  comments.value = []
  loadResource()
  loadRatings(1)
})

onMounted(() => {
  loadResource()
  loadRatings(1)
})
</script>
<template>
  <div class="page shell">
    <RouterLink :to="{ name: 'search' }" class="link-quiet inline-flex items-center gap-1.5 text-[12.5px]">
      <AppIcon name="arrow-left" :size="14" />
      返回资源库
    </RouterLink>

    <div v-if="resourceLoading" class="mt-5 grid grid-cols-1 gap-8 lg:grid-cols-3">
      <div class="lg:col-span-2">
        <div class="skeleton aspect-video w-full !rounded-[16px]"></div>
        <div class="skeleton mt-5 h-4 w-24"></div>
        <div class="skeleton mt-3 h-8 w-3/4"></div>
        <div class="skeleton mt-3 h-4 w-1/2"></div>
        <div class="card mt-6 p-5">
          <div class="skeleton h-4 w-20"></div>
          <div class="skeleton mt-3 h-4 w-full"></div>
          <div class="skeleton mt-2 h-4 w-5/6"></div>
        </div>
      </div>
      <div class="card h-64 p-5">
        <div class="skeleton h-8 w-24"></div>
        <div class="skeleton mt-4 h-4 w-full"></div>
        <div class="skeleton mt-2 h-4 w-2/3"></div>
      </div>
    </div>

    <div v-else-if="resourceError" class="state-box mt-6">
      <span class="state-icon !bg-danger-soft !text-danger">
        <AppIcon name="alert" :size="22" />
      </span>
      <p class="state-title">资源加载失败</p>
      <p class="state-copy">{{ resourceError }}</p>
      <RouterLink :to="{ name: 'home' }" class="btn btn-secondary btn-sm mt-2">返回首页</RouterLink>
    </div>

    <template v-else-if="resource">
      <div class="mt-4 grid grid-cols-1 gap-8 lg:grid-cols-[minmax(0,1fr)_340px] lg:gap-10">
        <div class="min-w-0">
          <!-- Hero -->
          <div v-if="resource.cover_url && !coverFailed" class="overflow-hidden rounded-[16px] border border-border bg-sunken">
            <img
              :src="resource.cover_url"
              :alt="resource.title"
              referrerpolicy="no-referrer"
              loading="lazy"
              class="max-h-[420px] w-full object-cover"
              @error="coverFailed = true"
            />
          </div>

          <div class="mt-5 flex flex-wrap items-center gap-2">
            <span
              class="chip"
              :class="resource.type === 'course' ? 'chip-success' : resource.type === 'video' ? 'chip-info' : 'chip-warning'"
            >
              <AppIcon :name="resource.type === 'video' ? 'play' : resource.type === 'course' ? 'book' : 'book-open'" :size="12" />
              {{ resource.type === 'course' ? '课程' : resource.type === 'video' ? '视频' : '文章' }}
            </span>
            <span class="tnum text-[12px] text-ink-muted">更新于 {{ formatDate(resource.updated_at) }}</span>
          </div>

          <h1 class="mt-3 text-[26px] font-extrabold leading-[1.22] tracking-[-0.03em] text-ink sm:text-[34px]">
            {{ resource.title }}
          </h1>

          <div class="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2 text-[12.5px] text-ink-secondary">
            <span v-if="resource.category?.name" class="flex items-center gap-1.5">
              <AppIcon name="folder" :size="14" class="text-ink-muted" />
              {{ resource.category.name }}
            </span>
            <span v-if="resource.author" class="flex items-center gap-1.5">
              <AppIcon name="user" :size="14" class="text-ink-muted" />
              {{ resource.author }}
            </span>
            <span v-if="resource.avg_rating > 0" class="tnum flex items-center gap-1.5 font-semibold text-ink">
              <AppIcon name="star-filled" :size="14" class="text-[#d99a20]" />
              {{ resource.avg_rating.toFixed(1) }}
            </span>
            <span v-else class="text-ink-muted">暂无评分</span>
            <span class="tnum flex items-center gap-1.5">
              <AppIcon name="eye" :size="14" class="text-ink-muted" />
              {{ resource.view_count }} 次浏览
            </span>
            <span v-if="resource.difficulty" class="flex items-center gap-1.5">
              <AppIcon name="layers" :size="14" class="text-ink-muted" />
              {{ difficultyLabel(resource.difficulty) }}
            </span>
            <span v-if="resource.duration_minutes" class="tnum flex items-center gap-1.5">
              <AppIcon name="clock" :size="14" class="text-ink-muted" />
              {{ formatDuration(resource.duration_minutes) }}
            </span>
          </div>

          <div v-if="resource.tags?.length" class="mt-4 flex flex-wrap gap-1.5">
            <span v-for="t in resource.tags" :key="t" class="chip !bg-primary-softer !text-primary-active">#{{ t }}</span>
          </div>

          <div class="mt-6 flex flex-wrap gap-3">
            <button
              v-if="resource.source_url"
              type="button"
              class="btn btn-primary"
              @click="openSource"
            >
              <AppIcon name="external" :size="16" />
              前往原站学习
            </button>
            <el-button
              :type="favorited ? 'warning' : 'default'"
              :loading="favoritePending"
              @click="toggleFavorite"
            >
              {{ favorited ? '已收藏' : '收藏' }}
            </el-button>
          </div>

          <!-- 简介 -->
          <section class="card mt-8 p-5 sm:p-6">
            <div class="flex items-center gap-2.5">
              <span class="flex h-8 w-8 items-center justify-center rounded-[9px] bg-primary-soft text-primary">
                <AppIcon name="book-open" :size="16" />
              </span>
              <h2 class="text-[16px] font-bold text-ink">简介</h2>
            </div>
            <p class="mt-4 whitespace-pre-line text-[14px] leading-[1.8] text-ink-secondary">{{ resource.description }}</p>
          </section>

          <!-- 元信息 -->
          <section v-if="metaEntries.length > 0" class="card mt-4 p-5 sm:p-6">
            <div class="flex items-center gap-2.5">
              <span class="flex h-8 w-8 items-center justify-center rounded-[9px] bg-sunken text-ink-secondary">
                <AppIcon name="info" :size="16" />
              </span>
              <h2 class="text-[16px] font-bold text-ink">元信息</h2>
            </div>
            <dl class="mt-4 grid grid-cols-2 gap-x-5 gap-y-4 text-[13px] sm:grid-cols-3">
              <div v-for="([key, value]) in metaEntries" :key="key" class="min-w-0">
                <dt class="text-[11.5px] font-semibold text-ink-muted">{{ key }}</dt>
                <dd class="mt-0.5 break-words text-ink">{{ formatMetadataValue(key, value) }}</dd>
              </div>
            </dl>
          </section>

          <KeyframeStrip v-if="frames" :frames="frames" />

          <!-- 评价 -->
          <section class="mt-8">
            <div class="flex items-center justify-between gap-3">
              <div class="flex items-center gap-2.5">
                <span class="flex h-8 w-8 items-center justify-center rounded-[9px] bg-primary-soft text-primary">
                  <AppIcon name="star" :size="16" />
                </span>
                <h2 class="text-[18px] font-bold text-ink">用户评价</h2>
                <span class="chip tnum !text-[11px]">{{ total }}</span>
              </div>
            </div>
            <div class="card mt-4 px-5 sm:px-6">
              <RatingList
                :ratings="ratings"
                :total="total"
                :page="page"
                :loading="ratingsLoading"
                @page-change="handlePageChange"
              />
            </div>
          </section>

          <!-- B 站评论 -->
          <section v-if="isBilibiliResource" class="mt-8">
            <div class="flex items-center gap-2.5">
              <span class="flex h-8 w-8 items-center justify-center rounded-[9px] bg-sunken text-ink-secondary">
                <AppIcon name="users" :size="16" />
              </span>
              <h2 class="text-[18px] font-bold text-ink">B 站评论</h2>
            </div>
            <div class="card mt-4 px-5 sm:px-6">
              <CommentList :comments="comments" :loading="commentsLoading" />
            </div>
          </section>
        </div>

        <!-- Rating rail -->
        <aside>
          <div class="card p-5 lg:sticky lg:top-24">
            <div class="flex items-center justify-between gap-3">
              <p class="text-[12.5px] font-semibold text-ink-muted">学习者的评分</p>
              <span class="chip tnum !text-[11px]">{{ total }} 条</span>
            </div>

            <div v-if="total > 0" class="mt-3 flex items-end gap-2">
              <span class="tnum text-[42px] font-extrabold leading-none tracking-[-0.04em] text-ink">
                {{ resource.avg_rating.toFixed(1) }}
              </span>
              <span class="pb-1 text-[12px] text-ink-muted">/ 5</span>
            </div>
            <p v-else class="mt-3 text-[14px] font-semibold text-ink-muted">暂无评分</p>
            <el-rate :model-value="Math.round(resource.avg_rating)" disabled class="mt-2" />

            <div class="mt-5 border-t border-border pt-5">
              <p class="mb-4 text-[13.5px] font-bold text-ink">写下你的评价</p>
              <RatingForm
                :initial-score="initialScore"
                :initial-comment="initialComment"
                :submitting="submitting"
                @submit="handleSubmit"
              />
            </div>
          </div>
        </aside>
      </div>
    </template>
  </div>
</template>
