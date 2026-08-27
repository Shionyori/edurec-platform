<script setup lang="ts">
import { ref } from 'vue'
import { createResource } from '@/api/resource'
import type { Category, ResourceType } from '@/types'

defineProps<{ categories: Category[] }>()
const emit = defineEmits<{ (e: 'created', resource: { id: number }): void }>()

const submitting = ref(false)
const error = ref('')

const title = ref('')
const description = ref('')
const type = ref<ResourceType | ''>('')
const categoryId = ref<number | undefined>(undefined)
const tags = ref('')
const author = ref('')
const sourceUrl = ref('')
const coverUrl = ref('')

async function submit() {
  if (!title.value.trim()) {
    error.value = '请填写标题'
    return
  }
  if (!description.value.trim()) {
    error.value = '请填写描述'
    return
  }
  if (!type.value) {
    error.value = '请选择类型'
    return
  }
  if (!categoryId.value) {
    error.value = '请选择分类'
    return
  }

  submitting.value = true
  error.value = ''
  try {
    const resource = await createResource({
      title: title.value.trim(),
      description: description.value.trim(),
      type: type.value,
      category_id: categoryId.value,
      tags: tags.value
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean),
      author: author.value.trim() || undefined,
      source_url: sourceUrl.value.trim() || undefined,
      cover_url: coverUrl.value.trim() || undefined,
    })
    emit('created', { id: resource.id })
  } catch (e) {
    error.value = e instanceof Error ? e.message : '创建失败'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="mt-6 rounded-lg border border-border bg-surface p-6">
    <div class="grid grid-cols-1 gap-5">
      <div>
        <label class="mb-1 block text-xs text-ink-muted">标题 *</label>
        <el-input v-model="title" placeholder="请填写标题" maxlength="256" />
      </div>
      <div>
        <label class="mb-1 block text-xs text-ink-muted">描述 *</label>
        <el-input v-model="description" type="textarea" :rows="4" placeholder="请填写描述" />
      </div>
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <div>
          <label class="mb-1 block text-xs text-ink-muted">类型 *</label>
          <el-select v-model="type" placeholder="选择类型" class="w-full">
            <el-option label="课程" value="course" />
            <el-option label="文章" value="article" />
            <el-option label="视频" value="video" />
          </el-select>
        </div>
        <div>
          <label class="mb-1 block text-xs text-ink-muted">分类 *</label>
          <el-select v-model="categoryId" placeholder="选择分类" class="w-full">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </div>
      </div>
      <div>
        <label class="mb-1 block text-xs text-ink-muted">标签（逗号分隔）</label>
        <el-input v-model="tags" placeholder="例如：Python, 数据分析" />
      </div>
      <div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
        <div>
          <label class="mb-1 block text-xs text-ink-muted">作者</label>
          <el-input v-model="author" placeholder="选填" maxlength="128" />
        </div>
        <div>
          <label class="mb-1 block text-xs text-ink-muted">封面 URL</label>
          <el-input v-model="coverUrl" placeholder="选填" maxlength="512" />
        </div>
      </div>
      <div>
        <label class="mb-1 block text-xs text-ink-muted">源地址</label>
        <el-input v-model="sourceUrl" placeholder="选填" maxlength="512" />
      </div>
    </div>

    <p v-if="error" class="mt-4 text-sm text-red-500">{{ error }}</p>
    <div class="mt-6 flex gap-2">
      <el-button type="primary" :loading="submitting" @click="submit">提交</el-button>
    </div>
  </div>
</template>
