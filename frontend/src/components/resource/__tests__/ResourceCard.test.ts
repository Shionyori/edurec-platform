import { describe, expect, it } from 'vitest'
import { RouterLinkStub, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { Resource } from '@/types'
import ResourceCard from '../ResourceCard.vue'

const resource: Resource = {
  id: 1, title: '机器学习入门', description: '面向零基础学习者',
  cover_url: null, type: 'course',
  category: { id: 1, name: '人工智能' },
  tags: ['AI'], metadata: {},
  author: '吴恩达', source_url: null,
  avg_rating: 4.5, view_count: 1024,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-15T10:00:00Z',
}

function mountCard() {
  return mount(ResourceCard, {
    props: { resource },
    global: {
      plugins: [ElementPlus],
      stubs: { RouterLink: RouterLinkStub },
    },
  })
}

describe('ResourceCard', () => {
  it('渲染资源核心信息', () => {
    const wrapper = mountCard()
    expect(wrapper.text()).toContain('机器学习入门')
    expect(wrapper.text()).toContain('面向零基础学习者')
    expect(wrapper.text()).toContain('人工智能')
    expect(wrapper.text()).toContain('4.5')
    expect(wrapper.text()).toContain('1024 次浏览')
  })

  it('作为链接跳转资源详情', () => {
    const wrapper = mountCard()
    const link = wrapper.findComponent(RouterLinkStub)
    expect(link.exists()).toBe(true)
    expect(link.props('to')).toEqual({ name: 'resource-detail', params: { id: 1 } })
  })
})
