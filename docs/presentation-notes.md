# edurec-platform 课题讨论讲解要点

> 用途：面向老师现场汇报的讲解提纲（科研/实验室课题讨论语境）
> 覆盖范围：本仓库 `edurec-platform`（platform 侧）。`edurec-engine` 是独立仓库，本文只描述它与平台的**接口契约**，不涉及它的内部实现与训练效果。
> 代码引用均给出仓库内路径，可现场打开对照。

---

## 1. 一句话定位 + 30 秒电梯陈述

### 一句话定位

**edurec-platform 是「教育资源推荐研究」的端到端落地载体与服务侧参照实现**：它负责把推荐系统真正需要的东西——用户、资源、行为、评分、对外服务接口、冷启动兜底、多源内容入池——都建好，然后把「算推荐」这件事整个交给独立仓库 edurec-engine，自己只做**数据下链（快照导出）**和**结果上链（推荐列表导入 + 缓存读服务）**。

### 30 秒电梯陈述（可直接照读）

> 这个平台是推荐研究的**服务侧那一半**。它不训练、不推理，模型完全在 edurec-engine 里。
> 两边的交叉接口就是平台的 MySQL：平台用 `export_snapshot` 把 users/resources/behaviors/ratings 导成 CSV 快照交给 engine，
> engine 训练 + 全量推理后输出 `{用户ID: [资源ID...]}`，平台由管理员通过一个导入接口写进 `Recommendation` 缓存表；
> 前端首页 `GET /api/v1/recommendations` 只读这张缓存表，未命中就按评分降序取热门兜底并回写缓存。
>
> 之所以刻意这样切分，是因为**离线批量训练 + 结果落库是真实推荐系统的 serving 形态**：训练和全量推理代价高、按周期跑，
> 在线服务只消费预计算结果。平台不做在线推理，所以模型可以随便换、重新训练不影响线上服务；engine 不做在线服务，所以它不需要背 web 框架和并发问题。
> 代价是时效性——推荐只反映「截至快照导出时刻」的状态，新行为要等下一轮刷新才生效。

---

## 2. 系统架构

### 2.1 全局视图

```
                     ┌──────────────────────────────────────────────────────────┐
                     │  浏览器                                                   │
                     │  Vue 3 + Vite + Element Plus + Pinia + Tailwind          │
                     │  frontend/src/pages/home/index.vue                       │
                     │    → getRecommendations(12)  frontend/src/api/recommendation.ts
                     └───────────────┬──────────────────────────────────────────┘
                                     │ HTTP/JSON   开发期：Vite proxy /api → :8080
                                     │             （frontend/vite.config.ts）
                     ┌───────────────▼──────────────────────────────────────────┐
                     │  后端 edurec-platform/backend （Go + Gin）                │
                     │                                                          │
                     │  ┌─ 中间件 ──────────────────────────────────────────┐   │
                     │  │ gin.Logger / gin.Recovery       (内置)            │   │
                     │  │ middleware.AuthRequired  → JWT 解析，写 ctx.user_id│   │
                     │  │ middleware.AdminRequired → 查 admins 表判管理员    │   │
                     │  └───────────────────────────────────────────────────┘   │
                     │                     │                                    │
                     │  ┌──────────────────▼────────────────────────────────┐   │
                     │  │ Handler 层  internal/handler/                     │   │
                     │  │   auth user resource category rating behavior     │   │
                     │  │   comment admin recommendation                    │   │
                     │  │   职责：参数校验、DTO 序列化、统一错误包装        │   │
                     │  │   （common.go: handleError；resource.go:          │   │
                     │  │     toResourceListItem 供推荐接口复用）            │   │
                     │  └──────────────────┬────────────────────────────────┘   │
                     │                     │ 单向依赖，下层不反向依赖上层       │
                     │  ┌──────────────────▼────────────────────────────────┐   │
                     │  │ Service 层  internal/service/                     │   │
                     │  │   recommendation.go        读缓存 + 热门兜底      │   │
                     │  │   recommendation_import.go 导入 engine 结果       │   │
                     │  │   crawl_import.go          ★ 多源落库内核 ImportItems│  │
                     │  │   dataset_source.go        数据集解析/字段映射     │   │
                     │  │   bilibili_online.go       exec 调 Python 在线抓取 │   │
                     │  │   auth / user / resource / category / rating /    │   │
                     │  │   user_behavior / comment / admin                 │   │
                     │  └──────────────────┬────────────────────────────────┘   │
                     │                     │                                    │
                     │  ┌──────────────────▼────────────────────────────────┐   │
                     │  │ Repository 层  internal/repository/  （GORM）      │   │
                     │  │   user admin category resource rating behavior    │   │
                     │  │   comment recommendation  refresh_token_store     │   │
                     │  └──────────────────┬────────────────────────────────┘   │
                     │                     │                                    │
                     │  ┌──────────────────▼────────────────────────────────┐   │
                     │  │ Model 层  internal/model/                         │   │
                     │  │   User Admin Category Resource Rating             │   │
                     │  │   UserBehavior ResourceComment CommentFetchState  │   │
                     │  │   Recommendation ← ★ 推荐结果缓存表                │   │
                     │  └───────────────────────────────────────────────────┘   │
                     └───────┬──────────────────────────────┬───────────────────┘
                             │                              │
                  ┌──────────▼──────────┐        ┌──────────▼────────────┐
                  │ MySQL               │        │ Redis                 │
                  │ 业务主存储 +         │        │ 仅用于 JWT refresh     │
                  │ Recommendation 缓存  │        │ token 存储             │
                  │ （database/mysql.go  │        │ (repository/          │
                  │   AutoMigrate）      │        │  refresh_token_store) │
                  └──────────┬──────────┘        └───────────────────────┘
                             │
                             │  ★ 两仓交叉接口 = 这个 MySQL + 文件快照
                             ▼
                  ┌──────────────────────────────────────────────┐
                  │ edurec-engine（独立仓库，Python + PyTorch）   │
                  │ DSSM 召回 → DeepFM 精排 → MMR 重排            │
                  │ 训练：python -m scripts.train_all             │
                  │ 推理：python -m scripts.run_batch_infer       │
                  └──────────────────────────────────────────────┘

  平台侧另有「非 HTTP 入口」的离线任务（cmd/，与 server 平级、共享 internal/）：
    cmd/server            在线服务入口（唯一对外 HTTP 服务）
    cmd/export_snapshot   数据下链：导出 CSV 快照 → backend/data/snapshots/<run_id>/
    cmd/import_bilibili   内容入池：B 站采集 JSON → resources 表
    cmd/import_dataset    内容入池：第三方数据集（字段映射配置化）→ resources 表
    cmd/demo_seed         演示播种：engine 的 sim 数据 → 业务表
    crawler/              B 站元数据采集（Python，与 Go 代码物理隔离）
```

### 2.2 分层与依赖规则

- **严格单向依赖**：`handler → service → repository → model`，下层不依赖上层（设计文档 §4.2 / 决策 #4、#23）。
- **handler 只做三件事**：参数校验、DTO 序列化、把 service 的 `apperror` 统一转成业务错误码。见 `backend/internal/handler/common.go` 的 `handleError`：`apperror.Error` 用它的 HTTPStatus + Code 返回，其他错误一律 500 + `10006`。
- **service 是纯业务逻辑**，通过 repository **接口**（如 `repository.RecommendationRepository`）依赖数据层，因此 service 层可以完全脱离数据库做单元测试——推荐模块正是这样测的（`backend/internal/service/recommendation_test.go` 里全是 `fakeRecommendationRepository`）。
- **统一响应格式**：`{code, message, data}`，业务错误码 `10001`–`10006`，见 `docs/api-design.md`。
- **路由装配**（`backend/internal/router/router.go`）：全部业务路由挂在 `protected := api.Group("")` 且统一 `Use(middleware.AuthRequired)`；管理类接口额外包 `middleware.AdminRequired(userRepo)`。全站只有 `POST /auth/register|login|refresh` 和 `GET /health` 不要求认证。

### 2.3 一处需要向老师说明的「文档 ↔ 代码」偏差（现场可主动提）

`docs/design.md` §3 技术栈表把 Redis 写成「JWT refresh token 存储 **+ 推荐结果缓存**」，但**代码里推荐结果缓存是 MySQL 的 `Recommendation` 表**（`model/recommendation.go`），Redis 实际只承载 refresh token（`repository/refresh_token_store.go`）。

- 这不算 bug，但**是文档口径没跟上实现**，建议对齐（也正好说明「缓存」一词在讨论里要区分：Redis 是 token 存储，推荐缓存是关系表）。
- 顺带：`docs/design.md` §4.2 / §8.1 提到 `migrations/`（golang-migrate）与 `docker/`、`docker-compose.yml`、`api/` 目录，**当前仓库里都不存在**；建表实际走 GORM `AutoMigrate`（`backend/internal/database/mysql.go:42`）。这类「设计意图 vs 落地范围」的差距建议在汇报时明确，避免被追问时被动。

---

## 3. 端到端数据流（含代码位置）

### 3.1 主链路 ASCII 流程图

```
【阶段 0】内容入池（三条渠道，一次性/周期性，与推荐链路解耦）
  ① 手工录入 / 管理后台   POST /api/v1/resources (管理员)      handler/resource.go
  ② demo_seed 模拟数据    cmd/demo_seed  ← engine dataset/sim 文件
     ⚠️ 当前库未启用此轨道（2026-09-19 按要求清除全部模拟数据）；
        代码与文档保留，需要回归演示时可复现，但**别当现状讲**
  ③ B 站离线采集          crawler/run.py → data/bilibili/latest.json
                          → cmd/import_bilibili
  ④ B 站在线搜索/评论     handler/resource.go + service/bilibili_online.go
                          └─ exec python crawler/online.py
  ⑤ 第三方数据集          cmd/import_dataset
                          └─ service/dataset_source.go（LoadDataset 字段映射）
  以上 ③④⑤ 最终都会走到同一个落库内核：
     service/crawl_import.go → ImportItems(items, opts, write)
        · 按 source_url 判重（FindBySourceURLs，一次性批量查）
        · 命中 → 只刷新 view_count / metadata，不新增行
        · 分类 find-or-create；字段按字符截断
  → 结果：全部混入统一的 resources 表（type ∈ course/video/article）

                    ▼ （所有资源进入同一 ID 空间，engine 侧看到的就是这一张表）

【阶段 1】数据下链：platform → engine
  ① cmd/export_snapshot/main.go
     · users.csv      仅 user_id（刻意不含 username/email 等个人字段）
     · resources.csv  resource_id,title,type,category_id,tags_json,
                      metadata_json,avg_rating,view_count,created_at
     · categories.csv category_id,name
     · behaviors.csv  user_id,resource_id,action,ts      （FindInBatches 5000/批）
     · ratings.csv    user_id,resource_id,score,ts        （FindInBatches 5000/批）
     · meta.json      contract_version=1, run_id, exported_at,
                      behavior_window{start,end}, 各表条数, files_sha256
     → backend/data/snapshots/<run_id>/
  ② 人工/脚本拷贝到 engine：dataset/platform_snapshot/<run_id>/
     （scripts/handoff.sh ② 步；平台与 engine 目录互不读写）

                    ▼

【阶段 2】engine 侧（独立仓库，平台不感知其内部）
  ③ python -m scripts.train_all       --data-source platform --snapshot-dir ...
  ④ python -m scripts.run_batch_infer --data-source platform --snapshot-dir ...
     → 输出 model/recommendations.json
       格式：{ "<user_id>": [<resource_id>, ...] }，平台原始 ID，覆盖快照全量用户

                    ▼

【阶段 3】结果上链：engine → platform
  ⑤ 拷回 backend/data/recommendations.json
     （路径来自 config：engine.recommendations_file）
  ⑥ 管理员 POST /api/v1/admin/recommendations/import   （无参数）
     handler/recommendation.go  → service/recommendation_import.go
       · os.ReadFile(filePath) → json.Unmarshal 成 map[string][]uint
       · 批量查存在性：users.FindByIDs(userIDs) / resources.FindByIDs(resourceIDs)
       · 只保留「platform 库里真实存在的 user / resource」（数值 ID 相同才匹配）
       · 逐用户过滤掉不存在的 resource_id
       · valid 为空 → 跳过该用户（保留其旧缓存行，不写空行）
       · 否则 recommendations.Replace(uid, idsJSON, now)
     → 返回统计 imported_users / skipped_users / imported_resources / skipped_resources
  ⑦ repository/recommendation.go → Replace()
     · 先 Delete 该 user_id 旧行，再 Create 新行 → 保证 UpdatedAt 就是本次生成时间
     · model.Recommendation: user_id UNIQUE, resource_ids JSON, created_at/updated_at(Unix)

                    ▼

【阶段 4】在线读服务
  ⑧ 前端首页 frontend/src/pages/home/index.vue
     onMounted → getRecommendations(12)   frontend/src/api/recommendation.ts
       → GET /api/v1/recommendations?limit=12      （JWT Bearer）
  ⑨ handler/recommendation.go Get
     · Middleware 取出 user_id；limit 默认 20、上限 50
  ⑩ service/recommendation.go Get(userID, limit)
     ├─ 命中缓存：parseResourceIDs → resources.FindByIDs → orderResources 按缓存顺序重排
     │            （已被删除的资源自动跳过）→ 截断到 limit → 返回 rec.UpdatedAt
     │            缓存 JSON 损坏 → 视为未命中，重新走兜底
     └─ 未命中缓存：resources.List(Page:1, PageSize:limit, Sort:"rating")
                     → 取 ID 列表 → Replace() 写缓存 → 返回
                     （repository/resource.go:96 `avg_rating DESC, id DESC`）
  ⑪ 前端渲染 ResourceCard 网格；updated_at 可显示「推荐生成时间」
```

### 3.2 每一步的代码位置速查表

| 步骤 | 动作 | 代码位置 |
|---|---|---|
| ① | 内容入池（统一内核） | `backend/internal/service/crawl_import.go` → `ImportItems` |
| ① | 数据集字段映射 | `backend/internal/service/dataset_source.go` → `LoadDataset` / `mapDatasetRecord` |
| ① | B 站在线抓取 | `backend/internal/service/bilibili_online.go`（`pythonTimeout = 30s`） |
| ② | 快照导出（数据下链） | `backend/cmd/export_snapshot/main.go` |
| ② | 快照契约版本 / 校验和 | 同上，`contractVersion = 1` + `files_sha256` |
| ③④ | engine 训练 / 推理 | **不在本仓库**，命令见 `docs/data-handoff.md` 与 `scripts/handoff.sh` |
| ⑤ | 结果文件路径配置 | `backend/configs/config.yaml` → `engine.recommendations_file` |
| ⑥ | 导入接口路由 | `backend/internal/router/router.go:93` |
| ⑥ | 导入 handler | `backend/internal/handler/recommendation.go` → `Import` |
| ⑥ | 导入过滤语义 | `backend/internal/service/recommendation_import.go` → `Import` |
| ⑦ | 缓存表模型 | `backend/internal/model/recommendation.go` |
| ⑦ | 缓存写入（先删后插） | `backend/internal/repository/recommendation.go` → `Replace` |
| ⑧ | 前端首页取推荐 | `frontend/src/pages/home/index.vue` + `frontend/src/api/recommendation.ts` |
| ⑨ | 推荐接口 handler | `backend/internal/handler/recommendation.go` → `Get` |
| ⑩ | 读缓存 + 热门兜底 | `backend/internal/service/recommendation.go` → `Get` |
| ⑩ | 热门排序口径 | `backend/internal/repository/resource.go:96` |
| — | 一键跑完整轮 | `scripts/handoff.sh`（六步与手动步骤一一对应） |

### 3.3 服务语义：空 / 缺失 / 陈旧（老师大概率会问「缓存没数据怎么办」）

这部分是有完整设计意图的，建议背下来：

| 情形 | 行为 | 依据 |
|---|---|---|
| 用户没有缓存行（新注册 / 从未被覆盖） | 按评分降序取热门资源，**写回缓存**后返回 → 不会出现「无推荐」 | `service/recommendation.go:68-92` |
| 缓存 JSON 损坏（脏数据） | 视为未命中，重新兜底生成并覆盖 | `service/recommendation.go:52-54` |
| 缓存里的资源已被删除 | `orderResources` 自动跳过，列表短一点但不报错 | `service/recommendation.go:104-117` |
| engine 单用户结果为空 / 用户不在平台库 | 导入跳过该用户，**保留其旧缓存行** | `service/recommendation_import.go:99-115` |
| 整份推荐文件为空或导入失败 | 缓存不变，服务继续返回旧的或兜底结果（陈旧但可用） | `service/recommendation_import.go:56-58` |

> 设计原则是「**宁可陈旧、不可失败**」：任何一环缺失都有确定的降级路径，服务永远有返回。

---

## 4. 关键技术决策与选型理由

> 编号说明：以下引用 `docs/design.md` 末尾「决策记录」表的编号。**该表编号不连续**（缺 11），且 **`12` 被两个不同决策项占用**（「engine 接入」与「Mock 方案」）；实际存在的编号是 `1–10` 与 `12–32`，共 30 条。下表覆盖其中的推荐链路与内容链路部分。

### 决策总览（编号 → 内容对照，便于现场定位）

| 编号 | 决策项 | 选择 |
|---|---|---|
| 1 | 与 edurec-engine 集成方式 | 离线批量 + 结果落库 |
| 2 | 与 engine 通信协议 | 文件/目录交接；在线 REST 留作演进 |
| 3 | engine 具体设计 | engine 独立仓库，消费平台快照；platform 提供导入接口 |
| 4 | 后端分层架构 | Handler → Service → Repository |
| 5 | 认证方案 | JWT Access + Refresh Token |
| 6 | API 设计规范 | CRUD RESTful + 操作 RPC 混合 |
| 7 | 状态管理 | Pinia |
| 8 | 数据库 ORM | GORM |
| 9 | 数据库迁移 | golang-migrate（**当前仓库未落地，实际用 AutoMigrate**） |
| 10 | API 文档 | 手写设计文档先行 |
| 12 | **engine 接入** | 离线批量 + 结果落库（已实现），platform 无在线推理 |
| 12（重号） | **Mock 方案** | MSW |
| 13–23 | 提交规范 / Docker / 配置管理 / 路由守卫 / 布局 / 日志 / 错误处理 / 测试策略 / 目录结构 | 均为工程规范类，非推荐核心 |
| 24 | 资源建模 | 统一抽象 + JSON metadata，`type` 字段区分 |
| 25 | 管理员 | 独立 admins 表（id, user_id） |
| 26 | 前端 HTTP 客户端 | Axios |
| 27 | CSS 方案 | Tailwind CSS + Element Plus |
| 28 | **外部内容来源** | 爬虫采集 + 命令导入（B 站公开元数据） |
| 29 | **采集内容建模** | 以 `type=video` 普通资源混入；按 `source_url` 判重，命中只刷新动态字段 |
| 30 | **在线搜索 + 评论爬取** | 在线实时（搜索耗尽本地后逐页爬、详情页无缓存时抓评论） |
| 31 | **第三方数据集导入** | 纯离线导入；慕课在线爬取因 TLS 指纹风控被判定不可行而放弃 |
| 32 | **采集落库契约泛化** | 抽出 `ImportItems(items, opts, write)` 通用内核 |

---

### 决策 A：两仓解耦，交叉接口就是 MySQL + 文件快照

**引用编号：1 / 2 / 3 / 12**

- **决策**：engine 与 platform **不直接调用**。数据下链用 `export_snapshot` 导出 CSV 快照，结果上链是 engine 输出的推荐列表 JSON 经管理员接口导入缓存表。两个仓库目录互不读写，文件经 `backend/data/` 人工/脚本传递（`scripts/handoff.sh` 固化了这一轮）。
- **备选方案**：
  1. platform 在进程内直接调 engine 的 Python（比如嵌个推理进程 / gRPC）；
  2. engine 暴露 REST 服务，platform 实时 RPC 调用；
  3. 共享数据库表，engine 直接读写平台业务表；
  4. **（选中）离线批量 + 文件快照 + 结果落库**。
- **为什么这样选**：
  - 「快照与结果文件只是 MySQL 数据的序列化载体」——交叉接口是一个**版本化的数据契约**（`contract_version = 1` + `meta.json` 里的 `files_sha256`），而不是一个网络协议。契约稳定后，engine 换模型、换框架、换语言都不影响平台。
  - 两仓独立迭代：平台不需要安装 PyTorch / CUDA，engine 不需要背 Web 框架和并发问题；两个仓库可以分别测试、分别部署。
  - 契约是**可审计的**：一次刷新产生一个 `run_id` 目录和一份带 sha256 的 `meta.json`，出了分歧可以逐文件核对，比在线接口的「当时返回了什么」可复现得多。
  - 对齐真实工业形态：训练 + 全量推理是重活，按周期批量跑；serving 只消费预计算结果。
- **代价 / 局限**：
  - **时效性是快照时刻的口径**：用户新行为、新上架资源要等下一轮「导出 → 训练/推理 → 导入」才生效；
  - **全量重算**：没有增量导出能力，快照是全表导出（`behaviors`/`ratings` 分页流式，但仍是全量）；
  - **需要人工/外部调度**：`scripts/handoff.sh` 目前是手工触发，没有内定时任务；
  - **没有原子性/事务保障**：拷文件、导库都是分步的，中途失败会停在中间态（好在读侧有兜底，不会崩）。

### 决策 B：平台不做在线推理（把推理留在 engine）

**引用编号：12（engine 接入）**

- **决策**：platform 进程内**没有任何模型代码、没有 PyTorch 依赖、没有 inference 路径**；`Recommendation` 是纯粹的「预计算结果表」。
- **备选方案**：platform 加载 engine 导出的模型权重后自行推理 / 缓存未命中时实时调 engine 的服务。
- **为什么这样选**：
  - **职责单一**：平台的职责是「服务化 + 数据治理」，模型的训练与推理留在研究侧（engine），这样模型迭代不需要动平台代码、不需要重新部署平台。
  - **延迟与稳定性**：实时推理会把模型的不确定性（加载失败、显存不足、超时）传导到用户请求路径上；物化结果把这条风险从在线路径彻底移走。
  - **可复现性**：结果落库后是**有版本的时间点快照**，离线回放/评测时不会被模型实时状态污染（不过见 §6：目前缺少把 run_id 记进结果的机制）。
  - **工程边界清晰**：这是「研究系统」和「生产系统」边界的一个具体表达——模型可以随便重训，线上服务形态不动。
- **代价 / 局限**：
  - 服务能力受限于「上一轮批处理」，无法做请求级个性化（比如会话内实时兴趣漂移）；
  - 文档在 `engine-integration.md` 里把「在线服务化（FastAPI + ANN）」和「近线增强（请求时过滤最近已看、新资源曝光通道、曝光打点）」列为**可选演进**，即已识别但未实现；
  - 没有 ANN 索引和在线特征服务，因此「换模型」目前等价于「重跑一遍全量推理」，实验迭代成本高。

### 决策 C：用缓存表落库，而不是缓存未命中就实时调 engine

**引用编号：12**

- **决策**：`GET /api/v1/recommendations` 只做「查表 → 按序取资源 → 返回」；未命中就本地兜底并写缓存，**绝不触发任何对 engine 的调用**。
- **备选方案**：Redis 缓存 + 未命中回源 engine / 直接实时计算。
- **为什么这样选**：
  - 读路径只有「一次主键/唯一索引查询 + 一次 `FindByIDs`」，尾延迟可控，而且**完全不受模型状态影响**；
  - `user_id` 上建了唯一索引（`model/recommendation.go` 的 `uniqueIndex`），每用户一行，读放大极小；
  - 用**关系表而不是 Redis** 存：推荐结果是「需要持久、需要审计、需要按用户批量覆盖写」的数据，落库天然支持 SQL 统计与离线核对（这个属性对后续做离线评测很关键，见 §7）；Redis 在本项目里专职承载 refresh token。
- **代价 / 局限**：
  - 多一份数据冗余，写侧要保证覆盖语义（`Replace` 用「先删后插」实现，非 UPSERT，理论上并发下有一个极短的空窗）；
  - 表名 `Recommendation` 叫「缓存」，但它其实是**物化结果表**——命名容易误导讨论，建议汇报时直接称「结果表/物化表」；
  - 目前**没有记录产生这批结果的 `run_id`/快照版本**，所以「这条推荐是哪个模型、哪份快照产出的」在库层面无法追溯（`created_at`/`updated_at` 只是导入时间）——这是科研上最该补的一环。

### 决策 D：热门兜底的必要性（冷启动的下界保障）

**引用编号：12（未命中按评分兜底）**

- **决策**：缓存未命中时，按 `avg_rating DESC, id DESC` 取前 `limit` 条资源，**写回缓存**后返回。
- **备选方案**：返回空列表 / 返回随机资源 / 返回最新资源（`created_at DESC`）/ 返回最热资源（`view_count DESC`）。
- **为什么这样选**：
  - **可用性优先**：首页是产品主入口，返回空列表在演示和真实使用中都是不可接受的失败态。「永远有返回」是硬约束。
  - **选评分而不是播放量**：`avg_rating` 是站内**用户主动表达的质量信号**，与「推荐质量」目标更对齐；B 站导入的视频 `avg_rating` 固定为 0（B 站无评分），所以它们不会靠评分挤占兜底位——这其实是个**隐性但合理的效果**，值得在讨论中说明。
  - **兜底也写缓存**：避免每次请求都重新扫一遍资源表（写一次、后续命中即可）；同时也让「兜底发生过」这件事在数据层留下痕迹（可用于统计冷启动比例）。
  - **按 `id DESC` 做次序稳定化**：避免评分相同的资源在分页/多次请求间顺序抖动。
  - 前端 mock 里也用同一语义（`frontend/src/mocks/handlers.ts:333` 按 `avg_rating` 降序），保证双模式下用户看到的行为一致。
- **代价 / 局限**：
  - 兜底结果**对所有未命中用户完全相同**，是个「全局热门 Top-N」，完全没有个性化；
  - 它和 engine 侧对冷启动用户的兜底是**两套独立逻辑**（engine 也会在出口给冷启动用户热门兜底），平台这套是「兜底的兜底」，两边口径未必一致——**这是一个可以做对比实验的点**：平台热门兜底 vs engine 冷启动兜底，谁的效果更好？
  - `avg_rating` 采用 `DECIMAL(2,1)`，取值粒度 0.1，新资源评分少时噪声大。

### 决策 E：ID 匹配策略——靠「平台数据源下全程使用平台原始 ID」这一约定

**引用编号：12 / 31**

- **决策**：导入时把 engine 输出里的 user_id / resource_id 当**数值**用，只保留平台库里真实存在的行，其余跳过；不做任何重映射。
- **备选方案**：
  1. 建立显式的 ID 映射表（engine 内部 id ↔ platform id），导入时查表转换；
  2. engine 输出用稳定的业务键（如 `source_url` / `username`）而不是数值 ID；
  3. **（选中）靠契约约定 ID 恒等 + 导入侧存在性过滤**。
- **为什么这样选**：
  - `docs/engine-integration.md`「ID 与覆盖口径」明确：**platform 数据源下 engine 全程使用平台原始 ID**，快照导出全量用户（含冷启动），清洗过滤只作用于训练集，推理对快照全量用户产出并在出口回映射为平台 ID；因此导入可全量命中、不存在 ID 错位匹配。
  - 存在性过滤是**廉价且必要**的防线：数值 ID 相同才匹配，因此 engine 用 sim/movielens 轨道（ID 与平台无关）产出的结果会被大量跳过——文档明确写了 **sim / movielens 轨道的结果「不得」作为导入真实平台的来源**，过滤逻辑就是把这条纪律变成机制。
  - 实现上做了批量存在性查询（`users.FindByIDs` / `resources.FindByIDs` 各一次），不是逐条查库。
- **代价 / 局限**：
  - **这是一个隐式契约**：`backend/internal/service/recommendation_import.go:44-45` 的注释还停留在旧口径——「engine 的用户/资源 ID 是它模拟数据集的内部编号，与平台库不一一对应」。**代码注释与 `engine-integration.md` 的说法现在不一致**，建议修正（否则后来人会按旧口径理解，误以为需要重映射）。
  - 没有 ID 映射表意味着**一旦有人用错轨道**（用 sim 结果导入真实平台），失败模式是「大量静默跳过」而不是显式报错——好在返回值里 `skipped_users` / `skipped_resources` 能暴露，但需要人去看；可以考虑加一个「跳过率超过阈值就拒绝整批导入」的保护。
  - 数值 ID 相同才匹配，天然无法表达「同一个资源在 engine 侧被合并/拆分了」的情况。

### 决策 F：内容判重键选 `source_url`，命中「只刷新动态字段、不新增行」

**引用编号：29（采集内容建模）、32（契约泛化）**

- **决策**：以 `source_url` 作为判重唯一键。未命中 → 插入新行；命中 → **不新增行**，只刷新 `view_count` 与 `metadata`；`title`/`description`/`category` 等保持平台侧的值。因此导入命令可反复执行（幂等）。
- **备选方案**：用 `bvid` / 数据集内部 ID 判重；命中时全字段覆盖；先清表再全量重建。
- **为什么这样选**：
  - **`source_url` 是跨来源唯一稳定的身份**：它能同时服务 B 站（`https://www.bilibili.com/video/<bvid>`）和第三方数据集（数据集自带的 target_url，或用 `source_url_template` 拼出来）。这正是决策 #32 把身份键从「必须有 `bvid`」放宽为「**`source_url` 必须能确定**」的原因。
  - **只刷新动态字段 = 不冲掉人工编辑**：运营改过标题/分类的资源，下一次导入不会被机器值覆盖。这是「人工维护」与「自动采集」共存的前提。
  - **`view_count` 必须刷新**：播放量会变，`sort=popular` 排序才不失真。
  - **宁可跳行也不产生半截 URL**：因为 `source_url` 是判重唯一依据，拼不出来就意味着每次导入都会重复插一行——这是必须拦住的失败模式。故数据集映射里「任一 `{占位符}` 取不到值 → 整体返回空 → 该行跳过」（`service/dataset_source.go:439-458`），B 站侧同理（`service/crawl_import.go:278-292`）。这是一个**「失败要显式」的设计选择**：宁可不导入，也不要污染 ID 空间。
  - 实现上的性能考虑：先 `FindBySourceURLs` 批量查出已存在的行做成 map，避免逐条查询；文件内重复的 `source_url` 后续条目按「已存在」处理（`crawl_import.go:230`）。
- **代价 / 局限**：
  - **`resources` 表没有 `source_url` 唯一索引**（`docs/bilibili-import.md` 与 `dataset-import.md` 的「已知限制」都列了这条）。当前靠「命令单进程串行执行」保证不重复；**一旦引入并发导入就会产生重复行**，需要 migration 加唯一索引（文档已给出结论）。
  - **上游字段更新同步不过来**：B 站/数据集改了标题，导入不会同步，需要先删行再导入——这是「不覆盖」策略的直接代价。
  - **分类静默新建**：分类名写错不会报错，会静默建一个新分类（两条链路同款行为），是个易踩的坑。

### 决策 G：字段映射全配置化（换数据集只改 YAML）

**引用编号：31（第三方数据集导入）、24（统一抽象 + JSON metadata）**

- **决策**：「外部字段 → 平台采集契约」的映射写在 `configs/config.yaml` 的 `datasets.<名称>` 段：`fields` 是「内部字段 → 外部字段候选列表（按序回退）」，另有 `items_path` / `format` / `resource_type` / `default_category` / `source_url_template` / `tag_separator` / `metadata`。**换数据集只改配置、不动代码。**
- **备选方案**：每个数据集写一个 Go 解析器；把映射写在代码常量里；用 JSONPath/JMESPath 表达式。
- **为什么这样选**：
  - 各家数据集的字段名和格式都不一样**且会更换**，为每个数据集写代码是不可持续的；外部字段名属于「运行期数据」而非「编译期结构」。
  - **候选列表 + 按序回退**：`title: ["title", "name"]` 这种写法让一份配置能同时吃多种变体（`firstNonEmpty` 会把空串/空数组/空对象都视为「没值」继续回退），支持点号路径（`author.nickname`）且优先按字面量键匹配（应对 `"a.b"` 这种扁平键，见 `walkPath`）。
  - **配置错误在启动时就暴露**：`config.Load()` 里调 `validateDatasets()`，未知的 `fields` key 与非法 `resource_type` / `format` 直接让服务启动失败（`internal/config/config.go:64-90, 200-206`）。文档的理由很到位：「拼错的 key 会让服务启动失败，而不是等导入跑完才发现整批数据都因缺字段被跳过」——**把配置错误从「静默降级」提升为「快速失败」**。
  - **归一化被收敛在映射之后**：`view_count` 兼容 `25612` / `"25612"` / `"25,612"` / `"2.5万"` / `"1.2w"` / `"3k"`（识别不了一律记 0，因为它是展示字段，不值得为此让整条记录失败）；`cover_url` 补 `//` → `https:`；`tags` 兼容数组/分隔符字符串/null，去重上限 10；`description` 为空回退到 `title`。这些规则集中在 `service/dataset_source.go`，**与落库内核完全解耦**。
  - 操作上是**三步走 + 两阶段报数**：`--preview`（不连库）→ `--dry-run`（连库不写）→ 真导入；解析阶段与落库阶段各有独立的跳过计数和原因（原因文案直接指明该改哪个 YAML key）。这套「先验证映射、再验证数量、最后写入」的流程本身就是可讲的设计点。
- **代价 / 局限**：
  - 表达力有上限：只支持「取路径 + 模板 + 几个归一化」，没有条件分支/拼接多个字段做 title 之类的能力；
  - `--file` / `--format` 只覆盖这两项，**字段映射永远以 YAML 为准**（这是刻意设计，避免「同一次导入究竟用了哪套映射」说不清）；
  - `metadata` 里原样收进去的外部字段（如 `obj_id`、`price`）**目前没有被推荐链路使用**，只是为将来换模型预留（对齐决策 #24）。

### 决策 H：多源内容以「普通资源」身份混入统一池

**引用编号：24（资源建模）、28 / 29（外部内容来源）、31 / 32（契约泛化）**

- **决策**：B 站视频以 `type=video`、第三方课程以 `type=course` **作为普通资源混入现有列表**，不新增专区、不改推荐链路。三条来源（手工/demo_seed、B 站、第三方数据集）最终都经过同一个 `ImportItems` 内核落进同一张 `resources` 表。
- **备选方案**：为外部内容单独建表/单独专区；外部内容不进推荐池、只在搜索里可见；每条来源一套导入代码。
- **为什么这样选**：
  - **推荐系统的输入必须是同一个 ID 空间**。多源内容如果分开建表，engine 侧就要做跨表 ID 对齐，收益为零、复杂度全在推荐侧。统一进 `resources` 后，engine 看到的资源池是单一、连续、可枚举的 ID 空间。
  - **泛化的方式是抽内核而不是加分支**（决策 #32）：`ImportItems(items, CrawlImportOptions{ResourceType, SourceURLTemplate}, write)`——两条来源的差异被压缩成两个参数（落库 type、URL 模板），判重、分类 find-or-create、字符截断全部复用。`cmd/import_dataset` 因此**不需要任何新的 Python 代码**，也不引入新的采集风险。
  - 写库前用 `model.IsValidResourceType` 再校验一次类型（`crawl_import.go:166-169`），**杜绝写入枚举外的值**——「宁可不导入，也不要写脏数据」。
  - 截断按**字符（rune）而非字节**，与 MySQL `varchar` 的长度语义一致（`truncateRunes`），避免中文被截半。
- **代价 / 局限**：
  - 不同来源的质量参差（B 站视频无评分、`avg_rating = 0`；第三方数据集可能缺元数据），但**在推荐里被同等对待**——`resource_type` 不影响推荐链路（文档明确写了）。这意味着「来源/质量」目前不是模型特征，可研究空间也在这里；
  - `resources` 用 JSON 列存 `tags` / `metadata`，**无法在 SQL 层做高效的标签聚合分析**，特征工程要落到 engine 侧做。

### 决策 I：MSW mock 与真实后端双模式（可脱离后端演示完整前端）

**引用编号：12（Mock 方案 = MSW）**

- **决策**：开发期默认 `VITE_MOCK=1`（`frontend/.env.development`），`main.ts` 在 bootstrap 时动态 `import('./mocks/browser')` 并 `worker.start({ onUnhandledRequest: 'bypass' })`；关掉该变量即走真实后端（Vite dev proxy `/api → http://localhost:8080`）。
- **备选方案**：本地起一个 mock server（json-server 等）；在 axios 拦截器里造假数据；写死假数据在组件里。
- **为什么这样选**：
  - **MSW 是浏览器级请求拦截（Service Worker）**，拦截发生在**网络层**，所以 axios 拦截器、错误码处理、超时逻辑全都走**真实代码路径**——「开发体验接近真实联调」（设计文档 §3.1），而 `json-server` 方案会绕开这些逻辑。
  - **前端可独立于后端和数据库开发/演示**：不需要 MySQL、Redis、engine 任何一环，`pnpm dev` 就能演示完整交互（含登录、评分、行为、推荐）。
  - **推荐 mock 与真实兜底语义对齐**：`handlers.ts:333` 按 `avg_rating` 降序取 limit 条并返回 `updated_at`，与后端兜底口径一致，避免「mock 看起来个性化、真实是热门」的认知错位。
  - 通过环境变量切换而非改代码，**双模式共用同一份 API client 与类型定义**（`src/api/*.ts`、`src/types/index.ts`），类型安全不断裂。
- **代价 / 局限**：
  - mock 里的推荐**是按评分排序的假个性化**，**无法用来验证 engine 的真实效果**——评审时要明确：演示用的是 mock，个性化效果必须走真实链路（后端 + 导入缓存）；
  - 两套数据源需要人工保持字段一致（mock 的 `db.ts` 有数据生成器 `generator.ts`），字段变更容易漂移；
  - MSW 只在开发/测试环境启用，生产构建里没有这层（这是对的，但意味着「生产路径」的验证依赖真实后端）。

### 决策 J：认证与安全（JWT 双 token）

**引用编号：5**

- **决策**：JWT Access Token（15 分钟）+ Refresh Token（7 天，存 Redis）。Access 过期后 `POST /auth/refresh` 换新；路由守卫未登录重定向 `/login`；管理员权限靠 `admins` 表存在性判断（决策 #25），由 `middleware.AdminRequired` 统一拦截。
- **为什么选**：无状态 Access Token 适合水平扩展；把长生命周期的 refresh token 放 Redis 可以**主动失效**（登出/风控），弥补纯 JWT 无法撤销的缺点。
- **代价 / 局限**：全站除 3 个认证接口外**都要求登录**（包括资源列表和详情），公开内容也需要先登录才能浏览——对「内容平台」来说这是个可以讨论的产品取舍；另外 `config.yaml` 里的 `access_secret` 仍是 `change-me-in-production` 占位值，属部署前必须替换项。

---

## 5. 可讲的研究点 / 创新点（工程实现里对科研有意义的部分）

> 这一节建议作为汇报的**重点段落**：把「工程怎么做」翻译成「研究可以问什么」。

### 5.1 离线批量 vs 在线服务：一个可量化的取舍

平台把「实时推理」这件事**整个排除在服务路径之外**，于是产生了一个干净的研究问题：

- 自变量：刷新周期（快照导出的频率）。因变量：推荐质量（离线指标）+ 服务成本（延迟、资源占用）。
- 由于结果落库、缓存表每用户一行、`updated_at` 记录了生成时间，**「结果有多陈旧」是可以被精确测量的**（`updated_at` 与线上行为时间戳的差）。
- 可以做的对比：**陈旧结果（batch）** vs **实时计算** vs **近线混合**（batch 为底 + 请求时轻量过滤/重排）。本项目已经把这条演进路径写进 `engine-integration.md` 的「演进方向」，但**没有实现、也没有量化对比**——这正好是课题可以填的空白。

### 5.2 冷启动兜底：平台侧有一份「可对照的基线」

- 平台的兜底策略（评分降序热门 Top-N，写回缓存）是一个**明确的、可复现的基线策略**。
- engine 侧在出口也做冷启动热门兜底——于是存在**两套兜底逻辑**：一套在平台（无缓存行时触发），一套在 engine（无行为用户时触发）。**它们的效果差异可以被实验**：哪个冷启动策略的离线指标更好？两者是否会互相掩盖（平台兜底把 engine 的冷启动结果覆盖掉）？
- 另一个可测的事实：`avg_rating` 对 B 站导入视频恒为 0，所以**多源内容在兜底排序里的地位是不对等的**——这本身就是一个可以建模的偏差（来源偏差 / position bias 的简化版）。

### 5.3 多源内容统一入池的建模问题

三条内容渠道（手工 / B 站公开元数据 / 第三方数据集）汇入同一张 `resources` 表，构成一个**真实的异构内容池**：

- **可研究方向**：来源（source）作为特征是否有用？统一池里「课程 vs 视频」的跨类型推荐效果如何（`resource_type` 目前完全不参与推荐）？
- `metadata` 是 JSON 列，B 站视频带 `duration` / `pubdate` / `like` / `favorite` / `review` / `typename`，第三方课程带 `obj_id` / `price`，**这些字段现在只被导出、没有被使用**——对特征工程是现成的素材。
- 采集侧的幂等设计（按 `source_url` 判重、只刷新 `view_count`/`metadata`）保证了**内容池的状态是可复现的**：任意两次导入之间只有动态字段变化。这对做「内容池演化对推荐的影响」类实验是必要条件。

### 5.4 DSSM + DeepFM + MMR 三段式在平台侧的「落地形态」

平台能看到的三段式的**产物形态**（不是它的内部）：

| 阶段 | 平台侧可见的形态 | 平台侧的落点 |
|---|---|---|
| DSSM 召回 | 候选集从全量资源收敛到一批；平台不可见 | 快照 `resources.csv` 提供全量候选池（`export_snapshot`） |
| DeepFM 精排 | 每个用户的**有序 ID 列表** `{uid: [rid, ...]}` | 推荐结果 JSON → 导入接口 → `Recommendation.resource_ids`（JSON 数组） |
| MMR 重排 | 列表的**多样性**体现在顺序里；平台不可见 | `orderResources` **严格保持导入顺序**（`service/recommendation.go:104-117`），因此重排效果不会被平台打乱 |

这里最值得讲的一点：**平台对顺序是「零干预」的**——`orderResources` 按缓存里的 ID 顺序重排，不做任何本地排序、过滤或补位（除了截断到 limit 和跳过已删除资源）。这使得平台的返回**就是 engine 的排名**，MMR 的重排语义能够端到端保真。要验证效果，只需要看 `resource_ids` 的先后顺序。

### 5.5 服务语义的「降级不失败」设计可作为可靠性研究素材

空 / 缺失 / 陈旧三种情形的明确降级路径（§3.3）构成了一个**有界的失败模型**。可以形式化地讨论：在「engine 产出缺失」「缓存未写」「资源被删」等扰动下，服务的可用性与结果质量如何变化。对推荐系统来说，「推荐服务本身挂了」比「推荐不够准」更严重，这个优先级在代码里是显式的。

### 5.6 可复现性设施（研究价值的工程基础）

- 快照带 `contract_version` + `run_id` + `files_sha256` + `behavior_window`（`cmd/export_snapshot/main.go`），**一轮训练的数据是可指纹化的**；
- `scripts/handoff.sh` 把「导出 → 训练 → 推理 → 导入」固化为一条命令，**一轮刷新是可复现的操作单元**；
- 导入返回 `imported_users / skipped_users / imported_resources / skipped_resources` 四个计数，**结果落库的全过程是可核对的**。
- 结论：数据侧的可复现性做得不错，**缺的是「结果侧可追溯」**（见 §6、§7）。

---

## 6. 当前完成度与局限（要诚实讲的部分）

### 6.1 已完成（本仓库内可验证）

**后端 Go + Gin + GORM（`backend/`）**
- 分层架构落地：handler / service / repository / model 四层，单向依赖；统一响应格式与业务错误码（`internal/response`、`internal/apperror`）。
- 认证：注册 / 登录 / 刷新，JWT 双 token，refresh token 存 Redis，`AuthRequired` + `AdminRequired` 中间件。
- 业务模块：用户、资源（含管理员 CRUD）、分类、评分评论（同用户同资源 upsert）、用户行为（view/click/favorite）、资源评论（B 站评论独立建模）、管理后台（用户/资源列表）。
- **推荐链路（核心）**：`GET /api/v1/recommendations` 读缓存 + 未命中热门兜底写缓存（`service/recommendation.go`）；`POST /api/v1/admin/recommendations/import` 导入 engine 结果（`service/recommendation_import.go`）。
- **数据下链**：`cmd/export_snapshot` 导出 5 张 CSV + `meta.json`（含 sha256 与各类计数）。
- **内容入池三条渠道**：`cmd/import_bilibili`（离线 JSON 导入）、`service/bilibili_online.go`（在线搜索逐页爬 + 评论抓取，30s 超时、风控不重试、stdout 只输出单行 UTF-8 JSON）、`cmd/import_dataset`（配置驱动字段映射，`--preview` / `--dry-run` / 真导入）。
- **统一落库内核**：`service/crawl_import.go` 的 `ImportItems`，被离线导入、在线搜索、数据集导入三处复用（决策 #32）。
- 测试：后端 16 个 `_test.go` 文件、约 102 个 `Test` 函数（`go test ./...`），推荐模块有覆盖**缓存命中顺序、limit 截断、已删资源跳过、未命中兜底与写缓存、limit 钳制、损坏缓存重生成、错误码映射**等分支的用例。

**前端 Vue 3 + Vite（`frontend/`）**
- Element Plus + Tailwind + Pinia + Axios（拦截器负责 JWT 附加与错误处理）+ Vue Router（路由守卫，未登录 → `/login`）。
- 页面：首页推荐流、登录、注册、资源详情（含评分与评论）、搜索（含在线无限滚动）、个人中心、资源上传、管理后台（仪表盘/用户/资源/分类）。
- MSW 双模式：`VITE_MOCK=1` 时浏览器级拦截，可脱离后端运行；mock 内还有数据生成器（`mocks/generator.ts`）。
- 测试：26 个 `*.test.ts`（Vitest + Vue Test Utils），覆盖组件、页面、store、router guards、mock handlers。
- 运维脚本：`scripts/handoff.sh` 一键跑完整轮刷新（六步）。

### 6.2 未接入 / 待验证（要主动承认）

1. **engine 是独立仓库，本仓库看不到任何训练效果指标**。没有 loss 曲线、没有离线指标（Recall@K / NDCG / HR / 覆盖率）、没有模型对比结果。**平台侧只能证明「链路通了」，不能证明「推荐变好了」**——这是最需要提前说明的一点，也是老师最可能追问的。
2. **平台侧没有离线评测设施**：没有把 `Recommendation` 结果与留出的行为/评分做回测的脚本，没有指标计算，没有基线对比（对照「热门兜底」这条现成基线）。**没有 AB 实验框架，没有曝光打点**（`user_behaviors` 只有 view/click/favorite 三种**正反馈**，没有 impression/exposure 日志）。
3. **推荐结果不可追溯**：`Recommendation` 表只有 `user_id / resource_ids / created_at / updated_at`，**没有记录产生它的 `run_id`、快照 sha256 或模型版本**。因此「这批推荐是哪次实验的产物」在库层面无法回答——做对比实验时无法区分结果来自哪次训练。
4. **没有与 engine 的端到端集成测试**：现有测试都是各仓内部的单元/接口测试；「导出 → 训练 → 推理 → 导入」全链路依赖人工执行 `scripts/handoff.sh` 验证，没有自动化校验（比如快照 schema 断言、导入后接口返回的冒烟测试）。
5. **导入过滤没有保护性阈值**：跳过率异常高（比如用错轨道）只会体现为 `skipped_*` 计数，不会拒绝整批导入。
6. **时效性与调度**：batch 刷新靠手工/外部触发，没有定时任务；近线增强（过滤最近已看、新资源曝光通道）已识别未实现。
7. **工程口径偏差（文档 ↔ 代码）**：
   - `design.md` §3 把 Redis 写成也做「推荐结果缓存」，实际推荐缓存是 MySQL 表，Redis 只存 refresh token；
   - `design.md` §4.2 / §8.1 提到的 `migrations/`（golang-migrate）、`docker/`、`docker-compose.yml`、`api/` 目录在当前仓库**不存在**，建表走 GORM `AutoMigrate`（决策 #9 未落地）；
   - `service/recommendation_import.go:44-45` 的注释仍是旧口径（说 engine ID 与平台不一一对应），与 `engine-integration.md`「platform 数据源全程使用平台原始 ID」不一致；
   - `design.md` 决策表编号缺 11、且 12 被两个决策项重号。
8. **无 `source_url` 唯一索引**：两条导入链路都靠「单进程串行」保证不重复，并发导入会产生重复行（文档已列为已知限制）。
9. **`skip` 计数语义与 API 文档不完全一致**：`imported_users`/`skipped_users` 的口径在「用户存在但过滤后无有效资源」这一情形上，实现把它计入 `skipped_users`（`service/recommendation_import.go:112-115`），而 `api-design.md` 对该字段的描述是「平台库中不存在，或无有效资源」——描述里包含了这种情况，但 `engine-integration.md` 写的是「导入跳过该用户、保留其旧缓存行」，两处表述需要统一。
10. **隐私相关**：快照 `users.csv` **刻意只导出 `user_id`**（不含 username/email），这是有意的隐私最小化设计，值得在汇报中点出来（同时也限制了基于用户画像特征的研究）。

---

## 7. 下一步计划（按优先级，偏科研方向）

### P0：补齐离线评测设施，让「推荐有没有变好」可回答

这是当前最大的短板，也是最容易出成果的一步。

1. **时间切分的离线评测**：利用快照里 `behaviors.csv` / `ratings.csv` 的 `ts` 字段（`export_snapshot` 已导出，且 `meta.json` 里有 `behavior_window`）做 leave-last-n-out 或时间窗口切分，构造测试集。
2. **指标计算**：对 `Recommendation.resource_ids` 计算 **Recall@K / Precision@K / NDCG@K / HitRate@K / 覆盖率（catalog coverage）/ 多样性（intra-list similarity）**。
3. **基线对照**：把「评分降序热门兜底」（平台已有，`Sort: "rating"`）与「最新资源」（`Sort: "latest"`）、「最热资源」（`Sort: "popular"`）作为 baseline，与 engine 结果做同口径对比。**这条基线是白送的**，因为 repository 里三种排序都已实现（`repository/resource.go:96-101`）。
4. **结果可追溯**：给 `Recommendation` 表加 `run_id` / `snapshot_sha256`（或独立的导入批次表），使每次实验的结果在库层面可区分、可回溯。这一步是**做对比实验的前置条件**。

### P1：召回 / 精排 / 重排的消融实验（把三段式拆开看）

5. **三段式消融**：在离线评测框架就位后，对比 `DSSM 召回 + DeepFM 精排`、`仅召回`、`仅精排`、`+MMR / -MMR` 四组配置的指标差异（模型改动在 engine 侧，评测与结果落库在平台侧）。因为平台**对顺序零干预**（§5.4），重排效果可以被端到端保真测量。
6. **召回规模 K 与精排 Top-N 的联合调参**：利用导入返回的 `imported_resources` 与评测指标，画「召回规模 → 指标/成本」曲线。

### P2：冷启动实验（平台侧已经有现成对照组）

7. **两套兜底的对决**：平台热门兜底 vs engine 冷启动兜底，在同一测试集上比冷启动用户群的指标；同时统计「平台兜底触发率」（有多少请求落在兜底上）与「engine 冷启动覆盖率」，看两者是否互相掩盖。
8. **冷启动的物品侧问题**：新上架资源没有交互，`avg_rating` 为 0（B 站视频恒为 0），可以研究**内容特征（`metadata` 里的 duration / typename / price / tags）驱动的冷启动曝光策略**，并对照当前「什么都不做」的基线。

### P3：时效性与在线/近线形态的量化取舍

9. **陈旧度实验**：测量 `updated_at`（推荐生成时间）与线上行为的时延，量化「结果陈旧 k 天」对离线指标的影响，为「batch 周期该多长」提供依据。
10. **近线增强的最小实现**：在 batch 结果之上加「请求时过滤最近已看」「新资源保底曝光位」，与纯 batch 做对比——这是 `engine-integration.md` 已列出的演进方向，成本低、可量化。

### P4：评测基础设施与工程口径收口

11. **补端到端集成测试**：对快照 schema、导入接口的过滤语义（含空文件 / 非法 JSON / 重复 ID / 全跳过）做自动化断言，让 `handoff.sh` 的每一步都有可回归的检查。
12. **加保护阈值**：导入时若 `skipped_users` / `skipped_resources` 占比超过阈值（如 50%）则拒绝整批，防止误用轨道静默污染缓存。
13. **修文档与注释口径**：Redis 与推荐缓存的关系、AutoMigrate vs golang-migrate、`recommendation_import.go` 的旧注释、决策表编号重号/缺号、`skipped_users` 的语义表述。
14. （可选）**给 `source_url` 加唯一索引**，为将来引入并发导入留出安全边界。

---

## 附：现场可能被追问的问题与建议回答方向

| 追问 | 建议回答要点 |
|---|---|
| 「推荐效果怎么样？」 | 坦诚：engine 是独立仓库，本仓库看不到训练指标；平台侧目前**只能证明链路贯通与可用性保障**，效果评测是下一步 P0。 |
| 「为什么不做实时推荐？」 | 训练 + 全量推理代价高，batch 是真实 serving 形态；平台不做推理使模型可随意重训而不影响线上。代价是快照时刻口径（§4 决策 B）。 |
| 「缓存没命中怎么办？」 | 按评分降序热门兜底并写回缓存；服务语义是「宁可陈旧、不可失败」（§3.3）。 |
| 「engine 和平台的 ID 怎么对齐？」 | platform 数据源下全程使用平台原始 ID，靠契约约定 + 导入侧存在性过滤保证；不用 ID 映射表，代价是注释与文档需要对齐（§4 决策 E）。 |
| 「为什么内容不单独建表？」 | 推荐输入必须是同一 ID 空间；三条渠道复用 `ImportItems` 统一落池（§4 决策 H）。 |
| 「重复导入会不会产生脏数据？」 | 按 `source_url` 判重、命中只刷新动态字段，命令幂等可反复执行；限制是 `source_url` 没唯一索引，靠单进程串行（§4 决策 F）。 |
| 「mock 模式下演示的是真推荐吗？」 | 不是。MSW mock 的推荐是按评分排序的假个性化，与后端兜底口径一致；真实个性化必须走「导入缓存表」的真实链路（§4 决策 I）。 |
