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
  <div class="card overflow-hidden">
    <div class="relative h-24 bg-surface-2 sm:h-28">
      <svg class="absolute inset-0 h-full w-full text-ink/[0.06]" viewBox="0 0 800 140" preserveAspectRatio="none" fill="none" aria-hidden="true">
        <path d="M0 112C160 72 300 40 470 52s250 52 330 34" stroke="currentColor" stroke-width="1.5" />
        <path d="M0 128C180 92 320 66 500 74s230 38 300 26" stroke="currentColor" stroke-width="1.5" stroke-dasharray="3 7" />
        <circle cx="700" cy="48" r="6" fill="var(--color-primary)" opacity="0.5" />
      </svg>
    </div>

    <div class="relative z-10 px-5 pb-5 sm:px-6 sm:pb-6">
      <div class="-mt-9 flex flex-wrap items-end justify-between gap-4">
        <div class="flex items-end gap-4">
          <el-avatar :size="72" :src="user.avatar_url ?? undefined" class="!border-4 !border-surface !bg-ink">
            {{ (user.display_name ?? user.username).charAt(0) }}
          </el-avatar>
          <div class="pb-1">
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="truncate text-[22px] font-extrabold tracking-[-0.03em] text-ink">
                {{ user.display_name ?? user.username }}
              </h2>
              <el-tag v-if="user.is_admin" type="warning" size="small">管理员</el-tag>
            </div>
            <p class="mt-1 truncate text-[13px] text-ink-secondary">
              @{{ user.username }} · {{ user.email }}
            </p>
          </div>
        </div>
        <button v-if="!editing" type="button" class="btn btn-secondary btn-sm" @click="editing = true">
          <AppIcon name="edit" :size="14" />
          编辑资料
        </button>
      </div>

      <p class="tnum mt-3 flex items-center gap-1.5 text-[11.5px] text-ink-muted">
        <AppIcon name="calendar" :size="13" />
        注册于 {{ formatDate(user.created_at) }}
      </p>

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
  </div>
</template>
