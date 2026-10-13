import { describe, expect, it, vi } from 'vitest'
import { RouterLinkStub, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { LearningPlanGroup, Resource } from '@/types'
import KnowledgePointPanel from '../KnowledgePointPanel.vue'

vi.mock('@/api/behavior', () => ({ recordBehavior: vi.fn(() => Promise.resolve(null)) }))

const resource: Resource = {
  id: 1, title: '机器学习入门', description: '面向零基础学习者',
  cover_url: null, type: 'course',
  category: { id: 1, name: '人工智能' },
  tags: ['机器学习'], metadata: {},
  author: '吴恩达', source_url: null,
  avg_rating: 4.5, view_count: 1024,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-15T10:00:00Z',
}

function group(overrides: Partial<LearningPlanGroup> = {}): LearningPlanGroup {
  return {
    knowledge_point: { id: 1, name: '监督学习', description: '', category: null },
    total: 1,
    resources: [resource],
    ...overrides,
  }
}

// el-collapse-item 必须挂在 el-collapse 里才有正确的注入上下文。
// 注意 props 的写法：组件定义里的 props 是「声明」，不是「值」——
// 值要从 mount 的第二个参数传进去，否则 :group 拿到 undefined。
const PanelHarness = {
  components: { KnowledgePointPanel },
  props: ['g'],
  template: '<el-collapse><KnowledgePointPanel :group="g" /></el-collapse>',
}

function mountPanel(g: LearningPlanGroup) {
  return mount(PanelHarness, {
    props: { g },
    global: { plugins: [ElementPlus], stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('KnowledgePointPanel', () => {
  it('渲染知识点名与资源总数', () => {
    const wrapper = mountPanel(group({ total: 12 }))
    expect(wrapper.text()).toContain('监督学习')
    expect(wrapper.text()).toContain('12 个资源')
  })

  it('渲染组内资源卡片', () => {
    const wrapper = mountPanel(group())
    expect(wrapper.text()).toContain('机器学习入门')
  })

  it('total 大于已展示数量时给出「查看全部」入口', () => {
    const wrapper = mountPanel(group({ total: 12 }))
    const link = wrapper.findAllComponents(RouterLinkStub).find((l) => l.text().includes('查看全部'))
    expect(link).toBeTruthy()
    expect(link!.props('to')).toEqual({ name: 'search', query: { keyword: '监督学习' } })
  })

  it('total 未超过已展示数量时不出现「查看全部」', () => {
    const wrapper = mountPanel(group({ total: 1 }))
    expect(wrapper.text()).not.toContain('查看全部')
  })

  it('兜底分组的占位知识点也能正常渲染', () => {
    const wrapper = mountPanel(
      group({
        knowledge_point: { id: 0, name: '为你找到的相关资源', description: '', category: null },
      }),
    )
    expect(wrapper.text()).toContain('为你找到的相关资源')
  })
})
