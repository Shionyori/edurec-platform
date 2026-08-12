import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import LoginPage from '../LoginPage.vue'

const pushMock = vi.fn()
const loginMock = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
  useRoute: () => ({ query: { redirect: '/resources/1' } }),
  RouterLink: { template: '<a><slot /></a>' },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ login: loginMock }),
}))

async function submit(wrapper: ReturnType<typeof mount>) {
  await wrapper.find('input[placeholder="用户名或邮箱"]').setValue('admin')
  await wrapper.find('input[placeholder="密码"]').setValue('admin123')
  await wrapper.find('button').trigger('click')
  await flushPromises()
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('登录成功跳转 redirect', async () => {
    loginMock.mockResolvedValue({ id: 1, username: 'admin' })
    const wrapper = mount(LoginPage, { global: { plugins: [ElementPlus] } })
    await submit(wrapper)
    expect(loginMock).toHaveBeenCalledWith('admin', 'admin123')
    expect(pushMock).toHaveBeenCalledWith('/resources/1')
  })

  it('登录失败弹出错误信息', async () => {
    vi.spyOn(ElMessage, 'error').mockImplementation(() => ({}) as never)
    loginMock.mockRejectedValue(new Error('用户名或密码错误'))
    const wrapper = mount(LoginPage, { global: { plugins: [ElementPlus] } })
    await submit(wrapper)
    expect(ElMessage.error).toHaveBeenCalledWith('用户名或密码错误')
  })

  it('表单为空时不提交', async () => {
    const wrapper = mount(LoginPage, { global: { plugins: [ElementPlus] } })
    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(loginMock).not.toHaveBeenCalled()
  })
})
