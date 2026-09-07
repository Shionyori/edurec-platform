# edurec-platform

教育资源推荐平台。作为 [edurec-engine](https://github.com/Shionyori/edurec-engine) 的应用场景，提供教育资源的浏览、搜索、评分、行为记录与个性化推荐，并内置管理后台。

> engine 是独立维护的仓库（推荐服务），通常与本仓库并列放在同一上级目录。当前以「路线 B（batch + 落库 + 手动文件交接）」接入：平台导出真实数据快照供 engine 训练，engine 推理结果拷回平台导入。目录相互隔离，见 [docs/data-handoff.md](docs/data-handoff.md)。远期按「独立微服务 + REST」演进。

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
# 准备 MySQL 与 Redis，按需修改 configs/config.yaml（含 engine.* 交接路径）
go run ./cmd/server
```

### 演示数据播种 + 推荐导入（快速体验 route B）

```bash
cd backend
# ① 拷贝 engine 的模拟数据集到本地 data/sim（见 docs/data-handoff.md）
# ② 播种：用户/资源/类目 ID 与 engine 一致；演示账号 demo<id>/demo123456、管理员 demo_admin/demo123456
CONFIG_PATH=configs/config.yaml go run ./cmd/demo_seed -with-behaviors
# ③ 拷贝 engine 推理结果到 data/recommendations.json（见 docs/data-handoff.md）
# ④ 管理员登录（demo_admin）→ POST /api/v1/admin/recommendations/import
# ⑤ 前端/接口登录 demo1 即可见个性化推荐
```

### 导出平台真实数据快照（供 engine 训练）

```bash
cd backend
CONFIG_PATH=configs/config.yaml go run ./cmd/export_snapshot   # → data/snapshots/<run_id>/
# 将快照手动拷给 engine 训练（见 docs/data-handoff.md），engine 产出后拷回 data/recommendations.json 再导入
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

| 模块 | 说明 |
|---|---|
| 认证 | 注册 / 登录 / 刷新 token |
| 用户 | 个人资料查看与编辑 |
| 资源 | 列表 / 详情 / 增删改（管理员） |
| 分类 | 列表 / 创建（管理员） |
| 评分 | 资源评分与评论 |
| 行为 | 浏览 / 点击 / 收藏上报与历史 |
| 管理后台 | 仪表盘 / 用户 / 资源 / 分类 |
| 推荐 | 个性化推荐 + engine 结果导入 |

## 推荐数据流（路线 B）

```
edurec-engine（独立仓库，目录隔离）
  ├─ 训练数据：读取平台手动拷贝来的 data/snapshots/<run_id>（export_snapshot 导出）
  ├─ 推理结果：model/recommendations.json（key/value 均为平台/数据集原始 ID）
  └─ 手动拷回 → 本仓库 data/recommendations.json
        ↓  POST /api/v1/admin/recommendations/import（管理员触发）
Recommendation 缓存表（每个用户一条，覆盖写）
        ↓  GET /api/v1/recommendations
前端首页展示个性化推荐（缓存未命中时兜底按评分降序取热门）
```

> 演示闭环：engine `dataset/sim` 与平台 `data/sim` 播种数据 ID 一致 → 导入可 100% 命中。
> 真实闭环：平台 `export_snapshot` 导出的快照即 engine 训练集，推理产物 ID 天然对齐。
> 详细说明见 [docs/engine-integration.md](docs/engine-integration.md) 与 [docs/data-handoff.md](docs/data-handoff.md)。

## 文档

- [docs/design.md](docs/design.md) —— 架构与决策记录
- [docs/api-design.md](docs/api-design.md) —— 接口设计（20 个接口全部实现）
- [docs/engine-integration.md](docs/engine-integration.md) —— edurec-engine 接入说明（路线 B 现状）
- [docs/data-handoff.md](docs/data-handoff.md) —— 目录隔离与手动文件交接手册
