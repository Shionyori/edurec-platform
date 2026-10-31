# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

**Assumption (from repository evidence and the current UI; not confirmed in an interview):**
中文用户（以大学生、自学者为主）在桌面浏览器上按自己的节奏寻找课程、视频与文章，也会用移动端浏览器碎片时间学习。核心任务是从“不知道下一步学什么”走到“找到当前该学的一节并学完”。

## Product Purpose

教育资源推荐平台：用知识点与先修关系组织课程资源，通过推荐与学习路径帮助用户发现下一步该学的内容。它同时是 edurec-engine 推荐模型的应用场景与完整前后端实践项目。

## Positioning

不是资源目录的堆积，而是“知识点 → 先修关系 → 学习路径”的推荐：结合掌握度跳过已掌握内容，只推荐当前该学的一节，并让推荐理由可读。

## Operating Context

- 桌面浏览器为主，中文界面；登录后使用。
- 新用户通过兴趣选择与初始学习问卷冷启动。
- 资源来自 B 站采集/导入；无资源学科可自动抓取。
- 本地开发：Vite dev server + Go 后端 + MySQL/Redis。
- 相关文档：`docs/adaptive-learning.md`、`docs/recsys-design.md`、`docs/demo-walkthrough.md`。

## Capabilities and Constraints

确认存在的能力（来自代码与文档）：

- 认证（JWT 登录/注册/刷新）、资源浏览/搜索/筛选/排序、评分、评论、收藏、行为历史。
- 首页推荐流（含推荐理由）、搜索页无限滚动、资源详情。
- 学科 → 知识点 → 学习路径（先修关系、逐节解锁、掌握度）。
- 初始学习问卷、兴趣选择、个人中心、管理后台（用户/资源/分类/推荐）。
- Element Plus + Tailwind CSS v4；现有 token 位于 `frontend/src/styles/index.css`。
- 知识点、概念、资源为空是常态，界面必须给出明确空状态与下一步动作。

未决：是否保留现有视觉识别；没有品牌手册或 Logo 资产。

## Brand Commitments

- 名称：edurec；定位：教育资源推荐平台。
- 现有产品名称与功能文案需保留；除此之外没有已确认的品牌资产。

## Evidence on Hand

- 真实资源数据来自 B 站（标题、封面、作者、评分、浏览数）。
- 21 个学科、72 个知识点、先修边与概念图（`docs/adaptive-learning.md`）。
- 视觉金样/截图基线：无。

## Product Principles

1. 先回答“下一步学什么”，再展示全部内容。
2. 先修关系是可解释的结构，不是装饰；界面要让“为什么推荐这个”可读。
3. 空状态不是失败页，而是下一步动作的入口。
4. 学习进度必须可见、可恢复（已完成/当前/未解锁）。
5. 中文内容优先保证可读性与扫描效率，动效服务于理解。

## Accessibility & Inclusion

- 保留键盘可达与 `prefers-reduced-motion` 支持。
- 正文对比度满足 WCAG AA；状态不能只靠颜色表达（需文字标签）。
