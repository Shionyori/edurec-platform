# edurec-platform 前端资源详情页设计

> 日期：2026-08-19 | 状态：已确认
> 前置：`2026-08-12-frontend-auth-design.md`（认证页面，已合入 main）
> 关联：`docs/api-design.md`（资源/评分/行为接口规范）、`docs/design.md`（页面设计、技术栈）

## 1. 目标

将路由 `resources/:id` 从占位页替换为真实的资源详情页，打通资源浏览闭环：进入自动上报 `view` 行为、查看资源信息与元数据、评分评论的展示与提交。同时完善 mock 数据生成策略，使演示数据便于测试、查找与再生成。

## 2. 范围

- 资源信息展示：类型徽标、标题、分类、标签、作者、平均分、浏览数、描述、元信息、`source_url` 原站链接
- 评分评论：评分概览（平均分 + 条数）、评价表单、分页列表
- 行为上报：进入页面自动上报 `view`（fire-and-forget，不阻塞渲染）
- Mock 数据：确定性生成器 + 核心实体手写扩充 + handler 评分后重算 avg_rating 对齐后端

**不做**：收藏按钮（`favorite` 行为留给个人中心里程碑）、评分分布图、多级评论、富文本评论。

## 3. 设计决策

| 决策 | 选择 | 理由 |
|---|---|---|
| 项目定位 | 模板/演示项目，UI 做薄 | 不追求生产级复杂度，聚焦可演示、可学习 |
| 布局 | 两栏（内容左 + 评分卡右），移动端折叠单栏 | 参考 Coursera/Bilibili 课堂，贴合设计文档前台布局基调 |
| view_count 语义 | 展示接口返回的 `view_count`，不本地自增 | mock 的 GET 详情已自增，与 view 上报并存即可 |
| view 上报时机 | 页面挂载成功后静默上报 | 失败不打断页面渲染 |
| 我的评分回填 | 从已加载评分页中匹配 `user.id` 预填表单 | 不在当前页则不预填，演示可接受 |
| 评分提交后 | 刷新评分列表 + 重拉资源 | 后端 upsert 重算 avg_rating |
| 评分分页 | page_size 固定 10，el-pagination | 让热门资源（>10 条评分）演示分页 |
| mock 数据策略 | 方案 C 混合：核心实体手写 + 评分/补充资源生成 | 兼顾可读、可生成、测试稳定 |

## 4. 页面结构与组件

```
┌───────────────────────────┬───────────────────────┐
│ [类型] 标题  ★4.5·1024浏览 │  评分概览卡（sticky）  │
│ 分类 · 标签 · 作者 · 更新   │    ★4.5  14条评价     │
├───────────────────────────┤   ─────────────       │
│ 简介                      │   ★★★★☆ 评分          │
│ 描述…                     │   [评论输入框]         │
│                           │   [提交评价]           │
│ 元信息：时长/章节/语言      ├───────────────────────┤
│ [前往原站学习]             │  （移动端折叠到下方）   │
├───────────────────────────┴───────────────────────┤
│ 用户评价（列表 + 分页 page_size=10）               │
└───────────────────────────────────────────────────┘
```

新建文件（遵循"页面按模块"目录约定）：

| 文件 | 职责 |
|---|---|
| `pages/resource/ResourceDetailPage.vue` | 页面编排：加载资源 + 加载评分 + view 上报 + 概览卡 + 错误态（404 / 网络错） |
| `components/resource/RatingForm.vue` | el-rate（必填 1-5）+ 评论输入框，校验后 emit `submit(payload)` |
| `components/resource/RatingList.vue` | 列表项（头像/昵称/星级/评论/时间）+ el-pagination，emit `page-change` |
| `mocks/generator.ts` | 确定性 mock 数据生成器（见第 6 节） |

路由调整：`router/index.ts` 中 `resources/:id` 的 `component` 从 `PlaceholderPage` 改为 `ResourceDetailPage`，`meta.requiresAuth` 不变。

## 5. 交互与数据流

**进入页面（onMounted）**
1. `getResource(id)` → 渲染资源信息（loading / 错误 / 404 → 友好"资源不存在" + 返回链接）
2. `recordBehavior(id, 'view')` → 静默上报，失败仅 console 记录
3. `listRatings(id, { page: 1, page_size: 10 })` → 渲染第一页评分

**提交评分**
1. 校验：score 必填（el-rate 天然 1-5），comment 选填
2. `upsertRating(id, { score, comment })` → 成功 ElMessage.success("评价成功")
3. 刷新评分列表 + 重拉资源（显示重算后的 avg_rating）
4. 表单重置

**我的评分回填**：从已加载评分列表找 `rating.user.id === auth.user?.id`，命中则预填表单并提示"你已评分，提交将更新"。不在当前页不预填。

**分页**：el-pagination，切页时 `listRatings(id, { page, page_size: 10 })`。

## 6. Mock 数据生成器与 handler 对齐

### 6.1 `mocks/generator.ts`

- 固定 seed 的 PRNG（mulberry32(42)），纯函数、确定性，同一 seed 两次生成结果一致
- 顶层 `DATA_CONFIG`：
  - `extraResourceCount: 7`（手写 5 + 生成 7 = 12 个资源）
  - `minRatingsPerResource: 2`、`maxRatingsPerResource: 8`
  - `hotResourceRatingCount: 14`（资源 1 评分堆到 >10 条，分页演示 2 页）
- 模板池：各分类 × 类型真实标题池、中文评论池、评分偏置（4-5 居多、少量 3、极少数 1-2）
- 返回 `{ resources, ratings }`，id 从手写实体后顺延

### 6.2 `mocks/db.ts`

- 核心实体手写、id 稳定（现有测试均通用断言，不破）：
  - 用户 4 个：`admin`（管理员）+ `user` + 新增 2 个普通用户
  - 分类 6 个：保留现有 3 个 + 新增 3 个
  - 精选资源 5 个：保留现有，id 1-5
  - 行为：保留现有几条示例
- 组装：`resources: [...core, ...generated.resources]`、`ratings: generated.ratings`

### 6.3 `mocks/handlers.ts`

`POST /resources/:id/ratings` upsert 后**重算该资源 avg_rating**（后端 `RatingService.updateResourceAverage` 会重算，当前 mock 缺失，属偏差修正）。创建与更新路径都需重算，结果保留一位小数。

## 7. 测试策略

| 测试 | 覆盖 |
|---|---|
| `pages/resource/__tests__/ResourceDetailPage.test.ts` | 渲染资源信息、进入上报 view、404 显示不存在、提交后刷新列表与资源 |
| `components/resource/__tests__/RatingForm.test.ts` | 未选评分不提交、合法提交 emit 载荷 |
| `components/resource/__tests__/RatingList.test.ts` | 渲染条目、切页 emit、空态 |
| `mocks/__tests__/generator.test.ts` | 同 seed 两次生成一致、数量符合配置、id 顺延 |
| `mocks/__tests__/handlers.test.ts` 补测 | 评分 upsert 后 avg_rating 重算 |

## 8. 交付物与验证

- ResourceDetailPage + RatingForm + RatingList + 路由调整
- generator + db.ts 扩充 + handlers 重算对齐
- `pnpm type-check`、`pnpm test`、`pnpm build` 全绿
- `pnpm dev` 手动验证：
  - 访问 `/resources/1` 完整渲染两栏布局
  - 提交评分后 ElMessage 提示、平均分变化、列表刷新
  - 热门资源分页可用（>10 条评分）
  - 刷新页面浏览数增长、行为历史可查到 view 记录
