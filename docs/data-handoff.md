# 数据交接手册（目录隔离 + 手动转移）

> 原则：**engine 与 platform 互不读写对方目录**。engine 的产物留在 engine
> 本地；platform 的产物留在 platform 本地；两者之间一律**手动拷贝**交接，
> 避免跨项目交叉引用。下方相对路径均相对 `platform/backend/` 目录运行。

## 目录边界

| 侧 | 只读写自己的目录 |
|---|---|
| engine | `model/`（models.pt、metrics.json、recommendations.json）、`dataset/`（sim、platform_snapshot） |
| platform | `data/`（data/snapshots、data/sim、data/recommendations.json，不入 git） |

```
engine 侧                              platform 侧 (backend/data/)
model/recommendations.json  ──拷贝──►  data/recommendations.json   (import 读)
dataset/sim/*.csv           ──拷贝──►  data/sim/                   (demo_seed 读)
                                      data/snapshots/<run_id>/     (export 写)
data/snapshots/<run_id>     ◄──拷贝──  (快照拷到 engine dataset/platform_snapshot/<run_id>/ 供训练)
```

## 交接命令

### ① 推荐结果：engine → platform（每次推理后）
```bash
mkdir -p <platform>/backend/data
cp <engine>/model/recommendations.json <platform>/backend/data/recommendations.json
```

### ② 模拟数据：engine → platform（仅演示播种需要）
```bash
cp -r <engine>/dataset/sim <platform>/backend/data/sim
cd <platform>/backend && CONFIG_PATH=configs/config.yaml go run ./cmd/demo_seed -with-behaviors
```

### ③ 真实数据快照：platform → engine（每次导出后）
```bash
cd <platform>/backend && CONFIG_PATH=configs/config.yaml go run ./cmd/export_snapshot
# 记下输出的 run_id，然后拷到 engine 侧
mkdir -p <engine>/dataset/platform_snapshot/<run_id>
cp -r <platform>/backend/data/snapshots/<run_id>/* <engine>/dataset/platform_snapshot/<run_id>/
# engine 训练/推理
cd <engine>
ENGINE_SNAPSHOT_DIR=dataset/platform_snapshot/<run_id> PYTHONPATH=src \
  python -m scripts.train_all --data-source platform
ENGINE_SNAPSHOT_DIR=dataset/platform_snapshot/<run_id> PYTHONPATH=src \
  python -m scripts.run_batch_infer
# 回到 ①：把新 recommendations.json 拷回 platform 再 import
```

> 手动转移的代价：容易忘记/拷错 run_id。后续自动化（M2 调度）可改为
> “定时执行上述命令 + 校验 run_id 与 sha256”，而不是放开目录交叉。
