---
version: 1
slug: "frontend-src-components-layout-frontlayout-vue"
primary_target: "frontend/src/components/layout/FrontLayout.vue"
related_targets: ["frontend/src/pages/home/index.vue","frontend/src/pages/discipline/index.vue","frontend/src/pages/discipline/DetailPage.vue","frontend/src/pages/knowledge/DetailPage.vue","frontend/src/pages/learning-path/DetailPage.vue"]
---

# Surface: 学习发现与导航

- Scope: `/` 首页推荐、`/knowledge` 学科导航、`/discipline/:code` 学科详情、`/knowledge/:code` 知识点详情、`/learning-path/:id` 学习路径、`/search` 搜索、`/questionnaire` 学习问卷；共享外壳 `FrontLayout` 与 `ResourceCard`。
- Mode: Operate。用户来这里完成任务：判断下一步学什么，并进入内容学完一节。
- Audience / job: 中文大学生与自学者；从“不知道学什么”到“找到当前该学的一节”。
- Action: 浏览推荐 → 选学科 → 看知识点与先修关系 → 生成学习路径 → 完成一节。
- Constraints: 保留现有路由、接口、Element Plus 组件语义、键盘可达与 reduced-motion；不新增虚构数据或商业声明。

## Direction contract

THESIS: 把“下一步学什么”画成一条可读的线路：知识点是先修线路上的站点，当前目标是“你在这里”。拒绝通用 SaaS 卡片瀑布（白底 + 紫色主色 + 等宽卡片）作为身份。

OWN-WORLD: 深墨蓝站牌 `#0B1B2B` 承担导航与标题带，冷白纸面 `#F5F4EF` 承担内容地；信号朱红 `#C9451F` 只用于“当前/行动”；线路色青 `#0E7C7B`、蓝 `#1D4ED8`、琥珀 `#B45309`、玫红 `#BE123C` 按层级与学科分配，承担真实语义。中文用系统黑体，编码/数字用紧凑大写微标签与 tabular-nums；2px 线路、圆点站点、方形端头组成可复用的线路语言。密度偏紧，用线路条组织信息。

STORY: 用户先看到“下一站”是什么；进入学科看到一条条线路；点进知识点看先修换乘图；学习路径是行程单，已完成/当前/未解锁一眼可辨。

FIRST VIEWPORT: 顶部深墨蓝站牌导航（线路牌式品牌、搜索、主导航、用户）；下方首页以“下一站，学什么？”为主标题，右侧是“学习问卷 / 全部资源”动作；推荐资源以“本站推荐”线路标题 + 三列站卡呈现，冷启动兴趣选择紧随其后并保持显著。

FORM: grounded direction #4 of 7（transit route map），code-led；seed:cf400a00f4b999af。

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
