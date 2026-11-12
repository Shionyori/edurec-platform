# edurec-engine 接入说明

## 1. 边界

edurec-engine 是独立仓库（`github.com/Shionyori/edurec-engine`），离线完成
「语义双塔召回 + 质量信号融合」。platform 与 engine 的接入形态是**离线批量训练 + 结果物化落库**：

```
platform 导出快照 → engine 训练 + 全量推理 → 推荐结果文件 → platform 导入缓存表
```

- **platform 不运行模型、不做推理**；模型权重只保留在 engine。
- 两仓库**不共享数据库、不共享代码**；交叉接口是 platform 的 MySQL 数据的序列化载体
  （CSV 快照进、排名 ID 列表出）。
- 唯一允许的在线调用是**可选**的知识生成服务（`kg_server.py`），
  只用于「标题/描述 → 概念」，不参与推荐主链路（见 §7）。

### 1.1 工程最终形态：一个仓库、两个部署单元

engine 虽然同时提供离线与在线能力，但**不是同一个部署单元**：

| 单元 | 形态 | 资源/失败域 | 交互 |
|---|---|---|---|
| `engine-batch` | Job（无入站端口） | 训练/推理资源；失败只影响本轮刷新 | C1 工件契约（快照进、推荐出） |
| `engine-kg` | 常驻 HTTP 服务（内网） | CPU/IO；无状态、可多副本；失败只影响自动生成概念 | C2 在线契约 |

platform 侧对应约束：

- 离线推荐永远不依赖 KG 服务；
- KG 调用有超时、并发上限、熔断、鉴权与 request id，不自动重试；
- platform 不加载模型、不执行 Python、不读 engine 目录。

完整设计见 [`target-architecture.md`](../architecture.md)，可执行约束见
[`architecture-constraints.md`](../constraints.md)。

规范契约（engine 仓库）：

- 主契约：[`docs/contracts/platform-contract.md`](https://github.com/Shionyori/edurec-engine/blob/main/docs/contracts/platform-contract.md)
- 可执行版本：`../edurec-engine/contracts/v1/`（JSON Schema + fixtures + 校验器）
- 当前数据流：[`docs/contracts/data-flow.md`](https://github.com/Shionyori/edurec-engine/blob/main/docs/contracts/data-flow.md)

## 2. 数据流

```
platform 导出快照（cmd/export_snapshot，平台真实数据）
   ↓ 拷至 engine dataset/platform_snapshot/<run_id>/
engine 训练 + 全量推理（train_semantic / run_batch_infer，同一快照目录）
   ↓ model/recommendations.json（+ 可选 recommendations.meta.json）
拷回 platform backend/data/ 后，POST /api/v1/admin/recommendations/import（管理员）
   ↓
Recommendation 缓存表（每个用户一条）
   ↓
GET /api/v1/recommendations → 命中缓存按序返回；未命中按评分降序热门兜底并写缓存
```

一轮完整刷新（含校验步骤）见 [data-handoff.md](../guides/data-handoff.md)，
一键脚本为 [`scripts/handoff.sh`](../../scripts/handoff.sh)。

## 3. 配置

`backend/configs/config.yaml`：

```yaml
engine:
  # 推荐结果文件：engine 生成后放至此处（见 docs/guides/data-handoff.md）
  recommendations_file: "data/recommendations.json"
  # 演示/模拟数据集：engine dataset/sim 拷至此处（demo_seed 播种用）
  dataset_dir: "data/sim"
  # 数据快照导出目录（export_snapshot 输出；交由 engine 训练）
  snapshot_dir: "data/snapshots"

knowledge:
  # MOOCCube 数据目录：平台本地外部数据（默认 backend/data/MOOCCube，不入库）。
  # 不指向 engine 仓库；为空时只建学科层、跳过概念层。
  source_dir: "data/MOOCCube"
  # 可选在线能力：engine 的 kg_server.py；未启动只影响自动生成概念。
  generator_url: "http://127.0.0.1:8090"
  generator_timeout_seconds: 180
```

## 4. 数据快照（platform → engine）

```bash
cd backend
CONFIG_PATH=configs/config.yaml go run ./cmd/export_snapshot
# → data/snapshots/<run_id>/
```

快照内容：`meta.json` + `users.csv / resources.csv / behaviors.csv / ratings.csv`
（`categories.csv` 附带）。engine 加载前会校验：

- `contract_version == 1`（不认识就报错退出）；
- `files_sha256`（挡住拷贝中断）；
- `resources.csv` 的必需列（`description` 为可选，缺失时降级为空串）。

## 5. 导入接口（engine → platform）

`POST /api/v1/admin/recommendations/import`（需管理员），无参数。

- 读取 `engine.recommendations_file` 指向的 JSON，格式为
  `{ "<user_id>": [<resource_id>, ...] }`（每用户 top-N，按相关性降序、已去重）。
- 按平台库中实际存在的用户/资源过滤后，覆盖写入该用户的 `Recommendation` 缓存行。
- 返回统计：`imported_users / skipped_users / imported_resources / skipped_resources`。
- 同目录的 `<主文件名>.meta.json` 是可选旁挂信封（run_id、生成时间、模型、分数、理由）；
  缺失/损坏不影响导入。

## 6. ID、覆盖与服务语义

- **ID 原样透传**：platform 轨道上 engine 使用平台原始整数 ID；sim / movielens 轨道仅用于
  演示与回归，其输出**不得**导入 platform。
- **覆盖全量用户**：包括冷启动用户（走热门兜底）。单用户为空数组或缺席 = 「本次不更新」，
  平台保留旧缓存行。
- **服务始终有返回**：缓存未命中 → 按评分热门兜底并写缓存；导入失败 → 旧结果或兜底继续服务
  （陈旧但可用）。
- **时效**：结果反映「截至导出时刻」的快照；新行为/新资源在下一轮刷新后生效。

## 7. 可选在线能力：知识生成

engine 提供 `scripts/kg_server.py`：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | liveness + `contract_version` + capabilities |
| GET | `/ready` | readiness（可选检查模型就绪标记） |
| POST | `/knowledge/generate` | `{title, description, point_code}` → 概念列表 |
| POST | `/knowledge/generate_batch` | 批量，单条失败不阻断整批 |

生产约束：

- **鉴权**：engine 启用 `KG_SERVICE_TOKEN`，platform 配置
  `KNOWLEDGE_GENERATOR_TOKEN`；两边一致，token 不写仓库。
- **限额**：请求体 ≤1 MiB、批量 ≤50、并发 ≤4（均可配置）；超限返回 4xx。
- **关联**：透传 `X-Request-ID`；响应带 `X-Contract-Version: 1`。
- **客户端**：超时 + 并发上限 + 连续失败熔断；**不自动重试生成**。
- **降级**：engine 未启动/熔断时只影响「自动生成概念」，推荐、搜索、学习路径不受影响。

契约见 engine 的 [`docs/contracts/kg-service-contract.md`](https://github.com/Shionyori/edurec-engine/blob/main/docs/contracts/kg-service-contract.md)。
这也与「两仓库目录互不读写」不冲突：它是显式、可选的 HTTP 契约。

## 8. 演进方向（可选，不改变主链路）

1. **近线增强**：请求时过滤「最近已看」、新资源曝光通道、曝光打点与效果指标。
2. **在线推理**：engine 增加 serving 层（FastAPI + ANN），缓存未命中时实时调用；
   成本高，仅在场景确需时引入。

## 9. 相关文档

- [data-handoff.md](../guides/data-handoff.md) —— 一轮刷新的完整步骤与校验
- [design.md](../design/architecture-decisions.md) —— 架构与决策记录
- [api-design.md](../design/api-design.md) —— 推荐接口与导入接口
- engine 侧：[`docs/contracts/platform-contract.md`](https://github.com/Shionyori/edurec-engine/blob/main/docs/contracts/platform-contract.md)、
  [`docs/contracts/data-flow.md`](https://github.com/Shionyori/edurec-engine/blob/main/docs/contracts/data-flow.md)
