<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listCategories } from '@/api/category'
import ResourceUploadForm from '@/components/resource/ResourceUploadForm.vue'
import AppIcon from '@/components/ui/AppIcon.vue'
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
  <div class="page shell">
    <div class="mx-auto max-w-[760px]">
      <RouterLink :to="{ name: 'admin-resources' }" class="link-quiet inline-flex items-center gap-1.5 text-[12.5px]">
        <AppIcon name="arrow-left" :size="14" />
        返回资源管理
      </RouterLink>
      <header class="mt-4">
        <h1 class="headline">上传资源</h1>
        <p class="lede mt-2.5">创建新的教育资源。标题与描述会直接出现在资源卡片与详情页。</p>
      </header>

      <div v-if="loading" class="mt-6 flex items-center gap-2 text-[13px] text-ink-muted">
        <span class="spinner-dots" aria-hidden="true"><span></span><span></span><span></span></span>
        加载分类…
      </div>
      <ResourceUploadForm v-else :categories="categories" @created="handleCreated" />
    </div>
  </div>
</template>
