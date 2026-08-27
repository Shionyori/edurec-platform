<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listCategories } from '@/api/category'
import ResourceUploadForm from '@/components/resource/ResourceUploadForm.vue'
import type { Category } from '@/types'

const router = useRouter()
const categories = ref<Category[]>([])
const loading = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    categories.value = await listCategories()
  } finally {
    loading.value = false
  }
})

function handleCreated({ id }: { id: number }) {
  router.push({ name: 'resource-detail', params: { id } })
}
</script>

<template>
  <div class="mx-auto max-w-2xl px-6 py-8">
    <h1 class="text-2xl font-bold">上传资源</h1>
    <p class="mt-2 text-sm text-ink-secondary">创建新的教育资源</p>

    <div v-if="loading" class="py-16 text-center text-sm text-ink-muted">加载分类…</div>
    <ResourceUploadForm v-else :categories="categories" @created="handleCreated" />
  </div>
</template>
