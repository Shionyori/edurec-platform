<script setup lang="ts">
import { ref } from 'vue'
import { updateMe } from '@/api/user'
import { formatDate } from '@/utils/format'
import AppIcon from '@/components/ui/AppIcon.vue'
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
  <div class="card p-5 sm:p-6">
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div class="flex min-w-0 items-center gap-4">
        <el-avatar
          :size="72"
          :src="user.avatar_url ?? undefined"
          class="shrink-0 !border-4 !border-surface !bg-ink"
        >
          {{ (user.display_name ?? user.username).charAt(0) }}
        </el-avatar>
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <h2 class="truncate text-[22px] font-extrabold tracking-[-0.03em] text-ink">
              {{ user.display_name ?? user.username }}
            </h2>
            <el-tag v-if="user.is_admin" type="warning" size="small">管理员</el-tag>
          </div>
          <p class="mt-1 truncate text-[13px] text-ink-secondary">
            @{{ user.username }} · {{ user.email }}
          </p>
          <p class="tnum mt-1.5 flex items-center gap-1.5 text-[11.5px] text-ink-muted">
            <AppIcon name="calendar" :size="12" />
            注册于 {{ formatDate(user.created_at) }}
          </p>
        </div>
      </div>
      <button v-if="!editing" type="button" class="btn btn-secondary btn-sm w-full shrink-0 justify-center sm:w-auto" @click="editing = true">
        <AppIcon name="edit" :size="14" />
        编辑资料
      </button>
    </div>

    <div v-if="editing" class="mt-5 border-t border-border pt-5">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="field-label">昵称</label>
          <el-input v-model="displayName" aria-label="昵称" placeholder="输入昵称" maxlength="32" />
        </div>
        <div>
          <label class="field-label">头像 URL</label>
          <el-input v-model="avatarUrl" aria-label="头像 URL" placeholder="https://…" maxlength="512" />
        </div>
      </div>
      <div v-if="error" class="mt-3 flex items-center gap-1.5 text-[12.5px] text-danger">
        <AppIcon name="alert" :size="14" />
        {{ error }}
      </div>
      <div class="mt-4 flex gap-2">
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
        <el-button @click="cancel">取消</el-button>
      </div>
    </div>
  </div>
</template>
