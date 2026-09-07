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

platform 侧不做模型推理：engine 离线完成「训练 + 全量推理」，产出每个用户的推荐结果列表；
platform 通过导入接口将其写入 `Recommendation` 缓存表，`GET /api/v1/recommendations` 读缓存返回，
未命中用户按评分降序热门兜底（空/缺失缓存不会导致无推荐，详见 [docs/engine-integration.md](docs/engine-integration.md)）。

```bash
cd backend
# （演示环境首次使用前播种：账号 demo<id>/demo123456，管理员 demo_admin/demo123456）
CONFIG_PATH=configs/config.yaml go run ./cmd/demo_seed -with-behaviors
# ① 平台真实数据导出为快照（供 engine 训练；engine 侧训练/推理见 docs/data-handoff.md）
CONFIG_PATH=configs/config.yaml go run ./cmd/export_snapshot   # → data/snapshots/<run_id>/
# ② engine 推理结果放至 data/recommendations.json 后，管理员导入 → 首页个性化推荐
#    demo_admin 登录 → POST /api/v1/admin/recommendations/import
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

- [docs/design.md](docs/design.md) · [docs/api-design.md](docs/api-design.md) · [docs/engine-integration.md](docs/engine-integration.md)
- [docs/data-handoff.md](docs/data-handoff.md) —— 数据交接与一轮刷新流程
