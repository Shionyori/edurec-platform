<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { getRecommendations } from '@/api/recommendation'
import { recordImpressions } from '@/api/impression'
import { useAuthStore } from '@/stores/auth'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import ResourceCardSkeleton from '@/components/resource/ResourceCardSkeleton.vue'
import InterestPicker from '@/components/profile/InterestPicker.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
import type { Resource } from '@/types'

const auth = useAuthStore()
const resources = ref<Resource[]>([])
const loading = ref(false)
const error = ref('')
const heroKeyword = ref('')

// 无兴趣且无历史的新用户，展示冷启动引导
const showInterestPicker = computed(
  () => !!auth.user && (auth.user.interests?.length ?? 0) === 0,
)

const nextUp = computed(() => resources.value[0] ?? null)

const nextUpType = computed(() => {
  switch (nextUp.value?.type) {
    case 'course':
      return { label: '课程', color: 'var(--color-line-teal)' }
    case 'video':
      return { label: '视频', color: 'var(--color-line-blue)' }
    default:
      return { label: '文章', color: 'var(--color-line-amber)' }
  }
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await getRecommendations(12)
    resources.value = data.list
    // 列表渲染即曝光：记录本屏展示的资源（CTR 的分母）。失败静默，埋点不影响页面。
    recordImpressions('home', data.list.map((r) => r.id)).catch(() => {})
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="overflow-x-clip pb-20 md:pb-28">
    <!-- Hero: the product's own question, then the actual next step -->
    <section class="shell pt-10 md:pt-16 lg:pt-20">
      <div class="grid items-center gap-12 lg:grid-cols-[1.02fr_0.98fr] lg:gap-16">
        <div class="max-w-xl">
          <h1 class="display">
            下一站，<span class="text-primary">学什么？</span>
          </h1>
          <p class="lede mt-5 max-w-[34rem]">
            从知识点与先修关系出发，跳过已经掌握的，只推荐你当前该学的一节——并告诉你为什么是它。
          </p>

          <form action="/search" method="get" class="mt-7 flex flex-col gap-2.5 sm:flex-row">
            <label class="relative flex-1">
              <span class="sr-only">搜索课程、文章或视频</span>
              <AppIcon
                name="search"
                :size="18"
                class="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-ink-muted"
              />
              <input
                v-model="heroKeyword"
                name="keyword"
                type="search"
                placeholder="搜索课程、文章或视频"
                class="h-[50px] w-full rounded-[12px] border border-border-strong bg-surface pl-11 pr-4 text-[14px] text-ink outline-none transition-[box-shadow,border-color] duration-150 placeholder:text-ink-muted focus:border-primary focus:shadow-[0_0_0_3px_rgba(201,69,31,0.14)]"
              />
            </label>
            <button type="submit" class="btn btn-primary btn-lg shrink-0">
              开始搜索
              <AppIcon name="arrow-right" :size="16" />
            </button>
          </form>

          <div class="mt-6 flex flex-wrap items-center gap-x-5 gap-y-2 text-[12.5px] text-ink-muted">
            <span class="flex items-center gap-1.5">
              <AppIcon name="layers" :size="14" />
              21 个学科 · 72 个知识点
            </span>
            <span class="flex items-center gap-1.5">
              <AppIcon name="route" :size="14" />
              先修关系驱动
            </span>
            <span class="flex items-center gap-1.5">
              <AppIcon name="spark" :size="14" class="text-primary" />
              每次只推荐一节
            </span>
          </div>

          <div class="mt-8 flex flex-wrap items-center gap-3">
            <RouterLink :to="{ name: 'knowledge' }" class="btn btn-secondary">
              <AppIcon name="compass" :size="16" />
              浏览知识点
            </RouterLink>
            <RouterLink :to="{ name: 'questionnaire' }" class="link-arrow">
              调整我的学习问卷
              <AppIcon name="arrow-right" :size="14" />
            </RouterLink>
          </div>
        </div>

        <!-- The live recommendation, not an illustration -->
        <div class="relative mx-auto w-full max-w-[480px] lg:mx-0">
          <svg
            class="pointer-events-none absolute -right-2 -top-10 h-[250px] w-[250px] text-ink/10 sm:-right-10 sm:-top-16 sm:h-[330px] sm:w-[330px]"
            viewBox="0 0 330 330"
            fill="none"
            aria-hidden="true"
          >
            <circle cx="165" cy="165" r="120" stroke="currentColor" stroke-width="1" />
            <circle cx="165" cy="165" r="152" stroke="currentColor" stroke-width="1" stroke-dasharray="2 7" />
            <path d="M165 13v304M13 165h304" stroke="currentColor" stroke-width="1" stroke-dasharray="2 9" />
            <circle cx="285" cy="165" r="5" fill="var(--color-primary)" />
          </svg>

          <div class="relative">
            <div class="card relative overflow-hidden shadow-[var(--shadow-lift)]">
              <div class="flex items-center justify-between border-b border-border px-5 py-3.5">
                <span class="flex items-center gap-2 text-[12.5px] font-semibold text-ink-secondary">
                  <span class="flex h-5 w-5 items-center justify-center rounded-md bg-primary-soft text-primary">
                    <AppIcon name="spark-filled" :size="12" />
                  </span>
                  下一步
                </span>
                <span class="chip chip-primary !text-[11px]">为你推荐</span>
              </div>

              <div v-if="loading" class="p-5">
                <div class="skeleton h-36 w-full !rounded-[10px]"></div>
                <div class="skeleton mt-4 h-4 w-3/4"></div>
                <div class="skeleton mt-2 h-4 w-1/2"></div>
                <div class="skeleton mt-4 h-10 w-full"></div>
              </div>

              <template v-else-if="nextUp">
                <div class="p-5">
                  <div class="flex items-center gap-2.5">
                    <span class="chip !text-[11px]">
                      <span class="dot" :style="{ color: nextUpType.color }" aria-hidden="true"></span>
                      {{ nextUpType.label }}
                    </span>
                    <span class="truncate text-[12px] text-ink-muted">{{ nextUp.category?.name || '未分类' }}</span>
                  </div>

                  <RouterLink
                    :to="{ name: 'resource-detail', params: { id: nextUp.id } }"
                    class="group mt-3 block"
                  >
                    <div class="flex gap-4">
                      <div class="relative h-[76px] w-[112px] shrink-0 overflow-hidden rounded-[10px] bg-sunken">
                        <img
                          v-if="nextUp.cover_url"
                          :src="nextUp.cover_url"
                          :alt="nextUp.title"
                          referrerpolicy="no-referrer"
                          loading="lazy"
                          class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.04]"
                        />
                        <span
                          v-else
                          class="flex h-full w-full items-center justify-center text-[28px] font-extrabold tracking-[-0.05em] text-ink/15"
                          aria-hidden="true"
                        >{{ nextUp.title.charAt(0) }}</span>
                      </div>
                      <div class="min-w-0">
                        <h2 class="line-clamp-2 text-[16px] font-bold leading-snug tracking-[-0.02em] text-ink transition-colors group-hover:text-primary">
                          {{ nextUp.title }}
                        </h2>
                        <p class="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-[12px] text-ink-muted">
                          <span v-if="nextUp.avg_rating > 0" class="tnum flex items-center gap-1">
                            <AppIcon name="star-filled" :size="11" class="text-[#d99a20]" />
                            {{ nextUp.avg_rating.toFixed(1) }}
                          </span>
                          <span class="tnum">{{ nextUp.view_count }} 次浏览</span>
                          <span v-if="nextUp.author" class="truncate">{{ nextUp.author }}</span>
                        </p>
                      </div>
                    </div>

                    <p class="mt-4 flex items-start gap-2 rounded-[10px] bg-primary-softer px-3 py-2.5 text-[12.5px] leading-relaxed text-primary-active">
                      <AppIcon name="spark" :size="14" class="mt-0.5 shrink-0" />
                      <span>{{ nextUp.reason || '按先修关系与你的学习记录排序，这一节是当下的最佳起点。' }}</span>
                    </p>
                  </RouterLink>
                </div>

                <RouterLink
                  :to="{ name: 'resource-detail', params: { id: nextUp.id } }"
                  class="flex items-center justify-between border-t border-border bg-surface-2 px-5 py-3.5 text-[13px] font-semibold text-ink transition-colors hover:bg-sunken"
                >
                  去看这一节
                  <AppIcon name="arrow-up-right" :size="15" class="text-ink-muted" />
                </RouterLink>
              </template>

              <div v-else class="flex flex-col items-center gap-2 px-5 py-12 text-center">
                <span class="state-icon !mb-1 !h-11 !w-11">
                  <AppIcon name="compass" :size="20" />
                </span>
                <p class="state-title">还没有可推荐的内容</p>
                <p class="state-copy">先按关键词采集一些资源，或直接浏览知识点。</p>
                <RouterLink :to="{ name: 'search' }" class="btn btn-secondary btn-sm mt-2">去搜索资源</RouterLink>
              </div>
            </div>

            <!-- The method behind the recommendation, in plain words -->
            <div class="card mt-3 flex items-center justify-between gap-3 px-5 py-3.5">
              <div
                v-for="(step, index) in [
                  { label: '先修检查', icon: 'route' },
                  { label: '掌握度匹配', icon: 'target' },
                  { label: '逐节解锁', icon: 'check-circle' },
                ]"
                :key="step.label"
                class="flex min-w-0 items-center gap-2"
              >
                <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-[8px] bg-sunken text-ink-secondary">
                  <AppIcon :name="step.icon" :size="14" />
                </span>
                <span class="truncate text-[11.5px] font-semibold text-ink-secondary">{{ step.label }}</span>
                <span v-if="index < 2" class="hidden h-px w-4 shrink-0 bg-border-strong sm:block" aria-hidden="true"></span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <div v-if="showInterestPicker" class="shell mt-14 md:mt-20">
      <InterestPicker @saved="load" />
    </div>

    <!-- Recommendations -->
    <section class="shell mt-14 md:mt-20">
      <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h2 class="section-title">为你挑选</h2>
          <p class="mt-1.5 text-[13px] text-ink-muted">
            按先修关系与掌握度排序<template v-if="resources.length"> · 共 {{ resources.length }} 条</template>
          </p>
        </div>
        <RouterLink :to="{ name: 'search' }" class="link-arrow">
          查看全部资源
          <AppIcon name="arrow-right" :size="14" />
        </RouterLink>
      </div>

      <div v-if="loading" class="mt-6 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
        <ResourceCardSkeleton v-for="i in 6" :key="i" />
      </div>

      <div v-else-if="error" class="state-box mt-6">
        <span class="state-icon !bg-danger-soft !text-danger">
          <AppIcon name="alert" :size="22" />
        </span>
        <p class="state-title">推荐暂时加载失败</p>
        <p class="state-copy">{{ error }}</p>
        <button type="button" class="btn btn-secondary btn-sm mt-2" @click="load">重新加载</button>
      </div>

      <div v-else-if="resources.length === 0" class="state-box mt-6">
        <span class="state-icon">
          <AppIcon name="compass" :size="22" />
        </span>
        <p class="state-title">还没有可推荐的内容</p>
        <p class="state-copy">先到搜索页按关键词采集资源，或浏览知识点导航。</p>
        <div class="mt-2 flex flex-wrap justify-center gap-2">
          <RouterLink :to="{ name: 'search' }" class="btn btn-primary btn-sm">去搜索资源</RouterLink>
          <RouterLink :to="{ name: 'knowledge' }" class="btn btn-secondary btn-sm">浏览知识点</RouterLink>
        </div>
      </div>

      <div v-else class="mt-6 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
        <ResourceCard
          v-for="(r, i) in resources"
          :key="r.id"
          :resource="r"
          show-reason-fallback
          class="fade-up"
          :style="{ animationDelay: `${Math.min(i, 8) * 35}ms` }"
        />
      </div>
    </section>

    <!-- How the path works: one panel, a real sequence -->
    <section class="shell mt-16 md:mt-24">
      <div class="panel-quiet overflow-hidden">
        <div class="grid gap-8 p-6 sm:p-8 md:grid-cols-3 md:gap-10 md:p-10">
          <div class="md:col-span-3 md:max-w-xl">
            <h2 class="section-title">一条路径，三步走完</h2>
            <p class="mt-2 text-[13.5px] leading-relaxed text-ink-secondary">
              不是把资源堆在你面前，而是按先修关系排好顺序；学完一节，才解锁下一节。
            </p>
          </div>

          <div class="flex gap-4 md:block">
            <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-surface text-ink shadow-[0_1px_2px_rgba(11,27,43,0.06)]">
              <AppIcon name="target" :size="17" />
            </span>
            <div class="md:mt-4">
              <p class="text-[13.5px] font-bold text-ink">选一个目标</p>
              <p class="mt-1 text-[12.5px] leading-relaxed text-ink-muted">从学科或知识点进入，告诉系统你想学什么。</p>
            </div>
          </div>

          <div class="flex gap-4 md:block">
            <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-surface text-ink shadow-[0_1px_2px_rgba(11,27,43,0.06)]">
              <AppIcon name="route" :size="17" />
            </span>
            <div class="md:mt-4">
              <p class="text-[13.5px] font-bold text-ink">生成学习路径</p>
              <p class="mt-1 text-[12.5px] leading-relaxed text-ink-muted">按先修关系排序，跳过你已经掌握的内容。</p>
            </div>
          </div>

          <div class="flex gap-4 md:block">
            <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-surface text-ink shadow-[0_1px_2px_rgba(11,27,43,0.06)]">
              <AppIcon name="check-circle" :size="17" />
            </span>
            <div class="md:mt-4">
              <p class="text-[13.5px] font-bold text-ink">完成本节</p>
              <p class="mt-1 text-[12.5px] leading-relaxed text-ink-muted">只推荐当前这一节，完成后自动解锁下一步。</p>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
