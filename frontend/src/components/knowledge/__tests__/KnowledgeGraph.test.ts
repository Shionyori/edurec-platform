import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { KnowledgeEdge, KnowledgeNode } from '@/types'
import KnowledgeGraph from '../KnowledgeGraph.vue'

const nodes: KnowledgeNode[] = [
  { id: 1, code: 'calculus', name: '高等数学', description: '', level: 1, discipline_code: 'math', resource_count: 10, concept_count: 3 },
  { id: 2, code: 'analysis', name: '数学分析', description: '', level: 2, discipline_code: 'math', resource_count: 5, concept_count: 0 },
]

const edges: KnowledgeEdge[] = [{ from: 1, to: 2 }]

describe('KnowledgeGraph', () => {
  it('渲染节点与边，并在节点上显示资源/概念数', () => {
    const wrapper = mount(KnowledgeGraph, { props: { nodes, edges, currentId: 1 } })

    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.findAll('g[role="button"]')).toHaveLength(2)
    expect(wrapper.text()).toContain('高等数学')
    expect(wrapper.text()).toContain('数学分析')
    expect(wrapper.text()).toContain('资源 10 · 概念 3')
    // 一条先修边
    expect(wrapper.findAll('path[marker-end]')).toHaveLength(1)
  })

  it('当前节点高亮', () => {
    const wrapper = mount(KnowledgeGraph, { props: { nodes, edges, currentId: 2 } })
    const rects = wrapper.findAll('rect')
    expect(rects).toHaveLength(2)
    // 当前节点：深墨蓝底 + 信号朱红描边
    expect(rects[1].attributes('fill')).toBe('#0B1B2B')
    expect(rects[1].attributes('stroke')).toBe('#E4572E')
    // 非当前节点保持白底
    expect(rects[0].attributes('fill')).toBe('#FFFFFF')
  })

  it('点击节点抛出 select 事件', async () => {
    const wrapper = mount(KnowledgeGraph, { props: { nodes, edges } })
    await wrapper.findAll('g[role="button"]')[1].trigger('click')

    const emitted = wrapper.emitted('select')
    expect(emitted).toBeTruthy()
    expect((emitted![0][0] as KnowledgeNode).code).toBe('analysis')
  })

  it('空图不渲染 svg 内容', () => {
    const wrapper = mount(KnowledgeGraph, { props: { nodes: [], edges: [] } })
    expect(wrapper.find('svg').exists()).toBe(false)
  })
})

