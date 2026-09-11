import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { formatUnixDate } from '@/utils/format'
import type { BilibiliComment } from '@/types'
import CommentList from '../CommentList.vue'

const comments: BilibiliComment[] = [
  {
    id: 1,
    author_name: '小明',
    content: '讲得太清楚了，点赞',
    like_count: 120,
    floor: 1,
    published_at: 1759990000,
  },
  {
    id: 2,
    author_name: '阿强',
    content: '跟着学完了，收获很大',
    like_count: 88,
    floor: 2,
    published_at: 1759991000,
  },
]

function mountList(props: { comments: BilibiliComment[]; loading?: boolean }) {
  return mount(CommentList, { props, global: { plugins: [ElementPlus] } })
}

describe('CommentList', () => {
  it('加载中展示抓取提示', () => {
    const wrapper = mountList({ comments: [], loading: true })
    expect(wrapper.text()).toContain('正在抓取 B 站评论')
  })

  it('空数据展示占位文案', () => {
    const wrapper = mountList({ comments: [] })
    expect(wrapper.text()).toContain('评论暂不可用或暂无评论')
  })

  it('渲染评论（昵称 / 内容 / 楼层 / 点赞 / 时间）', () => {
    const wrapper = mountList({ comments })
    const text = wrapper.text()
    expect(text).toContain('小明')
    expect(text).toContain('讲得太清楚了，点赞')
    expect(text).toContain('#1 楼')
    expect(text).toContain('👍 120')
    // 时间断言用同源格式化结果，避免运行环境时区差异
    expect(text).toContain(formatUnixDate(comments[0].published_at))
  })

  it('渲染全部评论条目', () => {
    const wrapper = mountList({ comments })
    expect(wrapper.findAll('li')).toHaveLength(comments.length)
  })

  it('作者名为空时头像回退为问号', () => {
    const wrapper = mountList({
      comments: [{ ...comments[0], id: 9, author_name: '' }],
    })
    expect(wrapper.find('.el-avatar').text()).toBe('?')
  })
})
