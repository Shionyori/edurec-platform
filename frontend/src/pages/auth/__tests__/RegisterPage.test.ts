import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import { register } from '@/api/auth'
import RegisterPage from '../RegisterPage.vue'

const pushMock = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: pushMock }),
  RouterLink: { template: '<a><slot /></a>' },
}))

vi.mock('@/api/auth', () => ({ register: vi.fn() }))

const mockedRegister = vi.mocked(register)

async function fillAndSubmit(wrapper: ReturnType<typeof mount>) {
  await wrapper.find('input[placeholder="用户名"]').setValue('newuser')
  await wrapper.find('input[placeholder="邮箱"]').setValue('newuser@example.com')
  await wrapper.find('input[placeholder="密码"]').setValue('secret123')
  await wrapper.find('input[placeholder="再次输入密码"]').setValue('secret123')
  await wrapper.find('button').trigger('click')
  await flushPromises()
}

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('注册成功跳转登录页并提示', async () => {
    vi.spyOn(ElMessage, 'success').mockImplementation(() => ({}) as never)
    mockedRegister.mockResolvedValue({
      user: {
        id: 3,
        username: 'newuser',
        email: 'newuser@example.com',
        display_name: null,
        avatar_url: null,
        created_at: '2026-08-12T00:00:00Z',
      },
    })
    const wrapper = mount(RegisterPage, { global: { plugins: [ElementPlus] } })
    await fillAndSubmit(wrapper)
    expect(mockedRegister).toHaveBeenCalledWith(
      expect.objectContaining({
        username: 'newuser',
        email: 'newuser@example.com',
        password: 'secret123',
      }),
    )
    expect(ElMessage.success).toHaveBeenCalledWith('注册成功，请登录')
    expect(pushMock).toHaveBeenCalledWith({ name: 'login' })
  })

  it('两次密码不一致时不提交', async () => {
    const wrapper = mount(RegisterPage, { global: { plugins: [ElementPlus] } })
    await wrapper.find('input[placeholder="用户名"]').setValue('newuser')
    await wrapper.find('input[placeholder="邮箱"]').setValue('newuser@example.com')
    await wrapper.find('input[placeholder="密码"]').setValue('secret123')
    await wrapper.find('input[placeholder="再次输入密码"]').setValue('different')
    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(mockedRegister).not.toHaveBeenCalled()
  })
})
