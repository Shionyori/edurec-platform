import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['src/test/setup.ts'],
    // 默认按 CPU 核数并发会同时拉起数十个 jsdom 实例，内存争用导致偶发 5s 超时；
    // 限制到 4 个 worker 后全量运行稳定在 12s 左右（远快于默认并发的 20-70s）。
    maxWorkers: 4,
  },
})
