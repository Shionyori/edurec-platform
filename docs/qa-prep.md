# 答辩问答准备（老师现场追问）

> 用途：科研/实验室课题讨论的现场问答手稿。全部答复基于仓库代码与文档的真实情况，
> 未实现的能力一律在「答不上来 / 不夸大」一节明说。
>
> 阅读约定：每个问题给 **问法 → 照说版答复（3～5 句）→ 依据（文件:行号 / 决策编号）→ 追问预判**。
> 行号对应写作时的仓库状态；引用文件均为仓库内真实文件。

---

## 0. 一页速记（真实现状）

| 维度 | 真实情况 |
|---|---|
| 两仓关系 | platform（Go+Vue）与 edurec-engine（Python+PyTorch）目录隔离；交叉接口是 **platform 的 MySQL**，载体是快照目录与推荐结果 JSON（`docs/engine-integration.md:13-14`） |
| 平台是否做推理 | **不做**。platform 无模型代码，权重只留在 engine（`docs/engine-integration.md:7-11`） |
| engine 是否在线服务 | **不是**。engine 离线训练 + 全量推理批量产出（`docs/engine-integration.md:16-30`） |
| 读侧链路 | `GET /api/v1/recommendations` 读 MySQL 缓存表 → 未命中按 `avg_rating DESC, id DESC` 热门兜底并写缓存（`backend/internal/service/recommendation.go:33-93`） |
| 结果上链 | engine 输出 `{"<user_id>":[<resource_id>,…]}` → 管理员 `POST /api/v1/admin/recommendations/import` → 过滤平台不存在的 ID 后覆盖写缓存（`backend/internal/service/recommendation_import.go:46-128`） |
| 内容来源 | 手工录入 / demo_seed 模拟数据 / B 站公开元数据采集（离线批量 + 在线实时）/ 第三方数据集离线导入（`docs/design.md:382-385`） |
| 缓存介质 | 推荐结果缓存在 **MySQL `recommendations` 表**，Redis 只放 refresh token（`backend/internal/repository/refresh_token_store.go:22-35`） |
| 效果评估 | **完全没有**：无离线指标、无 AB 实验、无曝光打点、无线上监控（全仓 grep `impression|exposure|AUC|CTR` 零命中） |
| 测试 | 后端 `go test ./...` 通过，均为单测（fake 仓库），`handler/`、`repository/` 目录**无测试文件**；前端 25 个测试文件约 156 条用例（本机沙箱无法启动 vitest，见 §10.1） |
| 迁移 | 仓库**没有** `migrations/` 目录，实际靠 GORM `AutoMigrate`（`backend/internal/database/mysql.go:41-53`）；`docs/design.md:74` 写的 golang-migrate 属规划未落地 |
| 部署 | 仓库内**没有** Dockerfile / docker-compose.yml（`docs/design.md:308-328` 写的是规划） |

---

## 1. 架构与两仓边界

### Q1. 为什么要拆成两个仓库？边界到底画在哪里？

**问法**：「你为什么把推荐引擎和平台拆成两个仓库？它们之间怎么通信？边界在哪？」

**照说版**：
1. 两仓的交叉接口只有两个：**platform 的 MySQL 数据库**，以及作为该库数据序列化载体的**文件快照**。
2. 数据下链是 `export_snapshot` 把 users/resources/categories/behaviors/ratings 导成 CSV 快照目录带 sha256；engine 读这个快照做训练和全量推理。
3. 结果上链是 engine 输出 `{"<user_id>": [<resource_id>, …]}` 的 JSON，管理员调导入接口写进 `recommendations` 缓存表。
4. 边界很清楚：**platform 不做模型推理，engine 不做在线服务**；两者不互相调用接口。
5. 这样 engine 换算法、换模型、换特征工程，platform 一行都不用改。

**依据**：`docs/engine-integration.md:13-14`（交叉接口即 MySQL）、`docs/engine-integration.md:46-63`（快照与导入）、`backend/cmd/export_snapshot/main.go:41-163`（导出字段与 sha256）、`backend/internal/router/router.go:93`（导入路由）、`docs/design.md:357`（决策 #12）。

**追问预判**：
- 「为什么不让 platform 直接调 engine 的 HTTP 接口？」→ 见 Q4；本形态是既定接入方式（`docs/engine-integration.md:11`），在线服务化列在演进方向（`docs/engine-integration.md:81-86`）。
- 「快照里有什么？」→ `meta.json` + 5 个 CSV；`users.csv` **只导出 user_id，不含邮箱、昵称等个人信息**（`backend/cmd/export_snapshot/main.go:47-54`）。

### Q2. 快照契约的版本和校验怎么保证？

**问法**：「交接文件格式改了怎么办？怎么知道 engine 读的是哪份数据？」

**照说版**：
1. `meta.json` 里有 `contract_version`（当前为 1）、`run_id`（按 `20060102_150405` 生成）、`exported_at`、各表行数，以及 5 个 CSV 的 **sha256**。
2. `run_id` 同时是快照目录名，训练和推理都指向同一个 `dataset/platform_snapshot/<run_id>`，保证训练/推理口径一致（`docs/data-handoff.md:46-48`）。
3. 契约版本号是给未来演进留的字段；目前**没有**做 engine 侧版本校验或拒绝逻辑，属于已知欠缺。

**依据**：`backend/cmd/export_snapshot/main.go:22`（`contractVersion = 1`）、`:138-162`（meta 与 sha256）、`:41-45`（目录名）、`docs/data-handoff.md:17-20`。

**追问预判**：「版本不匹配会怎样？」→ 诚实说：当前没有校验，靠人工对齐；这是一处可以立刻补的工程点。

### Q3. 为什么读侧不查 engine，而是查一张缓存表？

**问法**：「用户请求推荐时，你们真的没调用模型？」

**照说版**：
1. 真的没有。`GET /api/v1/recommendations` 的实现只有两条路径：读 `recommendations` 表命中就返回，未命中就按评分取热门并写缓存。
2. 这个形态对齐真实推荐系统的 **batch 链路**：训练与全量推理代价高，按周期批量跑，serving 只消费预计算结果。
3. 好处是首页响应与模型复杂度解耦，模型再重也不影响接口 P99。
4. 代价是时效性——新行为要等下一轮「导出 → 训练/推理 → 导入」才生效。

**依据**：`backend/internal/service/recommendation.go:33-93`、`backend/internal/handler/recommendation.go:26-60`、`docs/engine-integration.md:9-11,76-79`。

**追问预判**：「时效性具体多差？」→ 取决于人工触发刷新的频率，代码里没有定时任务、没有 TTL、也没有「数据多旧就视为过期」的判定（`docs/engine-integration.md:78-79`）。

### Q4. 平台不做在线推理，是不是因为做不了？

**问法**：「为什么不直接在 Go 服务里加载模型做实时推理？」

**照说版**（高风险问题，态度要诚实）：
1. 是**主动取舍**，不是能力缺失。engine 是 PyTorch（DSSM 召回 + DeepFM 精排 + MMR 重排），在 Go 服务里复现或在进程内跑 Python 推理，会把 serving 的稳定性和模型迭代耦合在一起。
2. 现形态让两边各自演进：engine 只管离线，platform 只管读表，失败面小。
3. 另一个原因是数据条件——平台侧真实行为量很小（详见 §10），实时推理在这个数据量下也换不来效果提升。
4. 在线服务化（engine 加 FastAPI + ANN，缓存未命中时实时调用）是写明的**可选演进**，不是已实现能力。

**依据**：`docs/engine-integration.md:7-11`（platform 侧不运行模型）、`:81-86`（演进方向 1）、`docs/design.md:96`（决策 #12 末句「实时服务化留作可选演进」）。

**追问预判**：
- 「那你的工作量在哪？」→ 见 Q29。
- 「实时化要改什么？」→ engine 侧加 serving 层与 ANN 索引，platform 侧加缓存未命中的实时调用与降级，两边都要处理超时与熔断——目前都没做。

### Q5. 前后端与鉴权是怎么切的？

**问法**：「后端的层次结构和管理员权限怎么实现的？」

**照说版**：
1. 后端是 Handler → Service → Repository 扁平三层，单向依赖，Redis 与 MySQL 都只在 repository 层出现。
2. 认证是 JWT Access（15 分钟）+ Refresh（7 天，存 Redis）；`AuthRequired` 只校验 access token 签名与类型。
3. 管理员不是角色字段，而是独立的 `admins` 表（`id, user_id`），`AdminRequired` 每次请求查库判断。
4. 所以「给某人管理员」就是往 `admins` 插一行，不用改用户表。

**依据**：`backend/internal/router/router.go:18-95`、`backend/internal/middleware/auth.go:17-37`、`backend/internal/middleware/admin.go:11-33`、`backend/internal/util/jwt/jwt.go:29-64`、`backend/internal/repository/refresh_token_store.go:30-57`、`docs/design.md:379`（决策 #25）。

**追问预判**：「每个请求都查一次 admins 会不会慢？」→ 当前量级无压力；要优化就加进程内短 TTL 缓存或把角色写进 JWT claim（后者会带来撤权延迟问题，需要权衡）。

---

## 2. 推荐链路与缓存策略

### Q6. 完整走一遍：从用户打开首页到看到卡片，中间发生什么？

**问法**：「把推荐这条链路从头到尾讲一遍。」

**照说版**：
1. 前端首页 `onMounted` 调 `getRecommendations(12)`，请求 `GET /api/v1/recommendations?limit=12`。
2. 后端 `RecommendationService.Get` 先 `FindByUserID` 查缓存表。
3. 命中：把 `resource_ids` JSON 解析成 ID 列表，`FindByIDs` 批量查资源，再按缓存里的**原始顺序**重排返回（`orderResources`），已删除的资源自动跳过。
4. 未命中：按 `avg_rating DESC, id DESC` 取前 N 条热门，写入缓存表，返回这批资源。
5. 响应带 `list` 与 `updated_at`（缓存行的生成时间，Unix 秒转 RFC3339）。

**依据**：`frontend/src/pages/home/index.vue:11-21`、`frontend/src/api/recommendation.ts:4-6`、`backend/internal/service/recommendation.go:41-93`、`backend/internal/service/recommendation.go:104-117`（顺序重排与跳过已删）、`backend/internal/handler/recommendation.go:56-59`。

**追问预判**：「顺序是模型的排序吗？」→ 命中缓存时是；兜底时是按评分排序，与模型无关。

### Q7. 缓存命中后为什么还要回表查资源？

**问法**：「缓存里只存 ID，不怕多一次查询吗？」

**照说版**：
1. 因为缓存表存的是 **ID 列表而不是内容快照**，资源标题被管理员改过、或资源被删掉，都要能立刻反映到首页。
2. `orderResources` 用 map 按 ID 对齐，只保留真实存在的行，所以缓存里残留的已删 ID 不会报错、只是被跳过。
3. 代价是每次命中多一次 `IN` 查询，量级上完全可接受。
4. 好处是避免「缓存里的标题和详情页不一致」这类经典问题。

**依据**：`backend/internal/service/recommendation.go:56-64,104-117`、`backend/internal/repository/resource.go:128-137`。

**追问预判**：「那如果资源全被删了？」→ 缓存里 ID 全失效时 `ordered` 为空，接口会返回**空列表且不触发兜底**（兜底只在「没有缓存行」时发生）。这是真实边界，测试里覆盖的是「部分被删」而非「全部被删」（`backend/internal/service/recommendation_test.go:110`）。见 Q25。

### Q8. 缓存的读写与失效是怎么设计的？

**问法**：「推荐缓存什么时候写、什么时候更新？有没有过期时间？」

**照说版**：
1. 只有两处写入：**缓存未命中时的兜底**，和**管理员导入 engine 结果**。都是 `Replace`——先删该用户旧行再插入新行。
2. `recommendations` 表对 `user_id` 建了唯一索引，语义是「每个用户只保留一份结果」。
3. **没有 TTL、没有过期判定**。陈旧与否只体现在响应里的 `updated_at`，前端目前也不对用户展示这个时间。
4. 删除-插入两步不是事务，理论上并发请求同一用户时有极小概率撞唯一索引；实际触发场景是「同一用户并发首次请求」+「同时导入」。
5. 导入语义是「整份文件成功才继续」，中途出错会直接返回 500，缓存保持原样（陈旧但可用）。

**依据**：`backend/internal/repository/recommendation.go:39-53`、`backend/internal/model/recommendation.go:6`（`uniqueIndex`）、`backend/internal/service/recommendation.go:87-90`、`backend/internal/service/recommendation_import.go:121-123`、`docs/engine-integration.md:70-74`（空/缺失/陈旧语义）。

**追问预判**：
- 「为什么要先删再插？」→ 为了让 `created_at/updated_at` 精确等于本次生成时间（代码注释在 `repository/recommendation.go:40`）。
- 「为什么不用 UPSERT？」→ 更干净，能避免时间戳被 GORM 覆盖；当前实现是开发阶段选择，属于可改进项。

### Q9. 兜底真的是「热门」吗？

**问法**：「热门兜底用的是什么指标？」

**照说版**：
1. 用的是**平均评分降序**（`sort=rating` → `ORDER BY avg_rating DESC, id DESC`），不是播放量。
2. 这里要诚实：站内评分数据极少（真实库内 ratings 基本为空），所以 `avg_rating` 普遍为 0，兜底实际上退化成**按 ID 倒序**——它是「有结果」的保证，不是「好结果」的保证。
3. 如果换成按播放量（`sort=popular`）可能更合理，因为 B 站导入的资源带真实 `view_count`；现在没换。
4. 兜底结果会被写进缓存，所以「先被兜底过的用户」在导入 engine 结果前看到的一直是这份列表。

**依据**：`backend/internal/service/recommendation.go:69-73`、`backend/internal/repository/resource.go:94-101`（排序分支）、`docs/bilibili-import.md:119`（B 站资源 `avg_rating` 固定 0）、`backend/cmd/export_snapshot/main.go:72`（快照里 avg_rating 以 0 为主）。

**追问预判**：「那为什么不改？」→ 可以直接改成 `view_count` 兜底，一行枚举值的事；当前没改是因为没有评测手段来证明哪个更好——这恰好暴露了缺评测设施的问题（见 Q20）。

### Q10. 导入接口是怎么过滤的？返回的四个计数什么意思？

**问法**：「`imported_users / skipped_users / imported_resources / skipped_resources` 分别是什么？」

**照说版**：
1. 先把 JSON 的所有 user key、所有 resource id 收集起来，**各做一次批量存在性查询**（不是逐条查）。
2. 遍历时：用户不在平台库 → `skipped_users++`；资源不在平台库 → 逐条 `skipped_resources++` 并从该用户的列表里剔除。
3. 过滤后有效资源为空 → 该用户计入 `skipped_users`，**保留其旧缓存行不动**。
4. 非数字的 key 直接忽略（不算 skipped 也不报错）。
5. 写入用 `Replace`，每个成功用户 `imported_users++`，有效 ID 总数累加到 `imported_resources`。

**依据**：`backend/internal/service/recommendation_import.go:60-126`、`docs/api-design.md:642-663`。

**追问预判**：
- 「为什么不整体事务？」→ 当前逐个用户写，中途失败已写入的部分不回滚。文件通常由管理员手动触发，可接受；要严格就得包事务。
- 「会原子替换全量结果吗？」→ 不会。只覆盖文件里出现的用户，其他用户的旧缓存保留（这是刻意的，见 `docs/engine-integration.md:73`）。

---

## 3. 冷启动与热门兜底

### Q11. 冷启动怎么处理？

**问法（高风险）**：「新用户没有任何行为，你怎么推荐？」

**照说版**：
1. 分两层。**平台侧**：该用户没有缓存行，`GET /recommendations` 就走按评分的热门兜底并写缓存，所以永远不会返回空。
2. **engine 侧**：快照导出的是**全量用户（含冷启动）**，engine 在训练集上做行为过滤，但推理对快照全量用户产出结果，无行为用户由 engine 内部走热门兜底。
3. 所以完整链路下，冷启动用户拿到的是 engine 的热门兜底；只有「engine 结果还没导入」时才会用到平台这层兜底。
4. 诚实补充：平台这层兜底因为站内评分稀疏，实际接近按 ID 倒序，效果上只是「有内容可看」。

**依据**：`docs/engine-integration.md:66-68`（导出全量用户含冷启动、清洗只作用于训练集）、`docs/data-handoff.md:58`（冷启动用户走热门兜底）、`backend/internal/service/recommendation.go:68-92`、`docs/engine-integration.md:70-72`。

**追问预判**：
- 「为什么不用内容侧特征做冷启动（分类/标签画像）？」→ engine 侧是否有内容特征链路由 engine 仓库决定，platform 侧没有实现基于内容画像的兜底。不要替 engine 编造。
- 「新注册用户在库里吗？」→ 是，注册即写 `users` 表，下次导出快照就带上。

### Q12. 用户不在快照里（导入之后才注册）会怎样？

**问法**：「导入之后新注册的用户，下一次请求会不会没结果？」

**照说版**：
1. 不会有问题。导入时文件里没有这个用户，他的缓存行也没被写，请求时走平台兜底并写缓存。
2. 下一轮导出快照会带上他，engine 推理时按冷启动路径出结果，导入后覆盖他的缓存行。
3. 也就是说「新用户」的过渡体验是平台兜底，稳定性由兜底保证，不依赖 engine。

**依据**：`backend/internal/service/recommendation.go:68-92`、`backend/internal/service/recommendation_import.go:99-102`、`docs/engine-integration.md:70-74`。

### Q13. 缓存是空的，接口会不会挂？

**问法**：「如果 `recommendations` 一行都没有，首页还能打开吗？」

**照说版**：
1. 能。空表就是「全部未命中」，每个用户首次请求都会走兜底并顺便把缓存建起来。
2. 这也是设计意图：文档明确写「空/缺失缓存不会导致无推荐」。
3. 副作用是所有用户首次请求都会各写一行缓存，`recommendations` 行数等于活跃用户数（`user_id` 唯一索引保证不膨胀）。

**依据**：`README.md:27`、`docs/engine-integration.md:72`、`backend/internal/model/recommendation.go:6`。

**追问预判**：「如果 engine 结果文件是空的呢？」→ 导入接口对空文件返回全 0 统计、**不改缓存**（`recommendation_import.go:56-58`），服务继续返回旧结果或兜底。

---

## 4. ID 映射与数据一致性

### Q14. engine 和平台的 ID 怎么对齐？对不上怎么办？

**问法（高风险）**：「两个仓库的用户 ID、资源 ID 怎么保证是同一套？如果对不上呢？」

**照说版**：
1. 对齐机制是「**platform 原始 ID 全程透传**」：快照导出的是平台库的 `users.id` / `resources.id`，engine 在 platform 数据源下出口回映射为平台 ID，所以导入时是按相同数值 ID 匹配的。
2. 兜底口径是「**不存在就跳过，不猜**」：导入时按平台库中真实存在的 user/resource 过滤，不存在的计入 `skipped_users` / `skipped_resources`，绝不按顺序或相似度硬凑。
3. 因此**不存在错位匹配**（把 A 的结果给 B）这种最糟的情况——最坏结果是「少推荐」，而不是「乱推荐」。
4. 但要诚实说明两点遗留风险：一是 **sim/movielens 轨道的数据与平台 ID 无关，明确禁止导入真实平台**；二是如果误把模拟数据播种进生产库，两套 ID 会撞在一起而语义不同，这是 ID 语义冲突而非数值错位。

**依据**：`docs/engine-integration.md:64-68`（ID 与覆盖口径）、`backend/internal/service/recommendation_import.go:43-45,99-115`、`docs/engine-integration.md:68`（sim/movielens 不得作为导入来源）、`backend/cmd/demo_seed/main.go:21-23`（demo_seed 保留原始 ID，用于演示）。

**追问预判**：
- 「为什么不做显式 ID 映射表？」→ 因为不需要：platform 轨道下两边本来就是同一套 ID，加映射表反而引入新的不一致源。如果将来接入外部数据集训练，才需要 source→platform 的映射表，那是演进项。
- 「怎么防止 sim 数据误导入？」→ 目前靠流程约束（文档明确禁止）+ 人工检查，**没有代码级闸门**（比如校验 run_id / data_source）。这是可以补的。

### Q15. 重复 ID、重复行怎么判重？

**问法**：「engine 的输出里同一个用户有重复资源 ID 会怎样？导入能重复执行吗？」

**照说版**：
1. 导入不做去重：`valid` 是原样保留的切片，重复 ID 会被重复写进 JSON 数组。返回列表时 `orderResources` 按 ID 顺序取，理论上会出现同一个资源占两格。
2. 但导入本身是**幂等**的：同一个用户重复导入是 `Replace`（整行覆盖），不会重复插行。
3. 内容导入（B 站/数据集）都按 `source_url` 判重，第二次跑全部走更新分支、不新增行。
4. 「engine 输出里有重复 ID」目前没有防御，属于已知欠缺。

**依据**：`backend/internal/service/recommendation_import.go:104-117`、`backend/internal/repository/recommendation.go:39-53`、`backend/internal/service/crawl_import.go:209-233`、`docs/bilibili-import.md:122-135`（第 1 次新增 / 第 2 次全刷新）。

### Q16. 内容导入的一致性语义是什么？

**问法**：「重复采集同一批视频，会不会把管理员改过的标题冲掉？」

**照说版**：
1. 不会。判重键是 `source_url`；命中时**只刷新 `view_count` 和 `metadata`**，标题、简介、分类保持平台侧的值。
2. 所以导入命令可以反复执行：第一次新增，第二次全部走「刷新」分支。
3. 反过来也意味着 B 站侧改了标题不会同步过来——需要同步只能先删行再导入，这条写在已知限制里。
4. `source_url` 判不出来（模板为空且 item 没给 URL）的行会被**跳过而不是硬插**，因为它是判重的唯一依据，拼不出来就会每次重复插入。

**依据**：`backend/internal/service/crawl_import.go:273-322`（校验与 URL 推导）、`:209-221`（命中只刷动态字段）、`docs/bilibili-import.md:122-138,174-181`（已知限制）、`docs/dataset-import.md:134-146`（行丢弃条件）。

**追问预判**：「`resources.source_url` 有唯一索引兜底吗？」→ **没有**，只有普通索引（`backend/internal/model/resource.go:34`）。单进程串行导入不会重复；并发导入才会出问题，文档已把它列为已知限制。

---

## 5. 算法与效果评估

### Q17. engine 的算法是什么？

**问法**：「你们的推荐算法是怎么设计的？」

**照说版**：
1. engine 是独立仓库的 PyTorch 实现：**双塔 DSSM 召回 → 多任务 DeepFM 精排 → MMR 重排**。
2. 我在 platform 这一侧的职责边界是：保证数据能正确进、结果能正确落库、服务能稳定返回，**不参与算法实现**。
3. 从 platform 视角，engine 就是一个「CSV 快照进、排名 ID 列表出」的黑盒契约。
4. 具体到网络结构、损失函数、训练超参，要以 engine 仓库为准，我不在 platform 侧复述细节。

**依据**：`README.md:9`、`docs/engine-integration.md:5`、`docs/engine-integration.md:6-7`、`scripts/handoff.sh:6-8`（脚本注释明确「engine 内部算法不可见，换算法本脚本无需改动」）。

**追问预判**：
- 「DSSM 的负采样怎么做的？」「DeepFM 的 FM 二阶项维度？」→ 这些属于 engine 仓库细节，**在 platform 侧答不上来就不猜**，说明「这需要看 engine 仓库的实现」。
- 「MMR 的 λ 取多少？」→ 同上，不要编造超参。

### Q18. 那 platform 侧的工程贡献是什么？

**问法**：「算法不是你做的，你做的是什么？」

**照说版**：
1. platform 做的是**把离线模型接到真实产品链路上所需的全部工程**：数据契约、ID 对齐、落库幂等、降级兜底、前端消费。
2. 具体包括：快照导出与 sha256 校验、按 `source_url` 判重的通用落库内核（B 站与第三方数据集复用同一段 `ImportItems`）、导入时按真实存在性过滤、缓存命中/未命中两条读路径。
3. 还包括数据来源的工程化：配置驱动的字段映射、三步走（preview / dry-run / 真导入）的导入命令、分类 find-or-create、按字符截断防整批失败。
4. 以及合规约束下的在线采集：30s 超时、风控不重试、失败降级不阻塞页面。

**依据**：`backend/internal/service/crawl_import.go:15-23,154-236`、`backend/internal/service/recommendation_import.go:46-128`、`backend/internal/service/dataset_source.go`（字段映射与归一化）、`backend/internal/service/bilibili_online.go:182-201`、`docs/design.md:385`（决策 #31）。

### Q19. 训练数据是什么？质量如何？

**问法（高风险）**：「推荐模型是用什么数据训的？」

**照说版**：
1. 训练数据来自快照的 `behaviors.csv` 与 `ratings.csv`，即平台库里的 `user_behaviors`（view/click/favorite）与 1-5 分评分。
2. 必须说清当前库的真实情况：**今天已按要求清除全部模拟数据**（模拟资源/用户/分类及其派生行为与评分），
   现在库里是 **170 条真实 B 站资源、4 个真实账号、37 条真实行为、3 条真实评分**。
   engine 的 `dataset/sim` 模拟轨道仍保留在仓库（`demo_seed` 可复现），但**当前库没有启用**，别拿它当现状讲。
3. 前端真实产生的行为**只有 `view`**：进资源详情页自动上报一次 `view`（`ResourceDetailPage.vue:53`），`click` 和 `favorite` 在真实前端**没有上报点**，只在 MSW mock 与数据类型定义里存在。
4. 所以我不会说「我们有丰富的真实用户行为数据」——真实数据量很小，这也是不做在线推理的原因之一。

**依据**：`frontend/src/pages/resource/ResourceDetailPage.vue:47-56`（只上报 view）、`frontend/src/api/behavior.ts:10-11`、`frontend/src/mocks/db.ts:266-267`（click/favorite 只出现在 mock 种子）、`backend/cmd/demo_seed/main.go:21-29,95-129`、`docs/design.md:385`（决策 #31 提到 sim 轨道仅用于演示与回归）、`docs/engine-integration.md:68`。

**追问预判**：
- 「为什么不做点击上报？」→ 前端确实没有埋点击/收藏行为，这是个明显的缺口，可以直接补（后端 `POST /resources/:id/behaviors` 已支持这三种 action）。
- 「曝光呢？」→ **完全没有曝光打点**，全仓无 impression/exposure 相关代码，所以连 CTR 的分母都拿不到。见 Q20。

### Q20. 推荐效果怎么证明？有没有离线指标或对比实验？

**问法（高风险，必须诚实）**：「你的推荐比热门列表好多少？有离线指标吗？做过 AB 吗？」

**照说版**：
1. 明确回答：**目前没有任何量化证据**。platform 侧没有离线评测、没有 AB 实验、没有线上指标监控设施。
2. 具体缺三样：一是没有曝光打点（拿不到 CTR 的分母）；二是没有切分评测脚本或 NDCG/Recall@K 一类离线评估（这属于 engine 仓库的职责，platform 侧也没有接口去触发或回写）；三是没有分流能力（`GET /recommendations` 对所有用户是同一条路径，没有实验分组概念）。
3. 现在能证明的只是**工程闭环成立**：快照导出 → engine 出结果 → 导入 → 首页顺序按 engine 结果变化，链路是可复现、可验证的。
4. 所以我不会讲任何「CTR 提升 X%」「AUC 达到 Y」这类数字——没有依据的数字在这个场合说出来就是硬伤。
5. 要补的话优先级是：先做曝光与点击打点（这是所有指标的前提），再在 engine 侧加离线评测，最后才谈线上 AB。

**依据**：全仓 grep `impression|exposure|曝光|metric|AUC|CTR` **零命中**；`backend/internal/router/router.go:90`（推荐接口无分流）；`frontend/src/pages/resource/ResourceDetailPage.vue:53`（唯一真实行为上报点）；`docs/engine-integration.md:81-86`（「曝光打点与效果指标」被列为**演进方向**而非已实现）。

**追问预判**：
- 「那你怎么知道有效果？」→ 只能答「工程链路验证过，效果未验证」，并把上面三条缺口讲清楚。**不要临场编一个提升比例。**
- 「如果给你两周补，先补哪个？」→ 先补曝光+点击打点，因为它同时是离线评测和线上 AB 的数据前提。

---

## 6. 数据来源与合规

### Q21. 内容是从哪来的？为什么混在一起？

**问法**：「平台上的资源都是你录入的吗？」

**照说版**：
1. 三条来源：手工录入 / `demo_seed` 播种的模拟数据 / 外部采集导入。
2. 外部导入有两条子链路：B 站公开元数据（离线批量 + 在线实时）与第三方数据集离线导入。
3. 它们都以 `type=video` / `type=course` 的**普通资源**身份混进同一张 `resources` 表，不新增专区、不改推荐链路。
4. 这样推荐侧不需要感知来源，代价是没法按来源做差异化策略（比如给视频和课程不同的排序），当前也不打算做。

**依据**：`docs/design.md:382-385`（决策 #28/#29/#31）、`docs/bilibili-import.md:4`（混入现有列表）、`docs/dataset-import.md:237`（`resource_type` 不影响推荐链路）。

### Q22. B 站数据采集的合规性怎么保证？可持续吗？

**问法（高风险）**：「爬 B 站数据合规吗？被封了怎么办？」

**照说版**：
1. 合规边界写进了代码和文档，分四点：**只取公开元数据**（标题/简介/封面/UP 主名/播放量）与公开热门评论 top N，不下载视频内容、不采集评论中的个人信息、不绕过登录墙。
2. **不伪造设备指纹**：`buvid3` 是直接 GET 首页由 B 站正常下发，与浏览器首次访问行为一致；不使用代理池。
3. **不实现任何风控绕过**：命中 `-352` / `-412` 直接报错退出、不重试；请求间强制 1.5–3.0 秒随机间隔。
4. 可持续性要诚实说：**它不可持续**。UP 主投稿入口已经被风控拦死（`-352`），需要自带 cookie 且仍可能被拦；页面展示不下载内容只做回链，所以 B 站若失效，平台只是少一个内容来源，不影响推荐链路。
5. 另外我也试过慕课网在线爬取，**验证为不可行**：腾讯 EdgeOne 对非浏览器客户端下发混淆 JS 挑战页，同 IP 同 UA 下 curl 拿真 JSON、Python 只拿挑战页，说明拦在 TLS 指纹层；打通只能解 JS 挑战或伪造指纹，**违反我们的合规红线，所以主动放弃**，改为导入用户自行下载的本地数据集。

**依据**：`docs/bilibili-import.md:166-172`（合规边界）、`backend/crawler/bili_client.py:1-11,36-37,96-121,235-241`（buvid3 来源、风控码不重试、不实现绕过）、`backend/crawler/README.md:157-167`、`docs/bilibili-import.md:48-49`、`:174-181`（已知限制）、`docs/dataset-import.md:12-27`（慕课网验证不可行 + 合规红线）。

**追问预判**：
- 「采集频率多少？会打爆对面吗？」→ 单关键词上限 = `search_limit 20 × search_max_pages 10 = 200` 条（`backend/configs/config.yaml:42-45`），每请求间隔 1.5–3.0 秒。
- 「评论也爬，算不算个人信息？」→ 只存公开昵称+正文+点赞+楼层+时间，不抓评论者 uid、不抓主页、不做跨站关联（`docs/bilibili-import.md:168`、`backend/internal/model/comment.go`）。

### Q23. 为什么要额外做数据集导入这条链路？

**问法**：「B 站已经能用了，为什么还做第三方数据集导入？」

**照说版**：
1. 因为在线爬取有硬上限：一个是风控（随时可能失效），一个是量级（采不出课程级别的结构化数据）。
2. 数据集导入是**纯离线**的：只解析用户合法获取的本地文件（json/jsonl/csv），Go 侧直接入库，不引入任何新的 Python 代码，也就避开了改动无测试覆盖的采集脚本的风险。
3. 字段映射外置到 YAML（`datasets.<名称>`），换数据集只改配置、不动代码；未知字段名在**服务启动时**就报错，而不是等导入跑完才发现整批被跳过。
4. 落库复用与 B 站**完全相同**的 `ImportItems` 内核，判重/分类/截断语义一致。

**依据**：`docs/dataset-import.md:11-30,50-95`、`backend/cmd/import_dataset/main.go:23-80`、`backend/internal/service/crawl_import.go:154-236`、`backend/internal/config/config.go:65-90,198-207`（加载期校验）、`docs/design.md:385`（决策 #31/#32）。

**追问预判**：「数据集合规吗？」→ 只导入用户合法获取、公开可用的数据集，遵守其许可条款；产物落在被 `.gitignore` 忽略的 `backend/data/`，不入库（`docs/dataset-import.md:219-226`）。

---

## 7. 工程质量（测试 / 并发 / 超时 / 缓存失效）

### Q24. 测试覆盖到什么程度？

**问法**：「你们怎么保证质量？测试覆盖率多少？」

**照说版**：
1. 后端 `go test ./... && go vet ./...` 当前全部通过（本机实测 exit 0）。测试集中在 service 层：推荐读写与兜底、导入过滤与跳过、采集落库判重与截断、数据集字段映射归一化、评论抓取与「抓到 0 条也标记」等。
2. 前端按仓库定义有 25 个测试文件、约 156 条用例，覆盖搜索无限滚动（含代次防串页、判重不重复渲染）、评论、评分、管理页、路由守卫、超时约定等。
3. **要主动坦白两个缺口**：一是后端 `handler/` 与 `repository/` 目录**没有任何测试文件**，`docs/design.md:335` 写的「httptest API 集成测试」并未落地；二是没有连真实 MySQL/Redis 的集成测试，service 测试全用 fake 仓库。
4. 所以我不报覆盖率数字（没跑过覆盖率），也不说「有完整集成测试」。

**依据**：`backend/` 下 107 个 `func Test`，其中 `handler/`、`repository/`、`router/` 目录无 `_test.go`；`docs/design.md:332-338`（测试策略含未落地的集成测试）；`frontend/src/**/__tests__/` 与 `*.test.ts` 共 25 个文件。

**追问预判**：
- 「为什么不做 handler 测试？」→ 直接承认是缺口，最有价值的第一批就是推荐接口与导入接口的 httptest 集成测试。
- 「那你怎么知道接口是对的？」→ 靠手动联调 + 前端 mock handler 对齐契约（`frontend/src/mocks/handlers.ts`），不是自动化保证。

### Q25. 缓存失效与边界场景怎么处理的？

**问法**：「缓存里的资源被删了、缓存 JSON 坏了、并发请求同一个用户，分别会怎样？」

**照说版**：
1. **资源被删**：`orderResources` 只保留查得到的 ID，已删的自动跳过，不报错。
2. **缓存 JSON 损坏**：解析失败视为未命中，重新走兜底并覆盖写缓存（有单测覆盖）。
3. **并发首次请求**：`Replace` 是先删后插两步、不带事务。两个请求同时进来可能都判定未命中、各写一次；`user_id` 唯一索引会让其中一个失败。实际概率低但存在，**属于已知未彻底解决的问题**。
4. **全部 ID 失效**：返回空列表且**不触发兜底**（兜底前置条件是「没有缓存行」），这时首页会是空的。这是一个真实边界，我目前没有为它写守卫，也没有对应的单测。

**依据**：`backend/internal/service/recommendation.go:51-66,104-117`、`backend/internal/service/recommendation_test.go:110,188`、`backend/internal/repository/recommendation.go:39-53`、`backend/internal/model/recommendation.go:6`。

**追问预判**：「怎么修？」→ 在返回前加一句「`ordered` 为空则走兜底」即可；并发问题可以用 `ON DUPLICATE KEY UPDATE` 或 `INSERT … ON DUPLICATE`/事务包住删除+插入。两者都是小改动。

### Q26. 超时、风控与失败降级是怎么约定的？

**问法**：「实时爬 B 站卡住了怎么办？前端会不会一直转圈？」

**照说版**：
1. 后端对 Python 子进程用 `exec.CommandContext` 加 **30 秒硬超时**，stdout 约定只输出单行 UTF-8 JSON，日志和错误走 stderr（这也规避了 Windows 子进程 GBK 乱码）。
2. 前端对会触发爬取的接口用 **35 秒超时**（`CRAWL_TIMEOUT`），**故意比后端宽 5 秒**——否则后端已经爬完落库了，浏览器却先超时，前端还会把列表错误标记成「已耗尽」；这条约定有单测锁住。
3. 搜索爬取失败：后端只 `slog.Warn` 后返回空列表，接口仍是 200，前端展示已加载的内容并标记到底，不阻塞页面。
4. 评论爬取失败：不标记 `fetched`（下次访问可重试），返回空列表，前端展示「评论暂不可用或暂无评论」空态。抓到 0 条**也要标记**，否则无评论的视频每次访问都会重启一个 Python 进程。

**依据**：`backend/internal/service/bilibili_online.go:18-19,182-201`、`frontend/src/api/client.ts:20-24`、`frontend/src/api/resource.ts:15-21`、`frontend/src/api/comment.ts:8-11`、`frontend/src/api/__tests__/crawlTimeout.test.ts:29-51`、`backend/internal/service/resource.go:72-79`、`backend/internal/service/comment.go:61-79`。

**追问预判**：
- 「并发很多用户同时搜索会怎样？」→ 每个请求各起一个 Python 进程，**没有并发闸门/限流**，这是真实的稳健性缺口（见 Q27）。
- 「为什么用子进程而不是常驻服务？」→ 保持采集逻辑与 Go 服务隔离、复用已验证的 Python 代码；代价是启动开销与进程管理成本。

### Q27. 并发与限流呢？

**问法**：「如果很多人同时滚到底触发爬取？」

**照说版**：
1. 现状是**没有限流、没有并发闸门**：每个 `online_page` 请求或评论请求都会独立 `exec` 一个 Python 进程。
2. 中间件只有 `gin.Logger` 和 `gin.Recovery`，没有 rate limiter；数据库连接池配了 `max_idle_conns=10 / max_open_conns=100`。
3. 风险是明确的：突发流量会拉起大量 Python 子进程（每个还可能带 1.5–3.0 秒的节流等待），既压本机也更容易触发 B 站风控。
4. 缓解措施目前只有业务侧上界（单关键词最多 200 条、单次 30s 超时、评论抓取状态表避免重复抓），**没有做进程池或信号量**。这是我要主动承认的短板。

**依据**：`backend/internal/router/router.go:60-62`（无 rate limit 中间件）、`backend/internal/service/bilibili_online.go:87-105,149-180`（每次调用都起进程）、`backend/internal/database/mysql.go:28-29`（连接池）、`backend/configs/config.yaml:44-45`（页数上限）、`backend/internal/service/comment.go:47-53`（抓取状态防重复）。

**追问预判**：「怎么改？」→ 用一个带缓冲 channel 或 `golang.org/x/sync/semaphore` 限制并发 Python 调用数，超限时直接降级为「本地结果」，顺手把搜索结果按 keyword 短 TTL 缓存。

### Q28. 前端几个容易踩的坑你们怎么处理的？

**问法**：「无限滚动 + 后端爬取，前端怎么保证不出乱序/重复？」

**照说版**：
1. **代次（requestGeneration）机制**：每次重置搜索自增代次，旧请求回来时代次不符就整段丢弃，既不追加进新列表也不推进页码——因为爬取可能长达数十秒，用户中途改关键词是常态。
2. **按 ID 去重**：在线爬取会把判重命中的本地已有行一并回传，直接 concat 会重复 id 触发 v-for 重复 key，所以追加前用 `Set` 过滤。
3. **页码只在成功时推进**：失败时保持原页码，避免下次滚动跳过一页结果。
4. **哨兵重观察**：`IntersectionObserver` 只在交叉状态变化时回调，内容不足一屏时不会再触发，所以每次加载结束都重新 observe 一次，让无限滚动能自启动。
5. **爬取失败的降级**：直接标记到底，不阻塞已加载的内容。

**依据**：`frontend/src/pages/search/SearchPage.vue:42-46,82-90,92-115,174-192`、`frontend/src/pages/search/__tests__/SearchPage.test.ts:114-288`（含「旧关键词在飞行的爬取结果不会追加进新列表」「判重命中不重复渲染」）。

---

## 8. 部署与规模

### Q29. 怎么部署？现在能一键起吗？

**问法**：「这套东西怎么跑起来？」

**照说版**：
1. 后端 `go run ./cmd/server`（需 MySQL 与 Redis），前端 `pnpm run dev`（dev 默认 `VITE_MOCK=1`，可以完全脱离后端只靠 MSW 跑通界面）。
2. 批量侧有 5 个命令入口：`export_snapshot`、`import_bilibili`、`import_dataset`、`demo_seed`、`server`，以及把「导出→训练→推理→导入」串成一条的 `scripts/handoff.sh`。
3. **要坦白的一点**：设计文档里写的 Docker 多阶段构建与 `docker-compose.yml` **在仓库里并不存在**，属于规划未落地；现在跑起来依赖本机已装好 MySQL/Redis/Python 环境。
4. `handoff.sh` 的前提是后端已启动（第 ⑥ 步走 HTTP 导入需要管理员账号），engine 默认取 `../edurec-engine`。

**依据**：`README.md:11-19,33-40`、`scripts/handoff.sh:10-38,57-103`、`docs/design.md:308-328`（Docker 规划）、仓库根目录无 Dockerfile / docker-compose.yml。

**追问预判**：「在线搜索依赖什么？」→ 依赖运行环境里有 Python 3 与 `backend/crawler/requirements.txt` 的依赖（requests），并且 `bilibili.crawler_dir` 指向 `backend/crawler`；`python_path` 配错或没装 Python 时在线搜索会失败并降级为本地结果。

### Q30. 规模上能撑多少？瓶颈在哪？

**问法**：「如果资源涨到几百万、用户几万，哪里先崩？」

**照说版**：
1. **推荐读侧最容易扩**：一次 `FindByUserID`（`user_id` 唯一索引）+ 一次 `IN` 查询，与用户数线性、与资源总数无关。
2. 三个明确的瓶颈：
   - `export_snapshot` 全量导出 + engine 全量推理，随用户数线性变重，需要改成按批/增量；
   - 搜索是 `title LIKE '%kw%' OR description LIKE '%kw%'`，**前置通配符无法走索引**，必然全表扫；
   - `resources.tags` 的筛选用 `JSON_CONTAINS`，同样没法有效索引。
3. 导入侧目前是单进程串行、分批 `INSERT IGNORE`/`Create`，`handoff.sh` 一轮全量；规模上来要换成增量与并行。
4. 这些都是**推导出来的瓶颈，没有做过压测**——我不会给 QPS 数字。

**依据**：`backend/internal/repository/resource.go:69-113`（LIKE 与 JSON_CONTAINS）、`backend/internal/service/recommendation.go:44-60`、`backend/cmd/export_snapshot/main.go:47-137`（全量导出）、`backend/internal/service/recommendation_import.go:60-126`（单进程逐用户写）。

**追问预判**：「换成什么？」→ 搜索换全文索引（MySQL FULLTEXT 或外部检索引擎），快照改增量+游标，推荐读侧可加 Redis 短 TTL 缓存——但**目前都没做**。

---

## 9. 后续计划

### Q31. 如果继续做，下一步做什么？

**问法**：「这个项目接下来你打算怎么推进？」

**照说版**（按优先级，全部是「未做」）：
1. **效果可度量**：补曝光打点与点击上报（前端现在只上报 view），这是离线评测与线上 AB 的共同前提。
2. **缓存健壮性**：全量 ID 失效时回退兜底、`Replace` 改原子写（消除并发窗口）。
3. **在线采集治理**：给实时爬取加并发信号量与按关键词的短 TTL 缓存，避免突发流量拉起大量子进程。
4. **工程底座补齐**：把文档里写了但没落地的部分补上——handler 层 httptest 集成测试、`source_url` 唯一索引 migration、Docker 化部署。
5. **实时化（可选）**：engine 加 HTTP serving + ANN，缓存未命中时实时调用，platform 侧配套超时与熔断。

**依据**：`docs/engine-integration.md:81-86`（演进方向原文）、`backend/internal/model/resource.go:34`（source_url 无唯一索引）、`docs/bilibili-import.md:174-181`（已知限制）。

### Q32. 这个跟直接用现成推荐系统有什么区别？

**问法（高风险）**：「现在有那么多开箱即用的推荐系统/推荐服务，你这个和直接用它们有什么区别？」

**照说版**：
1. 我不主张算法上的新颖性——召回/精排/重排这套组合是成熟做法，engine 的价值不在发明新模型。
2. 区别在于**这是一个从数据来源到前端消费都自己打通的完整闭环**：内容怎么合规地来、ID 怎么对齐、导入怎么幂等、缓存空了怎么兜底、爬取超时怎么降级，这些是拿开源推荐库直接接上去时不会替你解决的问题。
3. 第二个区别是**契约设计**：两仓只通过「CSV 快照进、排名 ID 列表出」交互，engine 换算法、platform 换存储都不影响对方。接入成本被压到一个很窄的接口面上。
4. 第三个区别是**边界诚实**：我们主动放弃了不合规的采集路径（慕课网的 TLS 指纹拦截就没绕），而不是为了数据把红线踩过去。
5. 所以如果问题问的是「算法有没有创新」，我的答案是**没有，不宣称**；如果问的是「工程闭环和边界控制做得是否完整」，那是我愿意被检验的部分。

**依据**：`scripts/handoff.sh:6-8`（契约窄接口）、`docs/design.md:385`（决策 #31 慕课网放弃理由）、`docs/engine-integration.md:64-68`（ID 与覆盖口径）、`docs/bilibili-import.md:166-172`（合规边界）。

**追问预判**：
- 「那你的科研贡献是什么？」→ 不要硬凑。可以说这是一个**工程验证平台**：为 engine 提供真实可跑的应用场景与可复现的数据交接契约，贡献在「让离线模型能被真实链路验证」，而不是新算法。
- 「跟 RecBole / 推荐 SaaS 比呢？」→ 它们是算法/服务提供方，本项目是其**落地宿主**：补的是数据来源、一致性、降级、前端消费这一段，两者是互补关系而非替代。

---

## 10. 答不上来 / 不夸大（现场守则）

> 这一节比前面任何一节都重要：**说错一句数字，整场可信度就没了**。

### 10.1 明确答不上来 / 未验证的事

| 事项 | 现场怎么说 |
|---|---|
| 离线指标（Recall@K、NDCG、AUC 等） | 「**没有做过**，platform 侧没有评测设施，我不给数字。」 |
| AB 实验、分流 | 「**没有**。推荐接口对所有用户是同一条路径，没有实验分组概念。」 |
| 线上 CTR / 转化提升 | 「**没有曝光打点，连分母都没有**，不存在这个数字。」 |
| engine 超参、网络细节、训练配置 | 「这属于 engine 仓库，我在 platform 侧不复述具体实现。」 |
| 压测 QPS / 承载规模 | 「**没做过压测**。只能从代码结构推导瓶颈（LIKE 全表扫、全量导出），不给 QPS。」 |
| 测试覆盖率 | 「没跑过覆盖率，不报数字。」 |
| 线上稳定性 / 已上线运行 | 「这是课题工程，**没有生产部署**，Docker 部分也还没落地。」 |
| 前端测试全套是否通过 | 「本机沙箱里 vitest 起不来（`spawn EPERM`），我只静态确认了用例存在；后端 `go test ./...` 是实测通过的。」 |

### 10.2 绝对不要说的话

1. ❌「我们做过 AB 实验」 / 「线上 CTR 提升了 X%」 —— **完全不存在**，仓库无任何分流与指标代码。
2. ❌「我们有大量真实用户行为数据」 —— 真实前端**只上报 `view`**，`click`/`favorite` 只存在于 mock 与类型定义里。
   当前库虽有 37 条真实行为，但只来自 **2 个真实账号**（BTXL、biliverify），量级远谈不上"大量"；
   且**绝不能**把已清除的 `demo_seed` 模拟数据（`demo1` 那批）当真实用户数据讲。
3. ❌「模型已经在线服务化了」/「请求时实时推理」 —— platform 侧零模型代码，权重只在 engine。
4. ❌ 把 B 站采集说成「稳定可持续的数据源」 —— UP 主入口已被风控拦死，随时可能整体失效；已明确列为不可持续。
5. ❌ 把慕课网说成「爬到了」 —— 恰恰相反，**验证为不可行后主动放弃**，改走本地数据集导入；这是合规亮点，不要讲反。
6. ❌ 说「有唯一索引保证不重复」 —— `resources.source_url` **没有唯一索引**，只有普通索引，重复风险靠单进程串行规避。
7. ❌ 说「用了 golang-migrate 管理迁移」/「有 docker-compose 一键部署」 —— 文档写了，**仓库里没有**，靠 `AutoMigrate` 和本机环境。
8. ❌ 说「兜底推荐是热门推荐」而不加限定 —— 兜底按 `avg_rating DESC`，而站内评分稀疏，实际接近按 ID 倒序。
9. ❌ 替 engine 报任何数字（模型规模、训练轮数、指标）—— 说「以 engine 仓库为准」。
10. ❌ 说「评论/评分是真实用户评价」 —— 站内评分是演示数据，B 站评论是采集来的公开评论，两者在库和语义上都是分开的。

### 10.3 主动坦白的短板清单（老师问「你觉得哪里做得不够」时直接念）

1. **效果完全未度量**：无曝光打点、无离线评测、无 AB、无线上监控 —— 这是最大短板。
2. **后端测试层次不全**：`handler/`、`repository/` 无测试，`docs/design.md` 承诺的 httptest 集成测试未落地；无连真实 MySQL/Redis 的集成测试。
3. **并发防护缺失**：实时爬取无并发闸门与限流，突发流量会拉起大量 Python 子进程。
4. **缓存边界未兜满**：全部 ID 失效时返回空列表且不回退兜底；`Replace` 先删后插非原子，存在并发写窗口。
5. **数据来源判决性约束靠流程而非代码**：sim 数据禁止导入真实平台只有文档约束，没有代码闸门；`source_url` 缺唯一索引。
6. **搜索可扩展性差**：`LIKE '%kw%'` + `JSON_CONTAINS` 都无法走索引，资源量上来必然全表扫。
7. **无时效机制**：推荐结果没有 TTL 与新鲜度判定，`updated_at` 也不在前端展示，用户无从感知结果多旧。
8. **部署与文档不一致**：设计文档中的 Docker 编排与 golang-migrate 均未落地，文档需要标注为规划。

---

## 附录 A. 现场可能被追问的机制细节（速查）

| 机制 | 细节 | 出处 |
|---|---|---|
| 推荐 limit | 默认 12（前端首页请求值）；后端默认 20、上限 50，越界会被夹取 | `frontend/src/pages/home/index.vue:14`、`backend/internal/service/recommendation.go:34-39` |
| 缓存写入粒度 | 按用户整行替换；`user_id` 唯一索引 | `backend/internal/repository/recommendation.go:39-53` |
| Access/Refresh | Access 15 分钟；Refresh 7 天存 Redis，刷新时轮换 | `backend/configs/config.yaml:22-26`、`docs/design.md:301-305` |
| 在线搜索触发条件 | 本地翻完（`resources.length >= total`）+ **纯关键词**（无分类/类型/标签）+ `has_more` 为真 | `frontend/src/pages/search/SearchPage.vue:29-31,142-155`、`backend/internal/service/resource.go:70-79` |
| B 站翻页上限 | `search_limit 20 × search_max_pages 10 = 200` 条/关键词 | `backend/configs/config.yaml:41-45` |
| `has_more` 判定 | 返回条数 ≥ searchLimit 且 page < maxPages | `backend/internal/service/bilibili_online.go:143-145` |
| 评论抓取上限 | 默认 20 条，仓库层再夹到最多 100 | `backend/configs/config.yaml:46`、`backend/internal/repository/comment.go:38-44` |
| 评论不重复抓 | 抓到 0 条也写 `comment_fetch_state`；抓失败不写以便重试 | `backend/internal/service/comment.go:45-79` |
| 字段截断 | 按**字符**（rune）而非字节：title 256 / cover 512 / author 128 / source_url 512 | `backend/internal/service/crawl_import.go:59-65,340-346` |
| 分类处理 | 按名称 find-or-create；单次导入内同名只查一次；写错分类名会静默新建 | `backend/internal/service/crawl_import.go:238-271`、`docs/bilibili-import.md:181` |
| 快照内容 | users(仅 id) / resources / categories / behaviors / ratings 五个 CSV + meta.json(含 sha256) | `backend/cmd/export_snapshot/main.go:47-162` |
| 一键刷新 | `bash scripts/handoff.sh`（导出→训练→推理→导入），`--infer-only` 可跳过训练 | `scripts/handoff.sh:1-103` |
| 配置校验时机 | 数据集字段映射写错 → **服务启动就失败**，不是导入时才报 | `backend/internal/config/config.go:198-207` |

## 附录 B. 库内数据事实（说之前先确认口径）

**老师问「现在库里有多少数据」时的标准答法**：

> 「数据多少取决于跑过哪条链路，且**今天刚清过一次库**，我给两组口径。」

| 口径 | resources | users | recommendations | 说明 |
|---|---|---|---|---|
| **演示库（2026-09-19 12:1x 清除模拟数据后，权威）** | **170 条，全部 B 站 video** | **4 条**<br>`BTXL`(2) `biliverify`(3) `demo_admin`(2000) `demo_fresh`(2001) | **3 行**（user_id 2/3/2000；2001 无行→走兜底） | 行为 37 条、评分 3 条、分类 2 个（人工智能 / B站视频）、评论缓存 30 条；`avg_rating` 153/170 有值 |
| 历史快照 `data/snapshots/20260916_181925/meta.json` | 139 | 2 | — | 导出时刻的真实库：behaviors 14 条、ratings 0 条、categories 2 个 |
| （已废弃）清除前跑过 `demo_seed` 的开发库 | 500（含 article/course 模拟件） | 2001 | 2001 | 掺入 engine `dataset/sim` 模拟数据、行为约 10 万条。**这批已按要求清除，不要再引用** |

**要点**：
1. **当前演示库是干净的纯真实数据**：170 条真实 B 站资源、4 个真实账号、37 条真实行为（view/click/favorite）。
   说「真实用户行为数据」时注意：**这 37 条来自 2 个真实账号**（BTXL、biliverify），量很小，别夸大成"大量真实行为"。
2. 清除模拟数据后，首页推荐**仍非空**：3 个账号读缓存行（各 20 条真实资源），`demo_fresh` 无行走 `avg_rating DESC` 兜底。
   所以「引擎没覆盖到我」的降级表现是**一个非空的真实热门列表**，不是空白页。
3. `admins` 表现在**有 1 行**（`demo_admin`），导入接口可用 —— 与"admins 为空"的旧口径不同。
4. 若被问到"模拟数据去哪了"：如实说「按要求清除了，包括模拟资源/用户/分类及其派生行为与评分；平台侧现在只有 B 站采集的真实内容」，
   并说明 `demo_seed` 仍保留在仓库里，需要时可用于回归演示，但**当前库没有启用**。

**依据**：本机 `edurec` 库实测查询（2026-09-19）；`backend/data/snapshots/20260919_121613/meta.json`（清除后快照：resources 170 / users 4 / behaviors 37 / ratings 3 / categories 2）；`backend/cmd/demo_seed/main.go:21-29`；`backend/internal/middleware/admin.go:20-30`（管理员判定依赖 `admins` 表）。
