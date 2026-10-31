<template>
  <!-- 资源卡片骨架屏：与 ResourceCard 大致同构，避免加载时的布局跳动 -->
  <div class="overflow-hidden rounded-[4px] border border-border bg-surface">
    <el-skeleton animated>
      <template #template>
        <el-skeleton-item variant="image" style="width: 100%; height: 176px" />
        <div class="p-5">
          <div class="flex items-center justify-between">
            <el-skeleton-item variant="text" style="width: 22%" />
            <el-skeleton-item variant="text" style="width: 30%" />
          </div>
          <el-skeleton-item variant="h3" style="width: 78%; margin-top: 14px" />
          <el-skeleton-item variant="text" style="width: 100%; margin-top: 10px" />
          <el-skeleton-item variant="text" style="width: 60%; margin-top: 8px" />
          <div class="mt-4 flex items-center justify-between border-t border-border pt-3">
            <el-skeleton-item variant="text" style="width: 18%" />
            <el-skeleton-item variant="text" style="width: 26%" />
          </div>
        </div>
      </template>
    </el-skeleton>
  </div>
</template>
