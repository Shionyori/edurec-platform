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

三种来源，按需选择：

1. **从 B 站爬取**（推荐做真实演示）：
```bash
cd backend/crawler
cp config.example.yaml config.yaml            # 可以按需更改任务与目标分类
.venv/bin/python run.py --config config.yaml  # 数据存储到 backend/data/bilibili
```
2. **导入自己的数据集**：json / jsonl / csv，在 `configs/config.yaml` 的
   `datasets.<名称>` 配好字段映射，再走 `cmd/import_dataset`（先 `--preview` / `--dry-run`）。
   见 [docs/guides/dataset-import.md](docs/guides/dataset-import.md)。
3. **纯模拟数据**（仅离线测试）：`bash scripts/seed.sh`。

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

# 2.校验快照并拷给 engine（engine 与 platform 并列）
../edurec-engine/.venv/bin/python ../edurec-engine/contracts/v1/validate.py snapshot data/snapshots/<run_id>
cp -r data/snapshots/<run_id> ../../edurec-engine/dataset/platform_snapshot/<run_id>/

# 3.engine 训练 + 推理（必须在 engine 仓库根目录）
cd ../../edurec-engine
.venv/bin/python -m scripts.train_semantic --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>
.venv/bin/python -m scripts.run_batch_infer --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>
.venv/bin/python contracts/v1/validate.py recommendations \
  model/recommendations.json --envelope model/recommendations.meta.json

# 4.结果拷回 platform（信封可选）
cp model/recommendations.json ../edurec-platform/backend/data/recommendations.json
cp model/recommendations.meta.json ../edurec-platform/backend/data/recommendations.meta.json

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
| `scripts/handoff.sh [--infer-only]` | 推荐刷新：导出 → **校验快照** → 训练 → 推理 → **校验产物** → 导入；`--infer-only` 跳过训练 |
| `scripts/check-contract-sync.sh [--update]` | 校验/同步 platform 侧的契约 fixture 与 engine 单一来源一致 |
| `scripts/verify-all.sh [--with-frontend]` | 一键验证：engine 测试 + 契约 fixture + 同步检查 + 后端测试（可选前端） |
| `scripts/mock-users.sh` | 生成模拟用户 + 对现有资源的行为（不引入 sim 资源） |
| `scripts/clean.sh` | 清空数据库所有表数据（保留表结构） |
| `scripts/rebuild-kg.ps1` / `.sh` | 重建知识图谱（学科 + 概念），`-DryRun` / `-KeepOld` |
| `scripts/set-kg-source.ps1` / `.sh` | 一键切换知识图谱数据源路径（平台本地 MOOCCube） |

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
| 数据集导入 | 配置驱动的 json/jsonl/csv 字段映射 → `resources`（`cmd/import_dataset`，见 docs/guides/dataset-import.md） |
| 知识图谱 | 21 个学科 / 72 个知识点 + 19 条先修边，关联 380+ 资源；导入 MOOCCube 概念图（425 概念 / 1027 边） |
| 学科导航 | 主页面按学科组织；学科页展示知识点与资源，无资源时自动爬取 B 站 |
| 自适应学习 | 初始问卷 → 知识画像（问卷/行为/完成度）→ 先修路径 → 逐节解锁，只推荐当前未掌握内容 |
| 关键帧 | B 站封面 + 视频关键帧（雪碧图）采集与详情页展示 |

### 待实现

| 功能 | 现状 |
|---|---|
| 用户发表评论 | 评论接口只读（来自 B 站爬取），无发表能力 |
| 多路召回 / 学习排序 / 重排 | engine 侧待做（见 docs/design/recsys-design.md 阶段 P2） |
| 完成信号（complete） | 当前只采 view/click/favorite + 曝光，未建模「学完」 |

## 测试

```bash
cd backend  && go test ./... && go vet ./...
cd frontend && pnpm test && pnpm type-check && pnpm lint
```

## 文档

- [docs/README.md](docs/README.md) —— **文档地图（从这里开始）**
- [docs/architecture.md](docs/architecture.md) —— 双仓目标架构与部署形态
- [docs/status.md](docs/status.md) —— 当前设计状态与路线图
- [docs/constraints.md](docs/constraints.md) —— 可执行的架构约束
- [docs/contracts/README.md](docs/contracts/README.md) —— 跨仓契约（C1 / C2）

详细设计在 `docs/design/`，操作指南在 `docs/guides/`；均已由文档地图索引。

