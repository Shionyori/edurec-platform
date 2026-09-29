# edurec-platform

教育资源推荐平台（edurec-engine 的应用场景），后端 Go + 前端 Vue3。

| 子项目 | 技术栈 |
|---|---|
| backend | Go · Gin · GORM · MySQL · Redis · JWT |
| frontend | Vue 3 · Vite · Element Plus · Pinia · MSW |
| engine（独立仓库） | Python · PyTorch（语义双塔召回 + 质量融合） |

## 快速开始

### 启动服务（前后端、数据库的 docker 容器）

确保数据库容器已启动（MySQL 8 + Redis 7），然后启动前后端服务
```bash
cd backend
CONFIG_PATH=configs/config.yaml go run ./cmd/server
cd ../frontend
pnpm install && pnpm dev
```

也可以通过脚本一键完成
```bash
bash scripts/start.sh
```

### 准备数据

目前暂不支持导入自己准备好的资源（待实现），推荐直接从 B 站爬取资源并导入
```bash
cd backend/crawler
cp config.example.yaml config.yaml            # 可以按需更改任务与目标分类
.venv/bin/python run.py --config config.yaml  # 数据存储到 backend/data/bilibili
```

也可以通过脚本一键完成
```bash
bash scripts/bilibili.sh
```

### 冷启动

刚起服务时，平台没有用户行为数据，推荐模型无法训练，需要先模拟一批用户和他们对现有资源的行为
```bash
bash scripts/mock-users.sh
```

### 训练模型并导入

```bash
# 1.导出平台数据快照
cd backend
CONFIG_PATH=configs/config.yaml go run ./cmd/export_snapshot   # → data/snapshots/<run_id>/

# 2.快照拷给 engine（engine 与 platform 并列）
cp -r data/snapshots/<run_id> ../../edurec-engine/dataset/platform_snapshot/<run_id>/

# 3.engine 训练 + 推理
cd ../../edurec-engine
.venv/bin/python -m scripts.train_semantic --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>
.venv/bin/python -m scripts.infer_batch    --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>

# 4.结果拷回 platform
cp model/recommendations.json ../edurec-platform/backend/data/recommendations.json

# 5.导入（需后端运行；管理员登录后 POST /api/v1/admin/recommendations/import）
```

以上操作也可以通过脚本一键完成
```bash
bash scripts/handoff.sh
```

## 预设方案

### 从 B 站爬取资源

从 B 站爬取教育视频资源并导入平台
```bash
bash scripts/start.sh        # 起服务
bash scripts/bilibili.sh     # 采真实资源（B 站公开视频入库）
bash scripts/mock-users.sh   # 解决冷启动：模拟一批用户 + 交互（复用真实资源，不引入假资源）
bash scripts/handoff.sh      # 训练首版推荐
```

### 完全依靠模拟数据（仅测试）

用户行为数据和资源数据都由模拟脚本生成，仅用于离线演示/测试
```bash
bash scripts/start.sh
bash scripts/seed.sh         # 播 sim 假数据（含假资源）
bash scripts/handoff.sh
```

## 脚本

| 脚本 | 作用 |
|---|---|
| `scripts/start.sh` | 一键启动：容器 → 后端 → 前端（不准备数据） |
| `scripts/stop.sh [--with-db]` | 停止服务；`--with-db` 连容器一起停 |
| `scripts/seed.sh` | 播 sim 假数据（仅离线演示/测试，生产不用） |
| `scripts/bilibili.sh` | B 站采集+导入（`--dry-run`/`--crawl-only`/`--import-only`） |
| `scripts/handoff.sh [--infer-only]` | 推荐刷新：导出 → 训练 → 推理 → 导入；`--infer-only` 跳过训练 |
| `scripts/mock-users.sh` | 生成模拟用户 + 对现有资源的行为（不引入 sim 资源） |
| `scripts/clean.sh` | 清空数据库所有表数据（保留表结构） |

## 功能模块

### 已实现

| 模块 | 说明 |
|---|---|
| 认证 | 注册、登录、token 刷新 |
| 用户 | 个人资料查看/更新、兴趣分类（冷启动）、行为历史 |
| 资源 | 列表（关键词/分类/类型/标签筛选、排序）、详情、录入/更新/删除（管理员）、难度与时长 |
| 分类 | 列表、创建（管理员） |
| 评分 | 查看、打分 |
| 行为 | 浏览/点击/收藏上报、行为历史查询 |
| 曝光 | 推荐曝光打点（独立于行为，用于 CTR 统计） |
| 评论 | 展示 B 站爬取的热门评论（只读） |
| 推荐 | 个性化推荐（engine 结果导入 + 兴趣冷启动 + 热门兜底）、推荐理由、可追溯（run_id） |
| 推荐运营 | 运行记录、效果看板（曝光/点击/CTR）、离线评估（vs 热门基线） |
| 管理后台 | 用户/资源/分类管理、推荐导入与推荐管理 |
| B 站采集 | 离线批量采集 + 在线搜索/评论爬取 |

### 待实现

| 功能 | 现状 |
|---|---|
| 用户发表评论 | 评论接口只读（来自 B 站爬取），无发表能力 |
| 批量导入自有资源 | 仅支持 B 站格式导入 + 管理后台单个录入 |
| 多路召回 / 学习排序 / 重排 | engine 侧待做（见 docs/recsys-design.md 阶段 P2） |

## 测试

```bash
cd backend  && go test ./... && go vet ./...
cd frontend && pnpm test && pnpm type-check && pnpm lint
```

## 文档

- [docs/design.md](docs/design.md) —— 平台架构设计与决策记录
- [docs/api-design.md](docs/api-design.md) —— REST API 设计（推荐模块：读缓存 + 导入）
- [docs/engine-integration.md](docs/engine-integration.md) —— edurec-engine 接入说明（离线批量 + 结果落库）
- [docs/data-handoff.md](docs/data-handoff.md) —— 数据交接与一轮刷新流程
- [docs/bilibili-import.md](docs/bilibili-import.md) —— B 站教育视频采集与导入（字段映射、判重语义、合规边界）
