import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { Resource } from '@/types'
import { listFavorites, setFavorite } from '@/api/behavior'
import FavoriteList from '../FavoriteList.vue'

vi.mock('@/api/behavior', () => ({ listFavorites: vi.fn(), setFavorite: vi.fn() }))

const mockedListFavorites = vi.mocked(listFavorites)
const mockedSetFavorite = vi.mocked(setFavorite)

const resource: Resource = {
  id: 1, title: '机器学习入门', description: 'x',
  cover_url: 'https://x/cover.jpg', type: 'video',
  category: { id: 1, name: '人工智能' },
  tags: [], metadata: {}, author: null, source_url: null,
  avg_rating: 4.5, view_count: 10,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-01T08:00:00Z',
}

async function mountList() {
  const wrapper = mount(FavoriteList, {
    global: { plugins: [ElementPlus], stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  return wrapper
}

describe('FavoriteList', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedListFavorites.mockResolvedValue({ list: [resource], total: 1, page: 1, page_size: 12 })
    mockedSetFavorite.mockResolvedValue({ favorited: false })
  })

  it('以紧凑条目渲染收藏（小缩略图，而非大封面卡）', async () => {
    const wrapper = await mountList()
    expect(wrapper.text()).toContain('机器学习入门')
    expect(wrapper.text()).toContain('人工智能')
    const img = wrapper.find('img')
    expect(img.exists()).toBe(true)
    expect(img.classes()).toContain('h-12') // 缩略图高 48px，远小于原大封面卡
  })

  it('取消收藏调用接口并移除该条目', async () => {
    const wrapper = await mountList()
    const btn = wrapper.findAll('button').find((b) => b.text().includes('取消收藏'))
    expect(btn).toBeTruthy()

    await btn!.trigger('click')
    await flushPromises()

    expect(mockedSetFavorite).toHaveBeenCalledWith(1, false)
    expect(wrapper.text()).not.toContain('机器学习入门')
  })

  it('没有收藏时展示空态', async () => {
    mockedListFavorites.mockResolvedValue({ list: [], total: 0, page: 1, page_size: 12 })
    const wrapper = await mountList()
    expect(wrapper.text()).toContain('还没有收藏')
  })
})
