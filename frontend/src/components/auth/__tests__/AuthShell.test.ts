import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AuthShell from '../AuthShell.vue'

vi.mock('vue-router', () => ({
  RouterLink: { template: '<a><slot /></a>' },
}))

describe('AuthShell', () => {
  it('渲染品牌标题与插槽内容', () => {
    const wrapper = mount(AuthShell, {
      slots: { default: '<p class="test-slot">卡片内容</p>' },
    })
    expect(wrapper.text()).toContain('edurec')
    expect(wrapper.find('.test-slot').text()).toBe('卡片内容')
  })
})
