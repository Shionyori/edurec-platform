import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { formatDate } from '@/utils/format'
import type { Rating } from '@/types'
import RatingList from '../RatingList.vue'

const ratings: Rating[] = [
  {
    id: 1,
    user: { id: 2, username: 'user', display_name: '张三', avatar_url: null },
    score: 5,
    comment: '讲解清晰，收获很大',
    // 无时区后缀的本地时间字符串，使断言与运行环境时区无关
    created_at: '2026-07-20T10:00:00',
  },
]

function mountList(props: Record<string, unknown>) {
  return mount(RatingList, { props, global: { plugins: [ElementPlus] } })
}

describe('RatingList', () => {
  it('渲染评价条目（昵称 / 评分 / 评论 / 时间）', () => {
    const wrapper = mountList({ ratings, total: 1, page: 1 })
    expect(wrapper.text()).toContain('张三')
    expect(wrapper.text()).toContain('讲解清晰，收获很大')
    expect(wrapper.text()).toContain('2026-07-20 10:00')
    expect(wrapper.findComponent({ name: 'ElRate' }).props('modelValue')).toBe(5)
  })

  it('空数据展示占位文案', () => {
    const wrapper = mountList({ ratings: [], total: 0, page: 1 })
    expect(wrapper.text()).toContain('暂无评价')
  })

  it('total > 10 时切页 emit page-change', async () => {
    const wrapper = mountList({ ratings, total: 15, page: 1 })
    await wrapper.findComponent({ name: 'ElPagination' }).vm.$emit('current-change', 2)
    expect(wrapper.emitted('page-change')?.[0]?.[0]).toBe(2)
  })

  it('total <= 10 时不显示分页器', () => {
    const wrapper = mountList({ ratings, total: 5, page: 1 })
    expect(wrapper.findComponent({ name: 'ElPagination' }).exists()).toBe(false)
  })

  it('formatDate 格式化本地时间，非法输入原样返回', () => {
    expect(formatDate('2026-07-20T10:00:00')).toBe('2026-07-20 10:00')
    expect(formatDate('not-a-date')).toBe('not-a-date')
  })
})
