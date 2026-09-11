<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getResource } from '@/api/resource'
import { listRatings, upsertRating } from '@/api/rating'
import { listComments } from '@/api/comment'
import { recordBehavior } from '@/api/behavior'
import { useAuthStore } from '@/stores/auth'
import { formatDate, formatUnixDate } from '@/utils/format'
import type { BilibiliComment, Rating, Resource } from '@/types'
import RatingForm from '@/components/resource/RatingForm.vue'
import RatingList from '@/components/resource/RatingList.vue'
import CommentList from '@/components/resource/CommentList.vue'

const PAGE_SIZE = 10

const route = useRoute()
const auth = useAuthStore()
const resourceId = computed(() => Number(route.params.id))

const resource = ref<Resource | null>(null)
const resourceLoading = ref(true)
const resourceError = ref('')

// 外链封面（如 B 站）可能失效：no-referrer 已规避防盗链，但仍需对加载失败做降级
const coverFailed = ref(false)

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

// 只有 B 站视频才有可爬取的评论
const isBilibiliVideo = computed(
  () => resource.value?.type === 'video' && !!resource.value.source_url?.includes('bilibili.com'),
)

async function loadResource() {
  resourceLoading.value = true
  resourceError.value = ''
  try {
    resource.value = await getResource(resourceId.value)
    // 资源加载成功才上报 view（404 时不产生行为记录）
    recordBehavior(resourceId.value, 'view').catch(() => {})
    if (isBilibiliVideo.value) {
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
  <div class="mx-auto max-w-7xl px-6 py-8">
    <div v-if="resourceLoading" class="py-24 text-center text-sm text-ink-muted">加载中…</div>
    <div v-else-if="resourceError" class="py-24 text-center">
      <p class="text-sm text-red-500">{{ resourceError }}</p>
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
            class="mb-5 max-h-96 w-full rounded-lg border border-border object-cover"
            @error="coverFailed = true"
          />

          <div class="flex items-center gap-2">
            <el-tag
              size="small"
              :type="resource.type === 'course' ? 'primary' : resource.type === 'video' ? 'success' : 'warning'"
            >
              {{ resource.type }}
            </el-tag>
            <span class="text-xs text-ink-muted">更新于 {{ formatDate(resource.updated_at) }}</span>
          </div>
          <h1 class="mt-3 text-2xl font-bold text-ink">{{ resource.title }}</h1>
          <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-ink-secondary">
            <span>{{ resource.category?.name }}</span>
            <span v-for="t in resource.tags" :key="t" class="text-primary">#{{ t }}</span>
            <span>作者：{{ resource.author }}</span>
            <span class="text-amber-500">★ {{ resource.avg_rating.toFixed(1) }}</span>
            <span>{{ resource.view_count }} 次浏览</span>
          </div>

          <div class="mt-6 rounded-lg border border-border bg-surface p-5">
            <h2 class="text-base font-semibold text-ink">简介</h2>
            <p class="mt-2 whitespace-pre-line text-sm leading-relaxed text-ink-secondary">{{ resource.description }}</p>
          </div>

          <div v-if="Object.keys(resource.metadata).length > 0" class="mt-4 rounded-lg border border-border bg-surface p-5">
            <h2 class="text-base font-semibold text-ink">元信息</h2>
            <dl class="mt-2 grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
              <div v-for="(value, key) in resource.metadata" :key="key">
                <dt class="text-ink-muted">{{ key }}</dt>
                <dd class="text-ink">{{ formatMetadataValue(key, value) }}</dd>
              </div>
            </dl>
          </div>

          <div v-if="resource.source_url" class="mt-4">
            <el-button type="primary" round @click="openSource">前往原站学习</el-button>
          </div>

          <div class="mt-8">
            <h2 class="text-base font-semibold text-ink">用户评价（{{ total }}）</h2>
            <RatingList
              :ratings="ratings"
              :total="total"
              :page="page"
              :loading="ratingsLoading"
              @page-change="handlePageChange"
            />
          </div>

          <div v-if="isBilibiliVideo" class="mt-8">
            <h2 class="text-base font-semibold text-ink">B 站评论</h2>
            <CommentList :comments="comments" :loading="commentsLoading" />
          </div>
        </div>

        <!-- 右侧评分卡 -->
        <aside>
          <div class="sticky top-20 rounded-lg border border-border bg-surface p-5">
            <div class="flex items-end justify-between">
              <div class="text-3xl font-bold text-ink">{{ resource.avg_rating.toFixed(1) }}</div>
              <div class="text-sm text-ink-muted">{{ total }} 条评价</div>
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
