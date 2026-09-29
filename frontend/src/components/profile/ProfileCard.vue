<script setup lang="ts">
import { ref } from 'vue'
import { updateMe } from '@/api/user'
import { formatDate } from '@/utils/format'
import type { User } from '@/types'

const props = defineProps<{ user: User }>()
const emit = defineEmits<{ (e: 'updated', user: Partial<User>): void }>()

const editing = ref(false)
const saving = ref(false)
const error = ref('')
const displayName = ref(props.user.display_name ?? '')
const avatarUrl = ref(props.user.avatar_url ?? '')

async function save() {
  saving.value = true
  error.value = ''
  try {
    const updated = await updateMe({
      display_name: displayName.value.trim() || undefined,
      avatar_url: avatarUrl.value.trim() || undefined,
    })
    emit('updated', updated)
    editing.value = false
  } catch (e) {
    error.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    saving.value = false
  }
}

function cancel() {
  editing.value = false
  displayName.value = props.user.display_name ?? ''
  avatarUrl.value = props.user.avatar_url ?? ''
  error.value = ''
}
</script>

<template>
  <div class="rounded-xl border border-border bg-surface p-6 shadow-sm">
    <div class="flex items-start gap-4">
      <el-avatar :size="64" :src="user.avatar_url ?? undefined">
        {{ (user.display_name ?? user.username).charAt(0) }}
      </el-avatar>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <h2 class="truncate text-xl font-semibold text-ink">{{ user.display_name ?? user.username }}</h2>
          <el-tag v-if="user.is_admin" type="warning" size="small">管理员</el-tag>
        </div>
        <p class="mt-1 truncate text-sm text-ink-secondary">@{{ user.username }} · {{ user.email }}</p>
        <p class="mt-0.5 text-xs text-ink-muted">注册于 {{ formatDate(user.created_at) }}</p>
      </div>
    </div>

    <div v-if="!editing" class="mt-5">
      <el-button round @click="editing = true">编辑资料</el-button>
    </div>

    <div v-else class="mt-6 border-t border-border pt-5">
      <div class="grid grid-cols-1 gap-4">
        <div>
          <label class="mb-1 block text-xs text-ink-muted">昵称</label>
          <el-input v-model="displayName" placeholder="输入昵称" maxlength="32" />
        </div>
        <div>
          <label class="mb-1 block text-xs text-ink-muted">头像 URL</label>
          <el-input v-model="avatarUrl" placeholder="https://…" maxlength="512" />
        </div>
      </div>
      <div v-if="error" class="mt-3 text-sm text-red-500">{{ error }}</div>
      <div class="mt-4 flex gap-2">
        <el-button type="primary" round :loading="saving" @click="save">保存</el-button>
        <el-button round @click="cancel">取消</el-button>
      </div>
    </div>
  </div>
</template>
