# edurec-platform 学习目标 · 知识点推荐层设计

> 日期：2026-10-13 | 状态：待评审
> 关联：`docs/recsys-design.md`（§1.3 / §7 决策修订，见第 9 节）、`docs/design.md`（页面设计与决策记录）、`docs/api-design.md`（接口规范）、`docs/engine-integration.md`（离线批量接入形态）
> 需求来源：`temp/requirement.md`

## 1. 目标

在现有「选兴趣方向 → 推荐」链条**前面**加一层入口：用户直接表达**学习目标**，系统给出该目标下的**知识点列表**，每个知识点各自挂一组推荐资源，让平台从"内容站"变成有目的的学习工具。

第一轮交付一条**只读但完整**的链路：提问框 → 知识点面板（每组带真实资源）。

## 2. 范围

**做**
- 后端：知识点表与导入命令（tag 聚合）+ 目标匹配服务 + 新接口 `GET /api/v1/learning-plan`
- 前端：首页提问框 → 知识点面板（可折叠，组内复用 `ResourceCard`）；原资源库（`/search`）保留不动
- 复用现有推荐信号作为组内排序依据；知识点面板资源纳入曝光打点

**不做（本轮明确排除）**
- 编辑/替换知识点下的资源
- 学习空间持久化（保存计划、进度、完成状态）
- LLM 打标、语义/向量匹配、在线实时拆解
- 学习路径排序（知识点之间的先后/依赖关系）
- 知识点面板分页（每组固定上限，不分页）

## 3. 核心机制

### 3.1 一句话

**知识点不是新的推荐算法，而是对现有推荐信号的一次"分桶"。**

```
用户目标 → 匹配知识点 KP[]
        → 每个 kp：用 kp 对应标签过滤资源池
        → 排序：engine 推荐序（命中缓存者优先）> avg_rating > view_count
        → 取 top-M 组成该知识点的面板
```

### 3.2 为什么是"过滤资源池"而不是"和推荐列表求交集"

engine 产出的是每用户 top-N（默认 20）的扁平列表。若拿它与知识点求交集，知识点一多，每组只剩 1-2 个资源，多数知识点直接空掉——这正是需求讨论中识别出的"不太好用"的根因。

改为**用知识点过滤整个资源池**后：每个知识点都有资源；排序信号仍是原有的那套；知识点面板因而天然是"清单"形态。

### 3.3 复用了什么

| 能力 | 位置 | 说明 |
|---|---|---|
| 按标签过滤 + 排序 | `ResourceRepo.List`（`repository/resource.go:59-115`） | 已有 `Tags`（`JSON_CONTAINS`）与 `Sort`，**无需新增查询语句** |
| 个性化排序信号 | `RecommendationService.Get`（`service/recommendation.go:37`） | 取其列表顺序当作组内排序的第一优先级 |
| 资源卡片 | `ResourceCard.vue` | 面板内直接复用 |
| 曝光打点 | `POST /api/v1/impressions` | 新场景 `learning-plan`，与首页 `home` 并存 |

**平台仍然不做任何推理**，`docs/engine-integration.md` 的既定接入形态不变。

## 4. 设计决策

| # | 决策 | 选择 | 理由 |
|---|---|---|---|
| 1 | 知识点来源（本轮） | 从 `resources.tags` 聚合 + 停用词过滤 | 零外部依赖、零 LLM。导入契约固定，将来换 LLM 打标时消费方不变 |
| 2 | 知识点存储 | **单表** `knowledge_points`，不建资源关联表 | 本轮知识点即标签，关联关系由标签隐式承载，服务层用 `List(AnyTags=[规范名+别名])`（OR 语义）直接可查；关联表属第 10 节的演进项 |
| 3 | 目标匹配算法 | 子串匹配（知识点名/别名 → 目标文本） | 中文无需分词，`"我想学机器学习"` 直接包含 `"机器学习"`；确定性、可测 |
| 4 | 匹配不到时 | 兜底为按目标做资源关键词搜索，返回单一分组 | 绝不返回空白页 |
| 5 | 组内排序 | engine 推荐序 > `avg_rating` > `view_count` | 复用既有信号，实现"个性化"而不引入新模型 |
| 6 | 空组处理 | 关联资源为 0 的知识点**从响应中剔除** | 杜绝"空头支票"——点开却没内容比不展示更伤 |
| 7 | 首页初始态 | 保留现有推荐流，提问框置于其上方 | 改动最小；`home` 场景曝光打点不断流（`home` 自身的行不受影响；但新场景会进入全表 CTR 分母，见 §11.3） |
| 8 | 提交目标后 | 整页切换为知识点面板 | 对应需求原文"页面变化为知识点列表的界面" |
| 9 | 路由 | 不新增路由，复用 `/`，状态驱动；`?goal=` 同步到 URL | 刷新/分享可恢复；无需新页面骨架 |
| 10 | 鉴权 | 需要登录 | 与全站一致，且排序信号依赖用户个性化数据 |
| 11 | 组内资源上限 | M = 6；知识点上限 N = 8 | 面板需要克制，超出部分本轮到资源库页自行探索 |

## 5. 数据模型

新增**一张表**（追加到 `database/mysql.go:42` 的 `AutoMigrate` 列表 —— 本项目用 GORM AutoMigrate，无迁移脚本）。

### knowledge_points（知识点）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | INT UNSIGNED AUTO_INCREMENT PK | |
| name | VARCHAR(128) UNIQUE NOT NULL | 知识点名（本轮即标签原文） |
| description | VARCHAR(512) | 知识点说明，可为空 |
| aliases | JSON | 别名数组，如 `["supervised learning","监督式学习"]`；**仅用于目标匹配** |
| category_id | INT UNSIGNED FK → categories.id | 可为空 |
| source | VARCHAR(32) | 来源：`tag` / `seed` / `llm`，用于将来区分产出方 |
| sort_order | INT DEFAULT 0 | 展示与匹配的稳定排序 |
| created_at / updated_at | DATETIME | |

> `name` 与 `resources.tags` 中的字符串**按原值对应**。别名**同时参与**目标匹配与资源检索：
> 服务层以 `AnyTags`（OR）把「规范名 + 别名」一起作为过滤条件，因此只带别名标签的资源也不会丢。

## 6. 接口

### 6.1 生成学习计划

```
GET /api/v1/learning-plan?goal=<学习目标>&knowledge_points=<int, 默认 8>
```

**需要认证：是**

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "goal": "我想入门机器学习",
    "matched": true,
    "groups": [
      {
        "knowledge_point": {
          "id": 3,
          "name": "监督学习",
          "description": "从带标签数据中学习映射关系",
          "category": { "id": 1, "name": "人工智能" }
        },
        "total": 12,
        "resources": [
          {
            "id": 12,
            "title": "机器学习入门：线性回归",
            "cover_url": "https://example.com/c1.jpg",
            "type": "video",
            "category": { "id": 1, "name": "人工智能" },
            "tags": ["监督学习", "回归"],
            "avg_rating": 4.6,
            "view_count": 3182,
            "author": "某UP主",
            "difficulty": "beginner",
            "duration_minutes": 42,
            "created_at": "2026-06-15T08:00:00Z"
          }
        ]
      }
    ]
  }
}
```

| 字段 | 说明 |
|---|---|
| `matched` | `false` 表示未匹配到任何知识点，`groups` 为兜底单一分组 |
| `groups[].total` | 该知识点关联的资源**总数**（不受 M 限制），供 UI 显示"12 个资源"。直接取自 `ResourceRepo.List` 已返回的 `Total`，零额外查询 |
| `groups[].resources` | 上限 M=6；已按决策 #5 排序；顺序即展示顺序 |

**错误**

| 场景 | 状态码 | 说明 |
|---|---|---|
| `goal` 为空或去空白后为空 | 400 | 业务错误码，提示"请输入学习目标" |
| `goal` 长度 > 100 字符 | 400 | 防止恶意长文本 |
| 未登录 | 401 | 统一中间件 |

### 6.2 匹配与排序算法（确定性，可单测）

```
PlanForGoal(userID, goal, nKP, mRes):
  1. goal = trim(goal)；若长度 < 2 → 400
  2. candidates = knowledge_points 全量（量小，直接载入）
  3. matched = { kp | len(kp.name) >= 2 且 (kp.name ⊂ goal 或 ∃a ∈ kp.aliases, a ⊂ goal) }
                ── 拉丁文忽略大小写；匹配词长 < 2 的名字跳过，避免 "C" 命中一切
  4. 若 matched 为空 → matched = { kp | kp.category.name ⊂ goal }
  5. 若仍为空 → 走兜底（第 7 节）
  6. matched 排序：(命中词长度 desc, sort_order asc, id asc)，取前 nKP
  7. recRank = map[resourceID]rank，来自 RecommendationService.Get(userID, 50)
  8. 对每个 kp：
       res = ResourceRepo.List(AnyTags=[kp.Name] + kp.Aliases, PageSize=100, Sort="rating")  # 100 为上限
       若 res.Total == 0 → 剔除该 kp
       排序键：(recRank 存在 ? recRank : +∞) asc, avg_rating desc, view_count desc
       取前 mRes；组内 total 字段直接透传 res.Total
  9. 若最终 groups 为空 → 走兜底
```

## 7. 兜底

未匹配到任何知识点时（或匹配到的知识点全部无资源）：

```
ResourceRepo.List(Keyword=goal, Sort="rating", PageSize=12)
```

返回**单一分组**，`matched=false`，`knowledge_point` 为 null 或占位名"为你找到的相关资源"。

前端据此渲染不同标题与一句说明（"没有找到对应知识点，以下是按目标搜索到的资源"），而不是报错或空白。

## 8. 导入命令与阶段 0 验收

### 8.1 命令

```
go run ./cmd/import_knowledge_points --from-tags            # 从 resources.tags 聚合（本轮实现）
go run ./cmd/import_knowledge_points --from-tags --preview  # 只读库、不写库，打印将要导入的知识点（本轮实现）
# --file data/knowledge_points.json                          # 手工/LLM 产出（本轮不实现，见 §10 演进）
```

沿用 `cmd/import_dataset` 的三步走惯例，但语义有一处不同：`--from-tags` 必须读 `resources` 表才能聚合，故其 `--preview` 是**只读库、不写库**（仍会建立数据库连接），并非 `import_dataset` 那种「完全不连库」。`--dry-run`（连库走完整聚合、不写）与真导入语义不变。幂等语义按 `name` upsert（存在则更新 `description`/`aliases`/`category_id`/`sort_order`，不重复插入）。

> `--file` 手工/LLM 产出路径**本轮刻意未实现**：`cmd/import_knowledge_points` 在建立数据库连接前即拒绝 `--file`（见 `main.go` 的 `log.Fatalf`）。保留该开关与 §8.3 契约是为将来换产出方留缝，但读取端要等真有第二个产出方时再落地。

### 8.2 `--from-tags` 算法

1. 遍历 `resources.tags`（每资源一个 JSON 数组），统计标签词频
2. 剔除：命中 `knowledge_points.tag_stoplist` 的标签；词频 < `knowledge_points.min_resource_count`（默认 2）的标签；长度 < 2 的标签
3. 按词频 desc 排序，取前 `knowledge_points.max_knowledge_points`（默认 60）
4. `category_id` = 携带该标签的资源中最常见的分类
5. `name` = 标签原文，`aliases` 为空，`source = "tag"`
6. 直接落库（本轮不产生中间 JSON 文件；契约见 §8.3，实现时再落地）

### 8.3 导入契约（将来换 LLM 打标的关键缝）

```json
{
  "knowledge_points": [
    {
      "name": "监督学习",
      "aliases": ["supervised learning", "监督式学习"],
      "category": "人工智能",
      "description": "从带标签数据中学习映射关系",
      "sort_order": 10
    }
  ]
}
```

`--from-tags` 产出**同一种格式**。将来换成 LLM 打标脚本，只需它产出同一格式，导入命令与下游全部不动。

> 本契约在**有第二个产出方时**才实现读取端。本轮只有 `--from-tags` 一个产出方，
> 写一个无法端到端验证的读取器只会留下没被测过的代码（YAGNI）。缝保留在
> 「聚合是独立纯函数、落库是独立 service 方法」这两个接缝上。

### 8.4 阶段 0：先验证质量，再往下做

`--preview` 跑通后**人工检视 20-30 个知识点的产出质量**，再决定是否继续。

**已知风险**：B 站视频标签常为 `["机器学习","吴彦祖","教程"]` 这类混合体，聚合结果可能**更像标签云而非知识点**。停用词表能挡掉一部分噪音，但挡不住语义噪音。

**这不阻塞链路**——若质量不合格，把第 1 条决策换成手工预置或 LLM 打标即可，导入契约与下游零改动。本阶段唯一的沉没成本是那个聚合命令。

## 9. 与既有设计文档的关系

### 9.1 修订 `docs/recsys-design.md` 的排除项

`docs/recsys-design.md` §1.3 把「学习路径/知识图谱」列为「另一选题，本方案不做」，§7 再次重申。**本次是有意修订这条决策**，理由与边界：

| | 该文档排除的 | 本次做的 |
|---|---|---|
| 对象 | 学习路径 / 知识图谱 / 掌握度 / 测评 | 目标 → 知识点 → 分组资源 |
| 核心问题 | "按什么顺序学"、掌握度建模、依赖关系 | "这个目标该学哪几块、每块有哪些资源" |
| 是否排序 | 是（路径 = 顺序） | **否**，知识点之间无先后、无依赖 |
| 是否建模掌握度 | 是 | **否** |

即：**只做"分块"，不做"排序"，不碰掌握度**。§7 的其余项（ANN 索引、实时在线推理、复杂深度排序、掌握度/测评）依然不做。本节结论应在 spec 通过后回写进 `recsys-design.md`，避免两份文档互相矛盾。

### 9.2 修正 `docs/design.md` 的漂移

`docs/design.md` §3.2 写迁移工具为 golang-migrate，实际是 **GORM AutoMigrate**（`database/mysql.go:42`），仓库内无任何 `.sql` 迁移文件。本次新增表按**代码实际**处理；该漂移在实现里程碑里顺带修正。

## 10. 演进（本轮不做，但留好缝）

| 演进项 | 缝在哪 | 改动面 |
|---|---|---|
| LLM 离线打标 | 第 8.3 节导入契约 | 换产出方，导入命令与下游不动 |
| 知识点↔资源改为显式关联表 | 知识点获取收敛在 `KnowledgePointService` 内 | 只改该 service 的取资源实现 |
| 学习空间持久化（保存计划/进度/编辑资源） | 接口由 `GET /learning-plan` 演进为 `POST /learning-plans` + `GET /learning-plans/:id` | 新增表与 CRUD，知识点层不动 |
| 语义匹配取代子串匹配 | 第 6.2 节算法第 3 步 | 只改匹配函数 |
| 在线实时 LLM 拆解 | 引入 `KnowledgePointProvider` 接口的双实现 | 本轮不定义该接口（YAGNI） |

## 11. 页面与交互

### 11.1 首页 `/`（改造 `pages/home/index.vue`）

```
┌─────────────────────────────────────────────┐
│  想学点什么？                                 │
│  [ 例如：我想入门机器学习            ] [开始] │   ← 提问框（置顶）
├─────────────────────────────────────────────┤
│  ── 未提交目标时：现有推荐流原样保留 ──        │
│  （含 InterestPicker 冷启动引导、曝光打点）    │
└─────────────────────────────────────────────┘

提交后整页切换 ↓

┌─────────────────────────────────────────────┐
│  [ 我想入门机器学习              ] [重新规划] │
│  为你规划了 5 个知识点                        │
├─────────────────────────────────────────────┤
│ ▾ 监督学习                        12 个资源  │   ← el-collapse
│   ┌────┐ ┌────┐ ┌────┐ ┌────┐              │
│   │卡片│ │卡片│ │卡片│ │卡片│  … 最多 6 个   │
│   └────┘ └────┘ └────┘ └────┘              │
├─────────────────────────────────────────────┤
│ ▸ 线性代数                         8 个资源  │
│ ▸ 神经网络                        15 个资源  │
└─────────────────────────────────────────────┘
```

- 首个知识点默认展开，其余折叠
- 面板标题显示 `total`（该知识点关联资源总数），组内最多展示 M=6 个卡片；`total > 6` 时给一个"去资源库查看全部"的入口链接到 `/search?keyword=<知识点名>`
- 未匹配到知识点 → 顶部一句说明 + 单一分组（第 7 节）
- 加载态：面板骨架屏；错误态：可重试
- `?goal=` 与 URL 同步，刷新可恢复

### 11.2 新增文件

| 文件 | 职责 |
|---|---|
| `pages/home/GoalInput.vue` | 提问框：输入校验（非空、≤100 字）、提交 emit |
| `pages/home/KnowledgePointPanel.vue` | 单个知识点折叠面板：标题 + 资源数 + 组内 `ResourceCard` 网格 |
| `pages/home/LearningPlanView.vue` | 计划编排：载入、加载/空/错态、面板列表、兜底文案、曝光打点 |
| `api/learningPlan.ts` | `getLearningPlan(goal, knowledgePoints)` |

`pages/home/index.vue` 负责在「推荐流」与「学习计划」两个状态间切换。

### 11.3 曝光打点

面板内资源展示时调用 `recordImpressions('learning-plan', items)`，`scene` 用新值。

- 与首页 `home` 场景并存：**行级隔离**——`learning-plan` 与 `home` 是 `resource_impressions` 里两批不同的行，`home` 自身的行不被改动
- **但看板数值会变**：管理后台 CTR（`GET /api/v1/admin/recommendation-stats`）由 `StatsRepo.RecommendationStats` 对 `resource_impressions` **全表计数、不按 scene 过滤**算出（`backend/internal/repository/stats.go`），新增的 `learning-plan` 行会抬高分母、拉低看板 CTR。即「不干扰看板」只在**行级**成立，不在**数值级**成立
- 这是产品口径选择，三种做法择一：① 保持现状（新入口计入整体 CTR）；② 在 `RecommendationStats` 中排除该场景；③ 增加按 scene 的分场景统计。本轮不改统计代码，保留现状
- 新增一个可度量的场景，将来可在管理后台对比"两个入口"的效果
- 打点失败静默，不阻塞渲染（沿用现状）

## 12. 测试策略

| 层 | 测试 | 覆盖 |
|---|---|---|
| 后端 service | `TestPlanForGoal_MatchByName` / `_MatchByAlias` / `_MatchByCategory` | 三级匹配路径 |
| | `TestPlanForGoal_SkipEmptyKnowledgePoint` | 决策 #6：无资源的知识点被剔除 |
| | `TestPlanForGoal_RankByRecommendationOrder` | 命中 engine 缓存的资源排在前 |
| | `TestPlanForGoal_FallbackToKeywordSearch` | 未匹配 → 兜底单一分组，`matched=false` |
| | `TestPlanForGoal_ShortNameNotMatched` | `"C"` 这类短名不命中一切 |
| 后端 handler | `TestLearningPlan_RequiresAuth` / `_RejectsEmptyGoal` / `_RejectsLongGoal` | 401 / 400 |
| 后端命令 | `TestImportFromTags_StoplistAndMinCount` | 停用词与词频阈值过滤；`name` upsert 幂等 |
| 前端 | `KnowledgePointPanel.test.ts` | 渲染标题与资源数、折叠交互、空组不渲染 |
| | `LearningPlanView.test.ts` | 加载态、已匹配渲染、兜底文案、曝光打点被调用 |
| | `GoalInput.test.ts` | 空输入不提交、超长不提交、合法提交 emit 载荷 |
| | `mocks/handlers.ts` 补 `GET /learning-plan` | MSW 对齐真实响应形状 |

## 13. 交付物与验证

**交付物**
- `model/knowledge_point.go` + `AutoMigrate` 追加
- `cmd/import_knowledge_points/`（本轮实现 `--from-tags` / `--preview` / `--dry-run`；`--file` 刻意不实现，见 §8.1）
- `repository/knowledge_point.go`、`service/knowledge_point.go`、`handler/knowledge_point.go`、`router.go` 注册
- `configs/config.yaml` 新增 `knowledge_points.{tag_stoplist,min_resource_count,max_knowledge_points}`
- 前端：`GoalInput.vue`、`KnowledgePointPanel.vue`、`LearningPlanView.vue`、`api/learningPlan.ts`、`pages/home/index.vue` 改造、MSW handler
- 文档：`README.md` 功能模块、`docs/api-design.md` 新增「学习计划模块」章节（现为 1-8 模块，紧随其后）、`docs/recsys-design.md` 回写第 9.1 节结论、`docs/design.md` 修正迁移工具漂移

**验证**
```bash
cd backend  && go test ./... && go vet ./...
cd frontend && pnpm test && pnpm type-check && pnpm lint
```

**手动验证**
- `--preview` 输出人工检视，知识点质量可接受（阶段 0 闸门）
- 输入"我想入门机器学习"→ 返回多个知识点面板，每组有资源
- 输入一个库里没有对应标签的目标 → 触发兜底，页面不空白
- 输入空白 / 超长 → 前端拦截，后端 400
- 刷新页面（带 `?goal=`）→ 恢复知识点面板
- 首页未提交时推荐流与曝光打点行为与改造前一致

## 14. 风险与已知限制

| 风险 / 限制 | 影响 | 处理 |
|---|---|---|
| tag 聚合出的知识点像标签云 | 演示说服力弱 | 阶段 0 闸门（§8.4）；不合格则换 LLM 打标，下游零改动 |
| 冷门目标匹配不到知识点 | 用户看到兜底而非规划 | 兜底为关键词搜索（§7），有明确文案，绝不空白 |
| 组内候选池受 `List` 上限 100 限制 | 关联资源极多时，低分但命中 engine 推荐的资源可能落选 | 本轮接受：库内资源量级有限，且 `Sort="rating"` 已优先召回高质量候选 |
| 知识点身份绑定标签字符串 | 标签被改写或清理会导致知识点失联 | 导入按 `name` upsert 幂等；标签清理属资源侧操作，本轮不改 |
| 中文子串匹配误召回 | 短名或通用词过度命中 | 已加长度 ≥2 门槛（§6.2 第 3 步），停用词表兜底 |
| 新入口进入曝光统计 | 会改变管理后台 CTR 数值 | 独立 `scene=learning-plan`：行级与 `home` 分开；但 CTR 由全表 `resource_impressions` 计数得出（无 scene 过滤），新场景会抬高分母、拉低看板 CTR。口径取舍见 §11.3 |

## 15. 里程碑

> 按 `CLAUDE.md`：每个功能点一次提交，提交后停下等待确认。

| 里程碑 | 分支 | 提交信息 | 内容 |
|---|---|---|---|
| M1 | `feat/knowledge-point-import` | `feat(backend): 新增知识点模型与标签聚合导入` | 表 + 模型 + AutoMigrate + 导入命令 + 停用词配置 + **阶段 0 质量检视** |
| M2 | `feat/learning-plan-api` | `feat(backend): 学习目标匹配知识点并分桶返回资源` | repository + service（匹配/排序/兜底）+ handler + 路由 + 单测 |
| M3 | `feat/frontend-learning-plan` | `feat(frontend): 首页提问框与知识点面板` | 三个组件 + api 模块 + 首页双状态切换 + URL 同步 + MSW + 组件测试 |
| M4 | `feat/frontend-learning-plan-impression` | `feat(frontend): 知识点面板曝光打点` | 新场景 `learning-plan` 打点（可与 M3 合并，视改动量而定） |
| M5 | `docs/recsys-boundary-revision` | `docs: 回写知识点层对推荐系统边界的修订` | `recsys-design.md` §9.1 结论 + `api-design.md` 学习计划模块章节 + `design.md` 迁移工具漂移修正 |

**M1 是闸门**：阶段 0 质量检视不通过则停下重新讨论知识点来源，不进入 M2。
