import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import FrontLayout from '../FrontLayout.vue'

// 顶栏搜索框回归测试。
// 原先这个 el-input 没有 v-model：输入的内容会被渲染清掉（用户描述为"什么也输入不进去"），
// 回车也只 router.push({name:'search'}) 丢掉关键词。这里锁住"可输入 + 带 keyword 跳转"。

// 搜索页在真实使用时由我们自己的测试覆盖；这里只需要一个能承接跳转的占位组件
const SearchStub = { name: 'SearchStub', template: '<div>搜索页占位</div>' }
// FrontLayout 内含 <RouterView />，用空组件替掉，避免渲染子路由
const RouterViewStub = { name: 'RouterViewStub', template: '<div />' }

async function mountLayout() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'home', component: { template: '<div>首页</div>' } },
      { path: '/login', name: 'login', component: { template: '<div>登录</div>' } },
      { path: '/user/me', name: 'user-me', component: { template: '<div>个人中心</div>' } },
      { path: '/search', name: 'search', component: SearchStub },
    ],
  })
  await router.push('/')
  await router.isReady()

  const wrapper = mount(FrontLayout, {
    global: {
      plugins: [ElementPlus, createPinia(), router],
      stubs: { RouterView: RouterViewStub },
    },
  })
  await flushPromises()
  return { wrapper, router }
}

const headerInput = (wrapper: ReturnType<typeof mount>) =>
  wrapper.find('input[placeholder="搜索课程、文章、视频…"]')

describe('FrontLayout 顶栏搜索框', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('可以输入文字（v-model 已绑定，输入不会被清掉）', async () => {
    const { wrapper } = await mountLayout()

    const input = headerInput(wrapper)
    expect(input.exists()).toBe(true)

    await input.setValue('数据结构')

    // 旧实现没有 v-model，这里拿到的会是空字符串
    expect((input.element as HTMLInputElement).value).toBe('数据结构')
  })

  it('回车跳转到搜索页并带上 keyword', async () => {
    const { wrapper, router } = await mountLayout()

    await headerInput(wrapper).setValue('数据结构')
    await headerInput(wrapper).trigger('keyup.enter')
    await flushPromises()

    expect(router.currentRoute.value.name).toBe('search')
    expect(router.currentRoute.value.query.keyword).toBe('数据结构')
  })

  it('空白关键词回车不跳转', async () => {
    const { wrapper, router } = await mountLayout()

    await headerInput(wrapper).setValue('   ')
    await headerInput(wrapper).trigger('keyup.enter')
    await flushPromises()

    expect(router.currentRoute.value.name).toBe('home')
  })

  it('关键词前后空格会被裁剪后再跳转', async () => {
    const { wrapper, router } = await mountLayout()

    await headerInput(wrapper).setValue('  雅思  ')
    await headerInput(wrapper).trigger('keyup.enter')
    await flushPromises()

    expect(router.currentRoute.value.query.keyword).toBe('雅思')
  })
})
