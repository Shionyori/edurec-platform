<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import AuthShell from '@/components/auth/AuthShell.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const formRef = ref<{ validate: () => Promise<void> }>()
const loading = ref(false)
const form = reactive({
  username: '',
  password: '',
})

const rules = {
  username: [{ required: true, message: '请输入用户名或邮箱', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function onSubmit() {
  // 轻量前置校验：空值直接返回（el-form 校验在真实浏览器生效，这里作为防御与可测性保证）
  if (!form.username.trim() || !form.password) return
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  loading.value = true
  try {
    await auth.login(form.username, form.password)
    ElMessage.success('登录成功')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : ''
    router.push(redirect || { name: 'home' })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '登录失败，请重试')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthShell>
    <h1 class="text-xl font-bold text-ink">欢迎回来</h1>
    <p class="mt-1 text-sm text-ink-secondary">登录 edurec，继续你的学习之旅</p>

    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" class="mt-6">
      <el-form-item label="用户名 / 邮箱" prop="username">
        <el-input
          v-model="form.username"
          placeholder="用户名或邮箱"
          size="large"
          autocomplete="username"
        />
      </el-form-item>
      <el-form-item label="密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          placeholder="密码"
          size="large"
          show-password
          autocomplete="current-password"
          @keyup.enter="onSubmit"
        />
      </el-form-item>
      <el-button type="primary" size="large" class="mt-2 w-full" :loading="loading" @click="onSubmit">
        登 录
      </el-button>
    </el-form>

    <p class="mt-6 text-center text-sm text-ink-secondary">
      还没有账号？
      <RouterLink :to="{ name: 'register' }" class="text-primary hover:underline">去注册</RouterLink>
    </p>
  </AuthShell>
</template>
