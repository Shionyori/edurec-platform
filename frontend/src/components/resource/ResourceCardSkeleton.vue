<template>
  <!-- 资源卡片骨架屏：与 ResourceCard 大致同构，避免加载时的布局跳动 -->
  <div class="overflow-hidden rounded-xl border border-border bg-surface shadow-sm">
    <el-skeleton animated>
      <template #template>
        <el-skeleton-item variant="image" style="width: 100%; height: 176px" />
        <div class="p-5">
          <el-skeleton-item variant="text" style="width: 40%" />
          <el-skeleton-item variant="h3" style="width: 75%; margin-top: 12px" />
          <el-skeleton-item variant="text" style="width: 100%; margin-top: 10px" />
          <el-skeleton-item variant="text" style="width: 60%; margin-top: 10px" />
        </div>
      </template>
    </el-skeleton>
  </div>
</template>
