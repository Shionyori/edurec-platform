# edurec-platform

教育资源推荐平台（edurec-engine 的应用场景）。后端 Go + 前端 Vue3。

| 子项目 | 技术栈 |
|---|---|
| backend | Go · Gin · GORM · MySQL · Redis · JWT |
| frontend | Vue 3 · Vite · Element Plus · Pinia · MSW |
| engine（独立仓库） | Python · PyTorch（DSSM 召回 + DeepFM 精排 + MMR 重排） |

## 快速开始

```bash
# 后端（需 MySQL/Redis，配置见 backend/configs/config.yaml）
cd backend && go run ./cmd/server

# 前端（dev 默认 VITE_MOCK=1，走 mock 可脱离后端）
cd frontend && pnpm install && pnpm dev
```

## 推荐闭环（engine 离线批量训练 → 结果落库）

两仓的交叉接口是 platform 的 **MySQL 数据库**——engine 与 platform 不直接调用，数据经数据库衔接：

- 数据下链：`export_snapshot` 将业务表（用户/资源/行为/评分）导出为快照目录 → engine 据此训练 + 全量推理
- 结果上链：engine 产出的推荐列表（平台原始 ID）经导入接口写入 `Recommendation` 缓存表
- 读侧：`GET /api/v1/recommendations` 读缓存表返回；未命中用户按评分降序热门兜底并写缓存（空/缺失缓存不会导致无推荐）

platform 不做模型推理、engine 不做在线服务：首页是否个性化，取决于 `Recommendation` 缓存表是否被 engine 结果填充——
已导入则返回导入列表，从未导入/新用户则走热门兜底。交接文件只是 MySQL 数据的序列化载体，
详见 [docs/engine-integration.md](docs/engine-integration.md)。

```bash
cd backend
# （可选）播种演示数据（管理员 demo_admin/demo123456）
CONFIG_PATH=configs/config.yaml go run ./cmd/demo_seed -with-behaviors
# ① 导出平台数据快照 → engine 据此训练/推理（见 docs/data-handoff.md）
CONFIG_PATH=configs/config.yaml go run ./cmd/export_snapshot   # → data/snapshots/<run_id>/
# ② 管理员导入 engine 推荐结果 → 个性化生效（文件须先置于 data/recommendations.json）
```

engine 与平台目录隔离，交接物为数据快照与推荐结果文件，经 `backend/data/` 目录传递，详见 [docs/data-handoff.md](docs/data-handoff.md)。

## 测试

```bash
cd backend  && go test ./... && go vet ./...
cd frontend && pnpm test && pnpm type-check && pnpm lint
```

## 功能模块

认证 / 用户 / 资源 / 分类 / 评分 / 行为 / 管理后台 / 推荐（engine 结果导入 + 热门兜底）

## 数据流

```
platform export_snapshot（真实数据 → data/snapshots/<run_id>/）
→ engine 训练 + 全量推理（读该快照）
→ 推荐结果放至 data/recommendations.json → 管理员导入
→ Recommendation 缓存表 → GET /api/v1/recommendations → 前端首页
```

## 文档

- [docs/design.md](docs/design.md) —— 平台架构设计与决策记录
- [docs/api-design.md](docs/api-design.md) —— REST API 设计（推荐模块：读缓存 + 导入）
- [docs/engine-integration.md](docs/engine-integration.md) —— edurec-engine 接入说明（离线批量 + 结果落库）
- [docs/data-handoff.md](docs/data-handoff.md) —— 数据交接与一轮刷新流程
