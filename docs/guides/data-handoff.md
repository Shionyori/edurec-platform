# 数据交接（platform ↔ engine）

两仓库目录互不读写，交接通过**文件**完成。下方 `<platform>`、`<engine>` 为两仓库根目录；
platform 交接目录为 `backend/data/`（快照、推荐结果、演示数据）。

契约的单一来源是 engine 的 `contracts/v1/`；platform 的 fixture 副本由
`scripts/check-contract-sync.sh` 维护。

## 交接物

| 交接 | 方向 | 位置 / 命令 |
|---|---|---|
| 数据快照（engine 训练/推理输入） | platform → engine | `<platform>/backend/data/snapshots/<run_id>/` → `<engine>/dataset/platform_snapshot/<run_id>/` |
| 推荐主文件 | engine → platform | `<engine>/model/recommendations.json` → `<platform>/backend/data/recommendations.json` |
| 推荐旁挂信封（可选） | engine → platform | `<engine>/model/recommendations.meta.json` → `<platform>/backend/data/recommendations.meta.json` |
| 模拟数据（仅演示） | engine → platform | `<engine>/dataset/sim` → `<platform>/backend/data/sim`，随后 `demo_seed` |

> 交接物是**推荐结果列表**，不是模型权重——platform 不做模型推理，模型只留在 engine。

## 一键刷新（推荐）

```bash
# 在 platform 仓库根目录执行（engine 默认取 ../edurec-engine）
bash scripts/handoff.sh               # 导出 → 校验快照 → 训练 → 推理 → 校验产物 → 导入
bash scripts/handoff.sh --infer-only  # 复用已有 model/semantic_recall.pt，只导出 → 推理 → 导入
```

脚本与下方手动步骤一一对应。可用环境变量覆盖：
`ENGINE_ROOT` / `CONFIG_PATH` / `SNAPSHOT_DIR` / `BASE_URL` / `ADMIN_USER` / `ADMIN_PASS`。
第 ⑥ 步经 HTTP 导入，需先启动后端服务器。

## 手动步骤（等价参考）

```bash
# ① platform：导出平台真实数据快照 → backend/data/snapshots/<run_id>/
cd <platform>/backend
CONFIG_PATH=configs/config.yaml go run ./cmd/export_snapshot

# ② 校验快照并交给 engine
<engine>/.venv/bin/python <engine>/contracts/v1/validate.py snapshot \
  data/snapshots/<run_id>
rm -rf <engine>/dataset/platform_snapshot/<run_id>
cp -r data/snapshots/<run_id> <engine>/dataset/platform_snapshot/<run_id>

# ③ engine：训练 + 全量推理（必须在 engine 仓库根目录）
cd <engine>
.venv/bin/python -m scripts.train_semantic \
  --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>
.venv/bin/python -m scripts.run_batch_infer \
  --data-source platform --snapshot-dir dataset/platform_snapshot/<run_id>

# ④ 校验产物并回传
.venv/bin/python contracts/v1/validate.py recommendations \
  model/recommendations.json --envelope model/recommendations.meta.json
cp model/recommendations.json <platform>/backend/data/recommendations.json
cp model/recommendations.meta.json <platform>/backend/data/recommendations.meta.json

# ⑤ platform：导入缓存表（管理员登录后）
# POST /api/v1/admin/recommendations/import
```

`--data-source` 在训练/推理入口都是**必填**；只有 `platform` 轨道可以导入 platform。
不带该参数会直接报用法错误，不会再出现「用 MOOCCube 产物导入、skipped_users 接近全量」的静默事故。

## 契约校验

| 校验点 | 命令 | 失败后果 |
|---|---|---|
| 快照 | `python contracts/v1/validate.py snapshot <dir>` | `handoff.sh` 中止，不训练 |
| 推荐主文件 + 信封 | `python contracts/v1/validate.py recommendations <file> --envelope <meta>` | `run_batch_infer` / `handoff.sh` 中止，不交付 |
| 两侧 fixture 同步 | `bash scripts/check-contract-sync.sh` | 提示先改 engine 契约再 `--update` |

校验器检查：BOM、UTF-8、`contract_version`、`files_sha256`、CSV 必需列/类型/行数、
推荐主文件的键/值类型、非负、去重、顶层无兄弟字段，以及信封 `scores`/`reasons`
与主文件长度一一对应。

## 服务语义

- 单用户结果为空或用户不在平台库 → 导入跳过该用户，保留其旧缓存行。
- 整份文件为空或导入失败 → 缓存不变，服务继续返回旧结果或热门兜底。
- 用户新行为/新资源在下一次完整刷新后生效（batch 时效口径）。
- 冷启动用户也在快照里；engine 为无行为用户输出热门兜底。

## 演示数据（不要与真实数据混用）

```bash
bash scripts/seed.sh            # 把 engine 的 dataset/sim 拷到 backend/data/sim 并播种
```

sim 轨道的用户/资源 ID 与平台真实库无关，**只用于离线演示/测试**；
`data/recommendations.json` 若来自 sim 轨道，导入真实库时只会命中极少数 ID。
生产/演示真实数据请走 `bilibili.sh` + `mock-users.sh` + `handoff.sh`。
