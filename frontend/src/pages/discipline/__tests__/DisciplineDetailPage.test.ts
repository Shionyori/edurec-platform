import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { DisciplineDetail, Resource } from '@/types'
import { getDiscipline, getDisciplineCrawlStatus, startDisciplineCrawl } from '@/api/discipline'
import DisciplineDetailPage from '../DetailPage.vue'

vi.mock('vue-router', async () => {
  const { reactive } = await import('vue')
  const route = reactive({ params: { code: 'chemistry' } })
  return {
    route,
    useRoute: () => route,
    RouterLink: { template: '<a><slot /></a>' },
  }
})
vi.mock('@/api/discipline', () => ({
  getDiscipline: vi.fn(),
  startDisciplineCrawl: vi.fn(),
  getDisciplineCrawlStatus: vi.fn(),
}))

const mockedGet = vi.mocked(getDiscipline)
const mockedStart = vi.mocked(startDisciplineCrawl)
const mockedStatus = vi.mocked(getDisciplineCrawlStatus)

const resource: Resource = {
  id: 700, title: '有机化学基础', description: '', cover_url: null, type: 'course',
  category: { id: 2, name: 'B站视频' }, tags: [], metadata: {}, author: '某老师',
  source_url: 'https://www.bilibili.com/video/BV1xx', avg_rating: 4.6, view_count: 100,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-02T08:00:00Z',
}

const emptyDetail: DisciplineDetail = {
  discipline: { code: 'chemistry', name: '化学', description: '无机、有机', point_count: 4, resource_count: 0 },
  points: [
    { id: 30, code: 'organic-chemistry', name: '有机化学', description: '', level: 1, discipline_code: 'chemistry', resource_count: 0, concept_count: 0 },
  ],
  resources: [],
  resource_total: 0,
}

const filledDetail: DisciplineDetail = {
  ...emptyDetail,
  discipline: { ...emptyDetail.discipline, resource_count: 1 },
  resources: [resource],
  resource_total: 1,
}

async function mountPage() {
  const wrapper = mount(DisciplineDetailPage, {
    global: { plugins: [ElementPlus], stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  return wrapper
}

describe('DisciplineDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedGet.mockResolvedValue(emptyDetail)
    mockedStatus.mockResolvedValue({
      discipline_code: 'chemistry',
      status: 'done',
      imported: 5,
      linked: 3,
      resource_total: 1,
    })
  })

  it('无资源时自动触发 B 站抓取，完成后刷新资源', async () => {
    mockedGet.mockResolvedValueOnce(emptyDetail).mockResolvedValue(filledDetail)
    mockedStart.mockResolvedValue({
      discipline_code: 'chemistry',
      status: 'done',
      imported: 5,
      linked: 3,
      resource_total: 1,
    })

    const wrapper = await mountPage()

    expect(mockedStart).toHaveBeenCalledWith('chemistry')
    expect(mockedGet).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('已自动补充 5 条资源')
    expect(wrapper.text()).toContain('有机化学基础')
  })

  it('后端把空资源返回为 null 时仍能渲染知识点卡片', async () => {
    // 旧版后端无资源时返回 resources: null；前端曾因 detail.resources.length 崩溃
    mockedGet.mockResolvedValue({ ...emptyDetail, resources: null } as unknown as DisciplineDetail)
    mockedStart.mockResolvedValue({
      discipline_code: 'chemistry',
      status: 'done',
      imported: 0,
      linked: 0,
      resource_total: 0,
    })

    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('有机化学')
    expect(wrapper.text()).toContain('该学科暂无资源')
  })

  it('已有资源时不触发抓取', async () => {
    mockedGet.mockResolvedValue(filledDetail)
    const wrapper = await mountPage()

    expect(mockedStart).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('有机化学基础')
    expect(wrapper.text()).toContain('有机化学')
  })

  it('抓取失败时提示且保留知识点', async () => {
    mockedStart.mockRejectedValue(new Error('engine 未启动'))
    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('engine 未启动')
    expect(wrapper.text()).toContain('有机化学')
  })
})
