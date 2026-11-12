# 双仓架构评估与重构记录

> 评估对象：`edurec-platform`（Go + Vue）与 `edurec-engine`（Python + PyTorch）
> 评估日期：2026-11-12
> 本轮性质：**评估 + 设计 + 已实施的重构**（未实施项明确标为路线图）

---

## 0. 结论摘要

整体架构是健康的：双仓边界清晰、交接契约可执行、反馈闭环（行为 → 快照 → 训练 → 导入 → 曝光/点击）已经成立。
本轮不推翻现有架构，而是修掉三类问题：

1. **分层泄漏**：HTTP Handler 直接持有 repository（推荐运行记录/统计），绕过 service。
2. **规模化缺陷**：推荐导入逐用户「删 + 插」、存在性查询一次性 `IN (...)`，在真实数据量下会慢、会撞 SQL 上限、且中途失败留下半批状态。
3. **静默正确性缺陷**（engine）：质量融合被重复归一化、编码器缓存不随语料失效——默认参数下不暴露，换权重/换语料时才出错。

本轮已落地 6 项重构（见 §4），全部测试通过（见 §6）。剩余建议按 P1–P3 排期（见 §5）。

---

## 1. 评估范围与基线

| 仓库 | 技术栈 | 基线测试 |
|---|---|---|
| `edurec-platform/backend` | Go 1.26 · Gin · GORM · MySQL · Redis | `go test ./...` 全绿，`go vet ./...` 无输出 |
| `edurec-platform/frontend` | Vue 3 · Vite · Pinia · MSW | `vitest run`：39 个文件 / 168 个用例全绿 |
| `edurec-engine` | Python 3.12 · PyTorch · numpy | `pytest`：168 个用例全绿（重构后 181 个） |

评估方法：读架构文档与契约 → 逐层核对实现（handler/service/repository、data/features/models/pipeline）
→ 跑基线测试 → 用「故障场景」反推问题（不是只看代码风格）。

---

## 2. 目标架构（设计）

### 2.1 双仓边界

```
┌──────────────────────────────┐         ┌──────────────────────────────┐
│ edurec-platform              │  文件   │ edurec-engine                │
│ 在线服务 + 数据治理           │ ◄─────► │ 离线训练 + 批量推理           │
│ Go / Vue / MySQL / Redis     │ 契约 v1 │ Python / PyTorch             │
└──────────────────────────────┘         └──────────────────────────────┘
        │  快照 CSV（方向 A）                     ▲
        └────────────────────────────────────────┘
             推荐 ID 列表（方向 B，旁挂信封）
```

保持双仓的理由与 `docs/design.md` #37 一致：engine 是可独立引用/复现的模型工件，
platform 是 Web 应用；两者工具链与数据体量差异大。**边界由 `contracts/v1` 固化**，
换模型不需要改 platform 代码。

### 2.2 platform 分层（本轮强化）

```
cmd/                 组装根（composition root）：只做依赖装配与 CLI 参数
  └─ handler/        HTTP：参数校验、序列化、调用 service；不碰 repository
       └─ service/   用例与业务口径：缓存策略、导入事务、CTR、权限后的规则
            └─ repository/  持久化：GORM、SQL、事务边界
                 └─ model/   数据模型
```

单向依赖：`handler → service → repository → model`。本轮把推荐运营接口从
`handler → repository` 拉回 `handler → service`，并把 CTR 口径收进 service。

### 2.3 推荐主链路（目标形态）

```
业务库 (MySQL，唯一事实来源)
  │ cmd/export_snapshot（快照 = 给模型的 read model，带 contract_version + sha256）
  ▼
engine：训练 → 全量推理 → recommendations.json（+ meta 信封）
  │ contracts/v1 校验（出口自检，违规拒绝交付）
  ▼
platform：POST /admin/recommendations/import
  │ ① 分批存在性校验（用户/资源，每批 ≤1000）
  │ ② 分批事务 UPSERT（每批 ≤500 用户，批内原子）
  ▼
recommendations 缓存表 → GET /recommendations（按数组顺序返回）
  └─ 未命中：兴趣分类 → 评分热门兜底，并写缓存
```

设计原则：

- **契约优先**：主文件形状是硬边界；元信息一律放旁挂信封。
- **读模型显式化**：快照、BundleAccess、推荐缓存都是派生读模型，不反向污染事实表。
- **失败要响**：契约校验失败、语料指纹不一致、配置非法都在入口报错，不静默降级。
- **批处理有边界**：所有外部输入的 ID 列表都要分块，避免撞数据库参数上限。

---

## 3. 发现的问题（按严重度）

| # | 位置 | 问题 | 影响 | 本轮处置 |
|---|---|---|---|---|
| P1 | `handler/recommendation.go` | Handler 直接持有 `RecommendationRunRepository` / `StatsRepository`，并自行计算 CTR | 分层被绕过；口径分散；运营规则变化会改到 HTTP 层 | ✅ 抽出 `RecommendationAdminService`，Handler 只调 service |
| P2 | `service/recommendation_import.go` | 逐用户 `Delete + Create` | N 次往返；中途失败留下「半批新、半批旧」 | ✅ `ReplaceMany` + 分批事务 UPSERT |
| P3 | 同上 | 用户/资源 ID 一次性 `FindByIDs`（无分块） | 真实规模下 `IN (...)` 撞 `max_allowed_packet` / 占位符上限 | ✅ 每批 ≤1000 分块查询 |
| P4 | 同上 | 导入写入无事务边界 | 失败不可回滚，缓存状态不一致 | ✅ 每批 ≤500 用户，批内事务原子 |
| P5 | `cmd/export_snapshot` | 用户/资源全量读入内存、行为/评分行累积在内存后才写 CSV | 数据量上来后导出进程 OOM | ⏳ 路线图 P1（改为流式写出） |
| P6 | `handler/resource.go`、`user_behavior.go`、`admin.go` | Handler 仍直接构造 `repository.*Query` 结构 | 耦合较浅但仍是 DTO 泄漏 | ⏳ 路线图 P2（service 自有权重入参） |
| P7 | `service/recommendation.go` | 平台兜底推荐不排除「用户已交互」资源 | 冷启动/未导入用户可能被推荐已看内容 | ⏳ 路线图 P2（注入行为仓储做过滤） |
| P8 | `docs/design.md` #9 | 建表仍靠 GORM `AutoMigrate`，无版本化迁移 | 生产变更不可审计、不可回滚 | ⏳ 路线图 P2（golang-migrate） |
| P9 | `service/recommendation.go` | 缓存无 TTL / 新鲜度策略；旧结果会一直返回 | 长期不刷新时推荐陈旧 | ⏳ 路线图 P3（读取信封 `generated_at`，超期走兜底并告警） |

> P1–P4 是本轮实际修改；P5–P9 已确认但不在本轮改动范围，避免一次提交过宽。

### 3.2 前端评估

前端整体是「按页面模块 + 按类型公共层」的混合组织，类型集中在 `src/types/index.ts`，
API 统一走 `src/api/client.ts`（JWT 注入、401 单飞刷新、错误归一）。这部分结构合理，
本轮未改动。仍有三个值得排期的点：

| # | 位置 | 观察 | 建议 |
|---|---|---|---|
| F1 | `pages/search/SearchPage.vue`（457 行）、`pages/resource/ResourceDetailPage.vue`（349 行） | 页面同时承担数据获取、滚动加载、筛选、展示 | 抽 `useSearchFeed` / `useResourceDetail` composable，页面只保留模板与交互 |
| F2 | `mocks/handlers.ts`（492 行）、`mocks/db.ts` | MSW 手写响应，容易与后端真实返回漂移 | 用契约 fixture 生成部分 mock；或把 handlers 按资源域拆分 |
| F3 | 测试结构 | 组件/页面测试充分（39 文件），但没有端到端链路 | 增加 1–2 条 Playwright 冒烟：登录 → 首页推荐 → 详情 → 收藏 |

---

## 4. 本轮重构落地

### 4.1 推荐导入：批量化 + 事务化（P2/P3/P4）

文件：

- `backend/internal/repository/recommendation.go`
- `backend/internal/service/recommendation_import.go`

变化：

1. `RecommendationRepository` 增加 `ReplaceMany([]RecommendationUpsert) error`。
   实现用 `clause.OnConflict{Columns: user_id, DoUpdates: ...}` + `CreateInBatches`，
   整批放在 `db.Transaction` 内，语义为「这批要么全部生效，要么全部不生效」。
2. 单用户 `Replace` 改为 `ReplaceMany` 的薄封装，避免两套写入口径。
3. 存在性校验改为 `uniqueUint` + `chunkUint(..., IDLookupChunk=1000)`；
   写入改为 `chunkUpserts(..., ImportWriteChunk=500)`。
4. 常量导出（`IDLookupChunk` / `ImportWriteChunk`），供测试锁定边界。

为什么用 UPSERT 而不是「先删后插」：`recommendations.user_id` 本身有唯一索引，
UPSERT 是一条语句、天然幂等，并且能把批内原子性交给数据库事务；逐用户删插既慢，
又会制造「删除后、插入前」的可见空窗。

### 4.2 分层修复：推荐运营服务（P1）

文件：

- 新增 `backend/internal/service/recommendation_admin.go`
- `backend/internal/handler/recommendation.go`
- `backend/internal/router/router.go`

变化：

- `RecommendationAdminService` 负责运行记录列表与效果统计，CTR 口径只在这里实现。
- Handler 只做参数解析与响应序列化，不再 import `repository`。
- 组装根（router）负责注入，符合 composition root 的职责。

### 4.3 测试

新增/更新：

- `service/recommendation_import_test.go`
  - `TestImportChunksLargeIdLookupsAndWrites`：2500 用户 → 验证查询分块 ≤1000、
    写入批次 ≤500、总写入数正确。
  - `errRecRepo` 改为覆盖 `ReplaceMany`。
- `service/recommendation_admin_test.go`
  - CTR 计算、零曝光、仓储错误映射。
- `service/recommendation_test.go`
  - fake 仓储实现 `ReplaceMany`，保持原有缓存命中/兜底测试语义。

---

## 5. 后续路线图（未在本轮实施）

### P1（数据量与正确性，优先）

1. **`export_snapshot` 流式导出**：用户/资源/行为/评分直接写入 `csv.Writer`，
   只在内存保留计数与 `maxTs`；避免全量 `[][]string` 累积。验收：10 万行为导出内存峰值
   与行数解耦。
2. **缓存新鲜度**：读取旁挂信封落库的 `generated_at`，超过阈值时返回兜底并记录告警。
   验收：可配置 TTL；接口返回 `stale=true` 或 `run_id` 可追溯。

### P2（架构一致性）

3. **service 层自有权重入参**：把 `repository.*Query` 从 handler 中移除，改为
   `service.ListResources(params)` 等；repository DTO 只在 service/repository 间流动。
4. **兜底排除已交互**：`RecommendationService` 注入行为仓储，fallback 时过滤 seen，
   避免「推荐已看过的内容」。
5. **版本化迁移**：引入 `golang-migrate`，把现有 AutoMigrate 的表结构固化为 up/down 脚本；
   验收：空库可迁移、重复迁移幂等。

### P3（可运营性）

6. **契约 CI**：在 `scripts/verify-all.sh` 中固定执行 engine 测试 → 契约 fixture 同步 →
   backend 测试，作为合并门禁。
7. **可观测性**：导入耗时/批次数/跳过率、缓存命中率、兜底率写入 `slog` 结构化日志。

---

## 6. 验证

```bash
# backend
cd edurec-platform/backend
go build ./...
go test ./...
go vet ./...

# frontend（本轮未改前端，确认未受影响）
cd ../frontend
pnpm test --run

# engine
cd ../../edurec-engine
.venv/bin/python -m pytest -q
```

本轮结果：

| 检查 | 结果 |
|---|---|
| `go build ./...` | 通过 |
| `go test ./...` | 全部包通过（含新增导入分块/运营服务测试） |
| `go vet ./...` | 无输出 |
| frontend `vitest run` | 39 文件 / 168 用例通过 |
| engine `pytest` | 181 用例通过 |
