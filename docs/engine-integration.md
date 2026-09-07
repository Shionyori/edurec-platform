# edurec-engine 接入说明

## 背景

edurec-engine 是独立仓库（`github.com/Shionyori/edurec-engine`），离线完成「双塔 DSSM 召回 → 多任务 DeepFM 精排 → MMR 重排」。
platform 与 engine 的接入形态为**离线批量训练 + 结果物化落库**：engine 消费平台导出的数据快照完成训练与全量推理，
产出每个用户的推荐结果列表；platform 将结果导入 `Recommendation` 缓存表，推荐接口读缓存返回。
**platform 侧不运行模型、不做推理**——个性化结果完全来自 engine 离线产出，模型权重只保留在 engine。

该形态对齐真实推荐系统的离线 batch 链路：训练与全量推理代价高，按周期批量执行；serving 只消费预计算结果。
batch 形态为本项目**既定接入方式**，不以实时服务化为前提。

## 接入形态：离线批量 + 结果落库

### 数据流

```
platform 导出快照（export_snapshot，平台真实数据）
   ↓ 拷至 engine dataset/platform_snapshot/<run_id>/
engine 训练 + 全量推理（train_all / run_batch_infer，同一快照目录）
   ↓ model/recommendations.json（平台原始 ID，覆盖全量用户，冷启动走热门兜底）
拷回 platform data/ 后，POST /api/v1/admin/recommendations/import（管理员）
   ↓
Recommendation 缓存表（每个用户一条）
   ↓
GET /api/v1/recommendations → 命中缓存按序返回；未命中按评分降序热门兜底并写缓存
```

### 配置文件

`backend/configs/config.yaml`：

```yaml
engine:
  # 推荐结果文件：engine 生成后放至此处（见 docs/data-handoff.md）
  recommendations_file: "data/recommendations.json"
  # 演示/模拟数据集：engine dataset/sim 拷至此处（demo_seed 播种用）
  dataset_dir: "data/sim"
  # 数据快照导出目录（export_snapshot 输出；交由 engine 训练）
  snapshot_dir: "data/snapshots"
```

### 数据快照（platform → engine）

`CONFIG_PATH=configs/config.yaml go run ./cmd/export_snapshot` 将平台真实数据导出到 `data/snapshots/<run_id>/`
（`meta.json` + `users/resources/categories/behaviors/ratings.csv` + sha256）。快照拷给 engine 后：

```bash
python -m scripts.train_all       --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>
python -m scripts.run_batch_infer --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>
```

### 导入接口（engine → platform）

`POST /api/v1/admin/recommendations/import`（需管理员），无参数。

- 读取 `engine.recommendations_file` 指向的 JSON，格式为 `{ "<user_id>": [<resource_id>, ...] }`（每用户 top-N）
- 按平台库中实际存在的用户/资源过滤后，覆盖写入该用户的 `Recommendation` 缓存行
- 返回统计：`imported_users / skipped_users / imported_resources / skipped_resources`

### ID 与覆盖口径

- platform 数据源下 engine 全程使用**平台原始 ID**：快照导出全量用户（含冷启动）；engine 的清洗过滤只作用于训练集，
  推理对快照全量用户产出并在出口回映射为平台 ID——导入可全量命中，不存在 ID 错位匹配。
- sim / movielens 轨道仅用于演示与回归测试，其数据与平台 ID 无关，**不得**作为导入真实平台的来源。

### 服务语义：空 / 缺失 / 陈旧

- 用户无缓存行（新注册、未被覆盖）→ `GET` 按评分热门兜底并写缓存，不会无推荐。
- 单用户结果为空或用户不在平台库 → 导入跳过该用户，保留其旧缓存行。
- 整份文件为空或导入失败 → 缓存不变，服务继续返回旧结果或兜底（陈旧但可用）。

### 刷新节奏与时效口径

- 结果反映「截至导出时刻」的快照；用户新行为/新资源在下一轮「导出 → 训练/推理 → 导入」后生效。
- 一轮完整刷新见 [data-handoff.md](./data-handoff.md)；由人工或后续定时任务驱动。

## 演进方向（可选）

batch 形态为既定选择，以下作为可选演进，不改变主链路：

1. **在线服务化**：engine 增加 HTTP serving 层（如 FastAPI + ANN），缓存未命中时实时调用——属于实时推理，成本高，仅在场景确需时引入。
2. **近线增强**：请求时过滤「最近已看」、为新上架资源提供曝光通道、曝光打点与效果指标——在 batch 之外补时效与反馈。

## 相关文档

- [design.md](./design.md) —— 决策记录 #12
- [api-design.md](./api-design.md) —— 6.1 推荐接口、6.2 导入接口
- [data-handoff.md](./data-handoff.md) —— 数据交接与一轮刷新流程
- [README.md](../README.md) —— 推荐数据流速览
