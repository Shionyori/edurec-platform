import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { DisciplineDetail } from '@/types'
import { getDiscipline } from '@/api/discipline'
import DisciplineDetailPage from '../DetailPage.vue'

// 用真实 router（而非 RouterLinkStub）验证「点击知识点卡片确实会跳转」
vi.mock('@/api/discipline', () => ({
  getDiscipline: vi.fn(),
  startDisciplineCrawl: vi.fn(),
  getDisciplineCrawlStatus: vi.fn(),
}))

const detail: DisciplineDetail = {
  discipline: { code: 'math', name: '数学', description: '', point_count: 1, resource_count: 1 },
  points: [
    {
      id: 1,
      code: 'calculus',
      name: '高等数学',
      description: '',
      level: 1,
      discipline_code: 'math',
      resource_count: 1,
      concept_count: 0,
    },
  ],
  resources: [],
  resource_total: 1,
}

describe('DisciplineDetailPage 知识点卡片导航', () => {
  it('点击知识点卡片跳转到知识点详情', async () => {
    vi.mocked(getDiscipline).mockResolvedValue(detail)

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/knowledge', name: 'knowledge', component: { template: '<div />' } },
        { path: '/discipline/:code', component: DisciplineDetailPage },
        {
          path: '/knowledge/:code',
          name: 'knowledge-detail',
          component: { template: '<div />' },
        },
      ],
    })
    await router.push('/discipline/math')
    await router.isReady()

    const wrapper = mount(DisciplineDetailPage, {
      global: { plugins: [ElementPlus, router] },
    })
    await flushPromises()

    const card = wrapper.findAll('a').find((link) => link.text().includes('高等数学'))
    expect(card).toBeTruthy()

    await card!.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.name).toBe('knowledge-detail')
    expect(router.currentRoute.value.params.code).toBe('calculus')
  })
})
