# edurec-platform

教育资源推荐平台。作为 [edurec-engine](https://github.com/Shionyori/edurec-engine) 的应用场景，提供教育资源的浏览、搜索、评分、行为记录与个性化推荐，并内置管理后台。

> engine 是独立维护的仓库（推荐服务），通常与本仓库并列放在同一上级目录。平台侧按「独立微服务 + REST」的远期设计推进，当前以「路线 B（batch + 落库）」接入，engine 仓库零改动。

## 技术栈

| 子项目 | 技术栈 |
|---|---|
| backend | Go 1.26 · Gin · GORM · MySQL · Redis · JWT（Access + Refresh） |
| frontend | Vue 3 · TypeScript · Vite · Element Plus · Pinia · Vitest · MSW |
| engine（独立仓库） | Python 3.10+ · PyTorch（双塔 DSSM 召回 + 多任务 DeepFM 精排 + MMR 重排） |

## 仓库结构

```
edurec-platform/
├── backend/   # Go 后端（Handler → Service → Repository 分层，AutoMigrate 建表）
├── frontend/  # Vue 前端（api / components / pages / router / stores）
├── docs/      # 设计文档（design / api-design / engine-integration）
└── CLAUDE.md  # 协作与分支提交约定
```

## 快速开始

### 后端

```bash
cd backend
# 准备 MySQL 与 Redis，按需修改 configs/config.yaml
go run ./cmd/server
```

### 前端

```bash
cd frontend
pnpm install
pnpm dev   # dev 环境 VITE_MOCK=1，启用 MSW mock，可脱离后端开发
```

### 测试与检查

```bash
cd backend  && go test ./... && go vet ./...
cd frontend && pnpm test && pnpm type-check && pnpm lint
```

## 功能模块

| 模块 | 说明 | 状态 |
|---|---|---|
| 认证 | 注册 / 登录 / 刷新 token | ✅ |
| 用户 | 个人资料查看与编辑 | ✅ |
| 资源 | 列表 / 详情 / 增删改（管理员） | ✅ |
| 分类 | 列表 / 创建（管理员） | ✅ |
| 评分 | 资源评分与评论 | ✅ |
| 行为 | 浏览 / 点击 / 收藏上报与历史 | ✅ |
| 管理后台 | 仪表盘 / 用户 / 资源 / 分类 | ✅ |
| 推荐 | 个性化推荐 + engine 结果导入 | ✅ |

## 推荐数据流（路线 B）

```
edurec-engine 离线推理 → model/recommendations.json
   ↓  POST /api/v1/admin/recommendations/import（管理员触发）
Recommendation 缓存表（每个用户一条）
   ↓  GET /api/v1/recommendations
前端首页展示个性化推荐（缓存未命中时兜底按评分降序取热门）
```

> engine 当前基于其模拟数据集训练，推荐 ID 与平台库真实 ID 并不对应；导入时只匹配数值 ID 相同的用户与资源。详见 [docs/engine-integration.md](docs/engine-integration.md)。

## 文档

- [docs/design.md](docs/design.md) —— 架构与决策记录
- [docs/api-design.md](docs/api-design.md) —— 接口设计（20 个接口全部实现）
- [docs/engine-integration.md](docs/engine-integration.md) —— edurec-engine 接入说明
