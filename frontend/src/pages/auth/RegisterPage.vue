<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { register } from '@/api/auth'
import AuthShell from '@/components/auth/AuthShell.vue'

const router = useRouter()

const formRef = ref<{ validate: () => Promise<void> }>()
const loading = ref(false)
const form = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  display_name: '',
})

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 64, message: '用户名长度 3-64 个字符', trigger: 'blur' },
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 128, message: '密码长度 6-128 个字符', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    {
      validator: (_rule: unknown, value: string, callback: (err?: Error) => void) => {
        if (value !== form.password) callback(new Error('两次输入的密码不一致'))
        else callback()
      },
      trigger: 'blur',
    },
  ],
}

async function onSubmit() {
  // 轻量前置校验：必填 + 密码一致性（el-form 校验在真实浏览器生效，这里作为防御与可测性保证）
  if (
    !form.username.trim() ||
    !form.email.trim() ||
    !form.password ||
    !form.confirmPassword ||
    form.password !== form.confirmPassword
  ) {
    return
  }
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  loading.value = true
  try {
    await register({
      username: form.username,
      email: form.email,
      password: form.password,
      display_name: form.display_name || undefined,
    })
    ElMessage.success('注册成功，请登录')
    router.push({ name: 'login' })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '注册失败，请重试')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthShell>
    <h1 class="text-xl font-bold text-ink">创建账号</h1>
    <p class="mt-1 text-sm text-ink-secondary">加入 edurec，发现适合你的教育资源</p>

    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="mt-6">
      <el-form-item label="用户名" prop="username">
        <el-input v-model="form.username" placeholder="用户名" size="large" autocomplete="username" />
      </el-form-item>
      <el-form-item label="邮箱" prop="email">
        <el-input v-model="form.email" placeholder="邮箱" size="large" autocomplete="email" />
      </el-form-item>
      <el-form-item label="密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          placeholder="密码"
          size="large"
          show-password
          autocomplete="new-password"
        />
      </el-form-item>
      <el-form-item label="确认密码" prop="confirmPassword">
        <el-input
          v-model="form.confirmPassword"
          type="password"
          placeholder="再次输入密码"
          size="large"
          show-password
          autocomplete="new-password"
        />
      </el-form-item>
      <el-form-item label="昵称（选填）" prop="display_name">
        <el-input v-model="form.display_name" placeholder="昵称" size="large" />
      </el-form-item>
      <el-button type="primary" size="large" class="mt-2 w-full" :loading="loading" @click="onSubmit">
        注 册
      </el-button>
    </el-form>

    <p class="mt-6 text-center text-sm text-ink-secondary">
      已有账号？
      <RouterLink :to="{ name: 'login' }" class="text-primary hover:underline">去登录</RouterLink>
    </p>
  </AuthShell>
</template>
