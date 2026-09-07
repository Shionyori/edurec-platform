# 数据交接（engine ↔ platform，手动拷贝）

两仓库目录互不读写，产物统一手动拷贝。下方 `<engine>`、`<platform>` 为两仓库根目录。

| 交接 | 命令 |
|---|---|
| 推荐结果 engine → platform（推理后） | `cp <engine>/model/recommendations.json <platform>/backend/data/recommendations.json` |
| 模拟数据 engine → platform（播种用） | `cp -r <engine>/dataset/sim <platform>/backend/data/sim`，随后 `demo_seed` |
| 数据快照 platform → engine（导出后） | `cp -r <platform>/backend/data/snapshots/<run_id> <engine>/dataset/platform_snapshot/<run_id>/` |

快照拷给 engine 后，训练/推理命令见 engine README（`ENGINE_SNAPSHOT_DIR=dataset/platform_snapshot/<run_id>` + `--data-source platform`）。
