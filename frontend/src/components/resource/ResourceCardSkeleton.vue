<template>
  <!-- 资源卡片骨架屏：与 ResourceCard 同构，避免加载时的布局跳动 -->
  <div class="card overflow-hidden">
    <div class="skeleton aspect-video w-full !rounded-none"></div>
    <div class="p-4 sm:p-[18px]">
      <div class="flex items-center justify-between">
        <div class="skeleton h-3 w-20"></div>
        <div class="skeleton h-3 w-10"></div>
      </div>
      <div class="skeleton mt-3 h-4 w-[86%]"></div>
      <div class="skeleton mt-2 h-4 w-[62%]"></div>
      <div class="skeleton mt-3 h-3 w-full"></div>
      <div class="skeleton mt-2 h-3 w-[72%]"></div>
      <div class="mt-4 flex items-center justify-between border-t border-border pt-3">
        <div class="skeleton h-3 w-16"></div>
        <div class="skeleton h-3 w-12"></div>
      </div>
    </div>
  </div>
</template>
