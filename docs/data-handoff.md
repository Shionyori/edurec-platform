# 数据交接（engine ↔ platform）

两仓库目录互不读写，交接通过文件完成。下方 `<engine>`、`<platform>` 为两仓库根目录，
platform 交接目录为 `backend/data/`（快照、推荐结果、演示数据）。

## 交接物一览

| 交接 | 方向 | 命令 |
|---|---|---|
| 数据快照（engine 训练/推理输入） | platform → engine | `cp -r <platform>/backend/data/snapshots/<run_id> <engine>/dataset/platform_snapshot/<run_id>/` |
| 推荐结果（engine 推理输出） | engine → platform | `cp <engine>/model/recommendations.json <platform>/backend/data/recommendations.json` |
| 模拟数据（演示播种用） | engine → platform | `cp -r <engine>/dataset/sim <platform>/backend/data/sim`，随后 `demo_seed` |

> 交接物是**推荐结果列表**（`{用户ID: [资源ID,…]}`），不是模型权重——platform 不做模型推理，
> 模型文件只保留在 engine，供下一次推理使用。

## 一轮推荐刷新（完整流程）

推荐反映「截至导出时刻」的快照：完成一次「导出 → engine 训练/推理 → 导入」后，
`GET /recommendations` 才会返回新结果。

### 一键刷新（推荐）

下方手动步骤已固化为 [`scripts/handoff.sh`](../scripts/handoff.sh)，一条命令跑完整轮：

```bash
# 在 platform 仓库根目录执行（engine 默认取 ../edurec-engine）
bash scripts/handoff.sh               # 导出 + 训练 + 推理 + 导入
bash scripts/handoff.sh --infer-only  # 复用已有 model/models.pt，只导出 + 推理 + 导入
```

脚本与手动步骤一一对应；engine 根目录、后端配置、管理员账号等可用环境变量覆盖
（`ENGINE_ROOT` / `CONFIG_PATH` / `SNAPSHOT_DIR` / `BASE_URL` / `ADMIN_USER` / `ADMIN_PASS`），详见脚本头注释。
第 ⑥ 步经 HTTP 导入，需先启动后端服务器。

### 手动步骤（等价参考）

```bash
# ① platform：导出平台真实数据快照
cd <platform>/backend && CONFIG_PATH=configs/config.yaml go run ./cmd/export_snapshot
#                                          → data/snapshots/<run_id>/

# ② 快照交给 engine（拷入 engine 的快照目录）
cp -r data/snapshots/<run_id> <engine>/dataset/platform_snapshot/<run_id>/

# ③ engine：训练 + 全量推理（同一快照目录，保证训练/推理口径一致）
cd <engine> && python -m scripts.train_all       --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>
cd <engine> && python -m scripts.run_batch_infer --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>

# ④ 推理结果放回 platform
cp <engine>/model/recommendations.json <platform>/backend/data/recommendations.json

# ⑤ platform：导入缓存表（管理员登录后 POST /api/v1/admin/recommendations/import）
```

说明：

- ③ 中 engine 输出为**平台原始 ID**、覆盖快照全量用户；无行为的冷启动用户走热门兜底。
- 单用户结果为空或用户不在平台库 → 导入跳过该用户、保留其旧缓存行；
  整份文件为空/导入失败 → 缓存不变。服务始终有返回（旧结果或热门兜底），不会出现空推荐。
- 用户新行为/新资源在下一次完整刷新后生效（batch 时效口径）。
