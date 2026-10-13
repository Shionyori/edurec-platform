import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import * as routerMock from 'vue-router'
import type { RecommendationResult, Resource } from '@/types'
import { getRecommendations } from '@/api/recommendation'
import { recordImpressions } from '@/api/impression'
import HomePage from '../index.vue'

// route 必须是**响应式**的：组件的 goal 是 computed(() => route.query.goal)，
// 若用普通对象，改写 query 不会触发重新求值，「切回推荐流」这条路径就测不到。
// 同时让 mock 的 push 真的改写 route——否则点「换一个目标」时 goal 不会变空。
vi.mock('vue-router', async () => {
  const { reactive } = await import('vue')
  const route = reactive({ query: {} as Record<string, string> })
  const push = vi.fn((to: { query?: Record<string, string> }) => {
    route.query = to?.query ?? {}
  })
  return {
    RouterLink: { template: '<a><slot /></a>' },
    useRoute: () => route,
    useRouter: () => ({ push }),
    __route: route,
    __push: push,
  }
})

vi.mock('@/api/recommendation', () => ({ getRecommendations: vi.fn() }))
vi.mock('@/api/impression', () => ({ recordImpressions: vi.fn(() => Promise.resolve(null)) }))
vi.mock('@/api/category', () => ({ listCategories: vi.fn(() => Promise.resolve([])) }))
vi.mock('@/api/learningPlan', () => ({ getLearningPlan: vi.fn(() => Promise.resolve({ goal: '', matched: false, groups: [] })) }))
vi.mock('@/api/behavior', () => ({ recordBehavior: vi.fn(() => Promise.resolve(null)) }))

const authMock = vi.hoisted(() => ({ user: null as unknown }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authMock }))

const routerInternals = routerMock as unknown as {
  __route: { query: Record<string, string> }
  __push: ReturnType<typeof vi.fn>
}
// 通过 setQuery 驱动路由变化（等价于用户导航/点击「换一个目标」）
const setQuery = (query: Record<string, string>) => {
  routerInternals.__route.query = query
}
const pushMock = routerInternals.__push

const mockedGetRecommendations = vi.mocked(getRecommendations)
const mockedRecordImpressions = vi.mocked(recordImpressions)

const resource: Resource = {
  id: 1, title: '机器学习入门', description: '面向零基础学习者',
  cover_url: null, type: 'course',
  category: { id: 1, name: '人工智能' },
  tags: ['AI'], metadata: {},
  author: '吴恩达', source_url: null,
  avg_rating: 4.5, view_count: 1024,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-15T10:00:00Z',
}

const recResult: RecommendationResult = { list: [resource], updated_at: '2026-08-01T00:00:00Z' }

async function mountPage() {
  const wrapper = mount(HomePage, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

// route 变响应式后，前一个用例残留的组件实例不会随测试结束消失：它的 watch 仍然
// 活着，会被后续 setQuery 唤醒并真的发出请求，污染下一个用例的调用次数断言。
// 因此每个用例结束后必须卸载挂载过的组件。
enableAutoUnmount(afterEach)

describe('HomePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setQuery({})
    authMock.user = { interests: [1] } // 默认已有兴趣 → 不展示引导
    mockedGetRecommendations.mockResolvedValue(recResult)
  })

  it('调用推荐接口并渲染推荐列表', async () => {
    const wrapper = await mountPage()
    expect(mockedGetRecommendations).toHaveBeenCalledWith(12)
    expect(wrapper.text()).toContain('机器学习入门')
  })

  it('推荐接口失败时展示错误态', async () => {
    mockedGetRecommendations.mockRejectedValue(new Error('加载失败'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('加载失败')
  })

  it('推荐加载成功后上报本屏曝光', async () => {
    await mountPage()
    expect(mockedRecordImpressions).toHaveBeenCalledWith('home', [1])
  })

  it('新用户（无兴趣）展示冷启动兴趣引导', async () => {
    authMock.user = { interests: [] }
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('选择你感兴趣的方向')
  })

  it('已有兴趣的用户不展示兴趣引导', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).not.toContain('选择你感兴趣的方向')
  })

  it('始终展示提问框', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('想学点什么？')
  })

  it('提交目标后跳转到带 ?goal= 的首页', async () => {
    const wrapper = await mountPage()
    await wrapper.find('input').setValue('机器学习')
    await wrapper.find('form').trigger('submit')

    expect(pushMock).toHaveBeenCalledWith({ name: 'home', query: { goal: '机器学习' } })
  })

  it('URL 带 goal 时进入学习计划态，不拉推荐流', async () => {
    setQuery({ goal: '机器学习' })
    const wrapper = await mountPage()
    expect(mockedGetRecommendations).not.toHaveBeenCalled()
    // 断言计划视图本身渲染了（mock 的 getLearningPlan 返回 matched:false），
    // 而不是只断言页面上有个静态标题——后者与 LearningPlanView 是否工作无关。
    expect(wrapper.text()).toContain('没有找到对应知识点')
  })

  // 回归：带 ?goal= 进入（或刷新时正处于计划态）后再「换一个目标」，
  // 推荐流必须被补拉，否则会停在「暂无推荐内容」且不再发请求。
  it('从计划态切回推荐流时会补拉推荐', async () => {
    setQuery({ goal: '机器学习' })
    const wrapper = await mountPage()
    expect(mockedGetRecommendations).not.toHaveBeenCalled()

    // 点「换一个目标」→ 组件 router.push({ name: 'home' }) → mock 把 query 清空。
    // 注意不能取第一个 button：提问框的「开始规划」也是一个 button，且在 DOM 中更靠前。
    const resetButton = wrapper.findAll('button').find((b) => b.text().includes('换一个目标'))
    expect(resetButton).toBeTruthy()
    await resetButton!.trigger('click')
    await flushPromises()

    // 恰好补拉一次：既证明切回推荐流会请求，也排除重复请求
    expect(mockedGetRecommendations).toHaveBeenCalledTimes(1)
    expect(mockedGetRecommendations).toHaveBeenCalledWith(12)
  })
})
