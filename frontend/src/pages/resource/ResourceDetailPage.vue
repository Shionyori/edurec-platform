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
// 长合集会被归入 course（见 docs/bilibili-import.md），若绑在 type 上这些课程就没了评论区。
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
  <div class="mx-auto max-w-7xl px-4 py-8 sm:px-6">
    <div v-if="resourceLoading" class="grid grid-cols-1 gap-8 lg:grid-cols-3">
      <div class="lg:col-span-2">
        <el-skeleton animated>
          <template #template>
            <el-skeleton-item variant="image" style="width: 100%; height: 320px; border-radius: 4px" />
            <el-skeleton-item variant="h1" style="width: 60%; margin-top: 20px" />
            <el-skeleton-item variant="text" style="width: 100%; margin-top: 14px" />
            <el-skeleton-item variant="text" style="width: 92%; margin-top: 8px" />
            <el-skeleton-item variant="text" style="width: 84%; margin-top: 8px" />
          </template>
        </el-skeleton>
      </div>
      <div>
        <el-skeleton :rows="5" animated />
      </div>
    </div>
    <div v-else-if="resourceError" class="py-24 text-center">
      <p class="text-sm text-danger">{{ resourceError }}</p>
      <RouterLink :to="{ name: 'home' }">
        <el-button class="mt-4">返回首页</el-button>
      </RouterLink>
    </div>

    <template v-else-if="resource">
      <div class="grid grid-cols-1 gap-8 lg:grid-cols-3">
        <!-- 主内容 -->
        <div class="lg:col-span-2">
          <img
            v-if="resource.cover_url && !coverFailed"
            :src="resource.cover_url"
            :alt="resource.title"
            referrerpolicy="no-referrer"
            loading="lazy"
            class="mb-5 max-h-96 w-full rounded-[4px] border border-border object-cover"
            @error="coverFailed = true"
          />

          <div class="flex flex-wrap items-center gap-2">
            <span
              class="line-chip"
              :style="{ '--chip': resource.type === 'course' ? 'var(--color-line-teal)' : resource.type === 'video' ? 'var(--color-line-blue)' : 'var(--color-line-amber)' }"
            >{{ resource.type === 'course' ? '课程' : resource.type === 'video' ? '视频' : '文章' }}</span>
            <span class="data-num text-xs text-ink-muted">更新于 {{ formatDate(resource.updated_at) }}</span>
          </div>
          <h1 class="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">{{ resource.title }}</h1>
          <div class="data-num mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-ink-secondary">
            <span>{{ resource.category?.name }}</span>
            <span v-for="t in resource.tags" :key="t" class="text-primary">#{{ t }}</span>
            <span>作者：{{ resource.author }}</span>
            <span v-if="resource.avg_rating > 0" class="flex items-center gap-1 text-ink">
              <svg viewBox="0 0 24 24" class="h-3.5 w-3.5" fill="currentColor" aria-hidden="true">
                <path d="M12 3.5l2.6 5.3 5.9.9-4.3 4.1 1 5.8-5.2-2.7-5.2 2.7 1-5.8L3.5 9.7l5.9-.9L12 3.5z" />
              </svg>
              {{ resource.avg_rating.toFixed(1) }}
            </span>
            <span v-else>暂无评分</span>
            <span>{{ resource.view_count }} 次浏览</span>
            <span v-if="resource.difficulty">{{ difficultyLabel(resource.difficulty) }}</span>
            <span v-if="resource.duration_minutes">{{ formatDuration(resource.duration_minutes) }}</span>
          </div>

          <div class="mt-6 rounded-[4px] border border-border bg-surface p-5">
            <div class="flex items-center gap-3 border-b-2 pb-2" style="border-color: var(--color-line-teal)">
              <span class="route-node" style="--node: var(--color-line-teal)" aria-hidden="true"></span>
              <h2 class="text-base font-bold text-ink">简介</h2>
            </div>
            <p class="mt-2 whitespace-pre-line text-sm leading-relaxed text-ink-secondary">{{ resource.description }}</p>
          </div>

          <div v-if="metaEntries.length > 0" class="mt-4 rounded-[4px] border border-border bg-surface p-5">
            <div class="flex items-center gap-3 border-b-2 pb-2" style="border-color: var(--color-line-amber)">
              <span class="route-node" style="--node: var(--color-line-amber)" aria-hidden="true"></span>
              <h2 class="text-base font-bold text-ink">元信息</h2>
            </div>
            <dl class="mt-2 grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
              <div v-for="([key, value]) in metaEntries" :key="key">
                <dt class="text-ink-muted">{{ key }}</dt>
                <dd class="text-ink">{{ formatMetadataValue(key, value) }}</dd>
              </div>
            </dl>
          </div>

          <KeyframeStrip v-if="frames" :frames="frames" />

          <div class="mt-4 flex flex-wrap gap-3">
            <el-button v-if="resource.source_url" type="primary" @click="openSource">前往原站学习</el-button>
            <el-button
              :type="favorited ? 'warning' : 'default'"
              :loading="favoritePending"
              @click="toggleFavorite"
            >
              {{ favorited ? '已收藏' : '收藏' }}
            </el-button>
          </div>

          <div class="mt-8">
            <div class="flex items-center gap-3 border-b-2 pb-2" style="border-color: var(--color-line-blue)">
              <span class="route-node" style="--node: var(--color-line-blue)" aria-hidden="true"></span>
              <h2 class="text-base font-bold text-ink">用户评价（{{ total }}）</h2>
            </div>
            <RatingList
              :ratings="ratings"
              :total="total"
              :page="page"
              :loading="ratingsLoading"
              @page-change="handlePageChange"
            />
          </div>

          <div v-if="isBilibiliResource" class="mt-8">
            <div class="flex items-center gap-3 border-b-2 pb-2" style="border-color: var(--color-line-rose)">
              <span class="route-node" style="--node: var(--color-line-rose)" aria-hidden="true"></span>
              <h2 class="text-base font-bold text-ink">B 站评论</h2>
            </div>
            <CommentList :comments="comments" :loading="commentsLoading" />
          </div>
        </div>

        <!-- 右侧评分卡 -->
        <aside>
          <div class="sticky top-20 overflow-hidden rounded-[4px] border border-border bg-surface p-5">
            <span class="absolute inset-x-0 top-0 h-[2px]" style="background: var(--color-primary)" aria-hidden="true"></span>
            <div class="flex items-end justify-between gap-3">
              <div v-if="total > 0" class="data-num text-3xl font-bold text-ink">{{ resource.avg_rating.toFixed(1) }}</div>
              <div v-else class="text-sm font-medium text-ink-muted">暂无评分</div>
              <div class="data-num text-sm text-ink-muted">{{ total }} 条评价</div>
            </div>
            <el-rate :model-value="Math.round(resource.avg_rating)" disabled class="mt-2" />
            <div class="mt-4 border-t border-border pt-4">
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
