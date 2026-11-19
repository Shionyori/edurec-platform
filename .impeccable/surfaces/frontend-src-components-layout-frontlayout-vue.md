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

THESIS: 把“下一步学什么”变成书桌上一张轻松可拿起的推荐卡：薄荷纸、森林墨、一支玉青笔。拒绝旧的车站牌语言（深蓝横条、2px 线路、站点圆点、全大写微标签、4px 直角），也拒绝上一版的奶油纸 + 赤陶红；这一版要清新、轻、不用力。

OWN-WORLD: 薄荷纸底 `#F4F7F5` 承担阅读，白纸卡片承担交互，森林墨 `#17231F` 用于文字与品牌标记；玉青 `#0F7A66` 只用于行动与当前状态。方向性标题面板改为浅玉色 `#F2F9F7`，不再使用深色横条。Manrope 自托管用于拉丁字母与数字，中文回退系统黑体；14px 卡片、10px 控件、16px 面板，pill 只给 chip 与小控件。层级色玉青/蓝/琥珀/玫红仅在层级与资源类型标记、知识图谱中承担语义。每页留出大段安静留白，用 1px 发丝线而不是 2px 轨道组织信息。

STORY: 用户先看到“下一步”这一节是什么、为什么是它；进入学科看到按层级分组的知识点；知识点详情用先修图解释顺序；学习路径用逐节解锁的时间线把“已完成 / 当前学习 / 未解锁”写清楚。移动端用底部四个目的地取代挤压的横向导航；登录页用浅玉色 44/56 分栏，而不是 1:1 深色海报。

FIRST VIEWPORT: 白色粘性顶栏（spark 品牌、首页/发现/知识点、搜索、账户）；首页以“下一站，学什么？”为唯一 h1，右侧是真实的“下一步”推荐卡，卡片右缘与内容右缘对齐并与左侧文案垂直居中；搜索、浏览知识点、调整问卷是首屏动作。

FORM: Operate mode, fresh mint-and-jade direction; code-led implementation in `frontend/src/styles/index.css` and `frontend/src/components/ui/AppIcon.vue`; no image assets required.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
