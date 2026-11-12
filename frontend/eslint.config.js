import pluginVue from 'eslint-plugin-vue'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import skipFormatting from '@vue/eslint-config-prettier/skip-formatting'

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,tsx,vue}'],
  },
  {
    name: 'app/files-to-ignore',
    ignores: ['**/dist/**', '**/coverage/**', '**/public/mockServiceWorker.js'],
  },
  pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,
  {
    name: 'app/page-index-names',
    files: ['src/pages/**/index.vue'],
    rules: {
      // 路由页面统一用 index.vue，文件名即路由；不强制多词组件名
      'vue/multi-word-component-names': 'off',
    },
  },
  skipFormatting,
)
