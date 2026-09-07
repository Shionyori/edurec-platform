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

## 推荐闭环（路线 B：batch + 手动交接）

```bash
cd backend
# 演示数据播种（用户/资源 ID 与 engine 一致；账号 demo<id>/demo123456，管理员 demo_admin/demo123456）
CONFIG_PATH=configs/config.yaml go run ./cmd/demo_seed -with-behaviors
# engine 推理结果拷入后由管理员导入 → 首页个性化推荐
#   cp <engine>/model/recommendations.json data/recommendations.json
#   demo_admin 登录 → POST /api/v1/admin/recommendations/import

# 平台真实数据导出（供 engine 训练）
CONFIG_PATH=configs/config.yaml go run ./cmd/export_snapshot   # → data/snapshots/<run_id>/
```

engine 与平台目录隔离、产物手动拷贝交接，详见 [docs/data-handoff.md](docs/data-handoff.md)。

## 测试

```bash
cd backend  && go test ./... && go vet ./...
cd frontend && pnpm test && pnpm type-check && pnpm lint
```

## 功能模块

认证 / 用户 / 资源 / 分类 / 评分 / 行为 / 管理后台 / 推荐（engine 结果导入 + 热门兜底）

## 数据流

```
platform 导出快照 → 手动拷给 engine 训练/推理 → 结果拷回 platform 导入
→ Recommendation 缓存表 → GET /api/v1/recommendations → 前端首页
```

## 文档

- [docs/design.md](docs/design.md) · [docs/api-design.md](docs/api-design.md) · [docs/engine-integration.md](docs/engine-integration.md)
- [docs/data-handoff.md](docs/data-handoff.md) —— 目录隔离与手动交接
