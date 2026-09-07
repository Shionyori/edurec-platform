# edurec-engine × edurec-platform 对接重设计方案（路线 B+）

> 状态：草案（待评审）
> 版本：v2 初稿
> 范围：platform（Go）与 engine（Python）两仓库之间的数据对接重新设计
> 替代文档：本方案落地后取代 `docs/engine-integration.md`（路线 B 现状版）
> 关联：engine `docs/design.md` §11/§13、platform `docs/api-design.md` §6

---

## 0. 现状问题清单（为什么重设计）

对现状代码与产物逐项核验后，确认以下对接层面的问题：

| # | 问题 | 证据 |
|---|------|------|
| P1 | **ID 身份断裂**：engine 输出的是内部连续编码，且未保存回映射 | `preprocess.build_vocab` 对清洗后的用户/资源重新连续编码；`pipeline/infer_batch.py` 直接以编码 ID 输出。实测 sim 场景丢弃用户 id=36/45/121…（散布），`recommendations.json` 的 key `"36"` 实为原始用户 37。platform 导入按“数值 ID 相同”匹配实际是在**错位匹配** |
| P2 | **engine 吃不到平台数据**：无导出通道、无 schema mapper | platform 无任何数据导出接口/脚本；engine 只有 `simulator`/`movielens` 两个 loader |
| P3 | **无真实效果度量与反馈**：无曝光日志 → 无真负样本、无上线后指标 | platform 仅 view/click/favorite 正行为上报；`RecommendationService.Get` 无打点 |
| P4 | **导入无版本/校验/回滚/门禁**，且靠管理员手动触发 | import 接口无参数直接覆盖写缓存表 |
| P5 | **engine 推理侧硬编码/失真**：数据源硬编码 sim、hour/dow=固定值、user 数值特征=0、冷启动加权实现为“全体 ×1.2” | `scripts/run_batch_infer.py` 写死读 `dataset/sim`；`pipeline/infer_batch.py` hour=12/dow=0、user numeric=0；`pipeline/rerank.py` 无条件乘 `cold_weight` |
| P6 | **train 与 infer 可分叉**：`train_all` 与 `run_batch_infer` 是两步，模型与推理数据源/清洗口径可能不一致 | 无“同一数据快照→同一产物”的原子约束 |
| P7 | **无自动刷新与新鲜度保障**：缓存表只随手动导入更新 | `Recommendation` 表无 TTL/调度 |

---

## 1. 目标与非目标

### 目标
1. 建立**身份连续**的数据链路：platform 真实 ID 从导出、训练、推理到导入全程可追溯、零错位。
2. 建立**训练数据从平台来的**闭环：平台可导出供 engine 训练的数据集；engine 消费该数据集产出**平台 ID 的推荐**。
3. 引入**版本化产物 + 校验 + 可回滚**，导入可审计、可灰度。
4. 引入**曝光打点**，使“推荐是否有效”可度量，并逐步获得真负样本。
5. 保持两仓库独立演进（维持现状拓扑，不 submodule、不 monorepo），对接物收敛为“**一份带版本的数据契约**”。

### 非目标（本期不做）
- 在线实时推荐服务（路线 A：REST/ANN/faiss）——仅留扩展点。
- 模型结构/特征体系重设计（DSSM/DeepFM/MMR 保留，只修对接侧缺陷）。
- 多租户、超大规模（>10 万用户级）工程。

---

## 2. 总体架构与数据流（三回路 + 一次流水线）

```
                    ┌──────── platform MySQL ────────┐
                    │ users/resources/categories/     │
                    │ user_behaviors/ratings          │
                    └───────┬────────────────────────┘
  回路① 训练数据           │ 导出 job（admin 鉴权，每日/可配置窗口）
                            ▼
            platform_snapshot/<run_id>/        ← meta.json + 5 张 csv + sha256
                            │   (契约 v1，字段见 §3.1)
                            ▼
   ┌──────────────── engine 单次流水线 ────────────────┐
   │  load(platform loader) → clean → vocab → 训练      │
   │  → 评估 → 推理(全量用户) → 回映射为平台 ID            │
   │  → artifacts/<run_id>/  ← manifest/models/vocab/   │
   │       recommendations/metrics（原子产出，同快照）    │
   └────────────────────────┬──────────────────────────┘
  回路② 推荐结果            │ 导入 job（admin 鉴权）
                            ▼
               recommendations 缓存表（按 run_id 全量替换）
                            │
                            ▼
              GET /api/v1/recommendations → 前端
                            │
  回路③ 反馈                ▼ 曝光打点（M3 起）
              recommendation_exposures + 既有行为
                            │
                            └─► 下一轮导出的负样本与效果指标
```

**关键设计决策**：engine 训练与推理合并为**单次流水线调用**（给定一个快照 → 原子产出 artifacts/<run_id>/），从机制上消除 P6（train/infer 分叉）；platform 只认“带 manifest 的 run_id 产物”，导入天然可回滚（回滚 = 重新导入上一个 run_id）。

---

## 3. 数据契约（对接的唯一真相）

契约文件：两仓库各存一份只读副本（engine `docs/contract.md`、platform `docs/contract.md`），变更需**双仓同步 + 版本号递增**。

### 3.1 快照格式（platform → engine），版本 v1

目录布局（`platform_snapshot/<run_id>/`，`run_id = yyyymmdd_HHMMSS`）：

| 文件 | 内容 | 说明 |
|------|------|------|
| `meta.json` | 见下 | 快照元信息，头部校验 |
| `users.csv` | `user_id` | **全量用户**（含冷启动），不含任何个人字段（见 §8） |
| `resources.csv` | `resource_id,title,type,category_id,tags_json,metadata_json,avg_rating,view_count,created_at` | `type ∈ {course,article,video}`；`tags_json` 为 JSON 字符串数组；`created_at` unix 秒 |
| `categories.csv` | `category_id,name` | 平台 categories 全表 |
| `behaviors.csv` | `user_id,resource_id,action,ts` | `action ∈ {view,click,favorite}`；`ts` unix 秒；窗口由 meta 声明 |
| `ratings.csv` | `user_id,resource_id,score,ts` | `score ∈ 1..5` |

`meta.json`：

```json
{
  "contract_version": 1,
  "run_id": "20260815_030000",
  "exported_at": 1755200000,
  "behavior_window": {"start": 0, "end": 1755200000},
  "users_count": 2000,
  "resources_count": 500,
  "behaviors_count": 100000,
  "excludes_soft_deleted": true,
  "files_sha256": {"users.csv": "…", "resources.csv": "…"}
}
```

规则：
- **所有 ID 均为平台数据库原始 ID**（无内部重排），包括被 soft delete 的行不导出。
- 时间统一转 unix 秒（UTC），避免 MySQL `LOCAL` 时区歧义。
- `users.csv` 是全量用户；**清洗过滤只发生在 engine 训练侧，导出侧不丢用户**（冷启动用户保留给推理兜底，P2/P6 配套）。
- engine 导入方先校验 `contract_version` 与各文件 `sha256`，不匹配即失败退出（不进入训练）。

### 3.2 结果格式（engine → platform），版本 v1

目录：`artifacts/<run_id>/`：

| 文件 | 内容 |
|------|------|
| `manifest.json` | `contract_version, snapshot_run_id, snapshot_sha256, model_version, trained_at, top_n, vocab(映射摘要), metrics(json 摘要)` |
| `recommendations.json` | `{ "<platform_user_id>": [<platform_resource_id>, …topN] }` —— **key/value 均为平台 ID** |
| `vocab.json` | engine 内部编码 ↔ 平台 ID 双向映射（user/item/category/tag），**随产物落盘** |
| `models.pt` | 模型权重（recall+rank） |
| `metrics.json` / `eval_report.json` | 训练/评估指标与门禁结果 |

### 3.3 ID 传递规则（P1 的解法）

1. 训练/推理内部：engine 保留 `build_vocab` 连续编码（embedding 需要），但**编码映射必须落盘 `vocab.json`**。
2. 推理出口：`infer_batch` 输出前用 `vocab` 把编码 ID **回映射为平台 ID**，`recommendations.json` 里不允许出现编码值。
3. platform 导入：key/value 与库中 ID 相等即命中；新增 **coverage 指标**（命中率应 =100%，`skipped > 0` 触发告警，见 §6）。
4. 禁止再出现“sim 自造 ID ↔ 平台 ID 数值碰巧相同才匹配”的隐式规则。

### 3.4 校验点与错误处理

| 环节 | 校验 | 失败动作 |
|------|------|---------|
| 导出 | meta 字段完整性、窗口合法性 | 不产出 run_id |
| engine 入口 | contract_version、sha256 | 退出码非 0，不写 artifacts |
| 训练后 | 训练集/验证集非空、指标可序列化 | 不产出该 run_id 的 recommendations |
| 推理后 | key/value 全部 ∈ 平台 ID 集（回映射断言）、每用户条数=top_n | 产物标记 invalid，禁止导入 |
| 导入 | manifest 校验、coverage 计算、可 dry-run | 默认 fail-closed：不替换缓存 |

---

## 4. platform 侧改造点

### 4.1 数据导出（新增，admin）
- 新增导出服务 + 管理员接口（如 `POST /api/v1/admin/data/export`），按 §3.1 输出快照到配置的 `platform.snapshot_dir`；或提供等价 SQL 脚本供定时任务调用（接口与脚本二选一或并存，推荐先脚本后接口）。
- 导出口径：`resources/` 仅非删除；`behaviors/ratings` 仅窗口内；全表分页流式读取，避免大表内存峰值。
- 一致性：导出时记录窗口边界（`behavior_window`），保证“重复导出不重不漏”的口径可复现。

### 4.2 导入增强（改造现有 `POST /api/v1/admin/recommendations/import`）
- 参数化：`{run_id}`（默认取 artifacts 下最新合法 run_id）＋ `dry_run: bool`。
- 校验：读 `manifest.json`（契约版本、snapshot 一致性）→ 校验通过才允许导入。
- 原子性：每个用户一行 upsert（现状即 replace）；建议在同一事务内完成全量替换或按用户分批替换 + 中途失败可续跑（导入幂等）。
- 审计：导入记录写 `recommendation_imports(id, run_id, imported/skipped 统计, triggered_by, created_at)`。
- 回滚：保留最近 N 个 run_id 的产物；回滚 = 重新导入上一合法 run_id（幂等覆盖）。
- 告警：`skipped_users/skipped_resources > 0`（ID 对齐后应恒为 0）→ 记 error 日志/通知。

### 4.3 曝光打点（M3 落地）
- 新增 `recommendation_exposures(id, user_id, resource_id, position, rec_run_id, action(view/click), created_at)`：
  - `GET /api/v1/recommendations` 返回时写入曝光行（含顺序位）；
  - 前端对推荐流内资源的 view/click 上报带上 `rec_run_id`/`position` 关联。
- 用途：① 上线后效果指标（曝光→点击→收藏漏斗、位置衰减 CTR）；② 下一轮导出的**真负样本**（曝光未点击）；③ 衡量缓存命中与兜底占比。

### 4.4 配置与读侧
- `backend/configs/config.yaml` 增加：
  ```yaml
  engine:
    recommendations_file: "<platform 可读的 artifacts 目录>/<run_id>/recommendations.json"  # 改为目录+run_id 解析
    snapshot_dir: "<platform_snapshot 目录>"
    min_run_interval_hours: 24   # 防抖
  platform:
    snapshot_dir: "<快照输出目录>"
  ```
- `GET /recommendations` 行为不变（命中缓存/热门兜底），但兜底也应视为“非个性化”并在指标里区分。

---

## 5. engine 侧改造点

### 5.1 新增 platform loader（与 `movielens.py` 同级 `platform.py`）
字段映射表（含类型与时间转换）：

| platform 快照 | engine schema | 转换 |
|---|---|---|
| users.user_id | User.user_id | 原样 |
| resources.resource_id | Resource.resource_id | 原样 |
| resources.type | Resource.type | 原样（枚举已一致 course/article/video） |
| resources.category_id | Resource.category_id | 原样（连续编码由 engine vocab 内部完成） |
| resources.tags_json | Resource.tags | `json.loads` → tuple[str] |
| resources.metadata_json | Resource.metadata | `json.loads` → dict（暂不参与训练，留扩展） |
| resources.created_at / view_count / avg_rating | 特征源 | → log_view / avg_rating / age_days（`age_days = (窗口end - created_at)/86400`） |
| behaviors.(user_id,resource_id,action,ts) | Behavior | ts 已 unix 秒 |
| ratings.(user_id,resource_id,score,ts) | Rating | score 1..5 原样 |

### 5.2 修推理侧缺陷（P5）
- `run_batch_infer`/新流水线：数据源由配置/参数决定，**去掉 sim 硬编码**。
- 推理用户集 = 快照**全量用户**；有画像用户走 DSSM 召回，无画像（冷启动）用户走热门兜底（热门口径：窗口内行为频次或 `view_count`，配置可选）——不再依赖“clean 后留在 vocab 的用户”。
- hour/dow：一期若仍无真实请求时间，则**从排序特征中移除**或置为训练分布（禁止训练用真实、推理写死 12/0 的错配）。
- user 数值特征（active_days 等）在推理端从快照统计得到（原样复用训练特征函数），不再填 0。
- 冷启动加权（`rerank.py`）：改为仅对 `age_days <= cold_age_days` 的资源加权，附单测断言（当前“全体 ×1.2”为缺陷）。
- 出口回映射：infer 输出前经 `vocab.json` 映射为平台 ID，并断言全集合法（§3.4）。

### 5.3 流水线入口（消除 P6）
- 新增单次入口：`python -m engine.pipeline.run --snapshot <run_id目录> --out artifacts/<run_id> [--skip-train-if-exists]`
  - 内部：load(platform) → clean(仅训练侧) → vocab → train → eval → infer(全量用户) → 回映射 → 写 manifest/产物。
  - 训练超参与阈值继续走 `EngineConfig`（yaml 可覆盖）。
- 保留 `train_all`/`run_batch_infer`/sim/movielens 轨道作为回归与单测用途，与新入口不共享产物目录，避免覆盖（旧产物命名改为 `models_<track>.pt` 等）。

### 5.4 训练配置（平台轨新增参数）
```
data_source: platform
snapshot_dir: <…>
train_window: 取 meta.behavior_window.end 前 N 天（默认全窗口）
min_user_interactions / min_item_interactions：仅作用于训练集
cold_min_interactions：低于此值的用户走推理兜底（默认=min_user_interactions）
```

---

## 6. 编排、调度与发布（谁来触发谁）

原则：**engine 跑多久、跑没跑成功，platform 不管；platform 只认“合法 run_id 产物”**。

- 建议拓扑：同机或同内网双部署；共享 `snapshot_dir`（platform 写）与 `artifacts_dir`（engine 写）。
- 调度（cron 示例，均为幂等 job）：
  ```
  0 2 * * *  platform: 数据导出 → platform_snapshot/<run_id>/          # 回路①
  0 3 * * 1  engine:    python -m engine.pipeline.run --snapshot 最新   # 每周重训
  0 4 * * *  engine:    python -m engine.pipeline.run --snapshot 最新 --skip-train-if-exists  # 每日推理复用最新模型
  0 5 * * *  platform: 导入最新合法 run_id（dry-run 失败则告警不替换）   # 回路②
  ```
- 灰度：导入支持 `--users <比例/样本>`，先 10% 用户生效观察，再全量。
- 门禁：模型较上一 run_id 的评估指标（见 §7）不满足阈值 → 自动跳过导入并告警（fail-closed）。
- 回滚触发：人工或监控新鲜度告警 → 重导上一 run_id。

---

## 7. 训练与效果验证

| 层 | 指标 | 说明 |
|---|---|---|
| 离线（快照自评） | 召回 Recall@K/HitRate@K、排序 AUC/GAUC/RMSE | 在快照内按用户时间序 80/10/10 切分（engine 已有 `time_split`），**画像统计仅用训练集**（防泄漏规则不变） |
| 有效性 | MovieLens 轨道 | 模型有效性以公开集为基准（sim 指标 `ctr_auc=1.0` 等失真的指标不得作为上线依据） |
| 门禁 | 相对上一 run_id 阈值 | 例如：HitRate@50 不降、RMSE 不升、coverage=100% |
| 上线后（M3） | 曝光→点击率（分位置）、收藏率、兜底占比、缓存新鲜度 | 来自 §4.3 曝光打点；这是“推荐是否有效”的最终裁判 |

---

## 8. 安全与合规

- 快照仅导出业务行为所需字段；`users.csv` 只有 `user_id`，**不含 username/email** 等个人数据。
- 导出/导入接口均要求 admin 鉴权；快照与产物目录文件权限收紧（如 `0700`/属主专用）。
- 快照与产物落盘含 `sha256`，防篡改/防传输损坏。
- 用户注销/资源删除（soft delete）：导出排除，且下一轮导入自然移除其缓存行；曝光表按留存/合规策略定期清理。
- 行为数据属敏感数据：跨机传输建议内网或加密通道。

---

## 9. 部署与运维关注点

- 目录约定：`/opt/edurec/data/snapshots`、`/opt/edurec/data/artifacts`、engine `model/` 仅放训练检查点副本；`dataset/`、`model/`、快照、产物全部纳入备份。
- 监控项（平台已有健康检查 `/api/v1/health`）：导入 coverage=100%？缓存新鲜度（`Recommendation.UpdatedAt` 距今）？兜底占比？导入失败率？
- 告警：import skipped>0、run 失败、快照为空、模型指标回退。
- 配置外置：`recommendations_file` 的 WSL 路径问题随目录化配置消除；密钥/DB 密码一律环境变量注入。

---

## 10. 分期落地（每期独立可验收，遵循仓库提交约定）

### M1：身份连续 + 数据可训（核心，先做）
- platform：数据导出（脚本优先）+ §3.1 快照格式 + 元数据校验单测。
- engine：`platform.py` loader + 出口回映射 + vocab.json 落盘 + 产物 manifest + 修复 P5（hour/dow/user numeric/冷启动加权）+ 流水线入口 `engine.pipeline.run`。
- 验收：用**平台真实导出**快照端到端跑通：导出 → 训练 → 推理 → 回映射 → 导入 → `GET /recommendations` 返回且 ID 全部命中（coverage 100%、skipped=0）；engine 单测含 ID 往返断言。

### M2：导入硬化与自动化
- platform：import 支持 run_id/dry-run/审计表/告警/回滚；调度 cron 落地（§6）。
- 验收：自动导入全流程幂等可重复；人为制造坏产物不污染缓存；回滚演练通过。

### M3：反馈闭环与效果度量
- platform：`recommendation_exposures` 打点 + 前端关联；engine：导出负样本接入曝光未点击（口径文档先行）。
- 验收：能产出“曝光→点击”漏斗周报；下一轮训练消费曝光负样本后指标可对比。

### M4（远期，可选）
- 路线 A：engine REST 服务化（FastAPI）+ ANN 检索 + 缓存未命中实时调用；平台行为实时/准实时回流。

> 每期按两仓库 AGENTS：分支命名 `<type>/<desc>`、单逻辑变更一提交、提交后暂停待确认。

---

## 11. 决策记录（摘要）

| # | 决策 | 理由 |
|---|------|------|
| D1 | 维持两仓库独立（不 submodule/monorepo） | 无共享代码、语言/生命周期不同；对接物收敛为数据契约 |
| D2 | 训练/推理合并为“单快照→单产物”流水线 | 从机制上消除 train/infer 数据分叉（P6） |
| D3 | 全程平台 ID；engine 内部编码必须落盘并出口回映射 | 解决 P1 身份断裂，导入可 100% 命中 |
| D4 | 平台数据以“带版本快照文件”交付 engine，而非 engine 直连 DB | 解耦、可重放、可审计、不共享 DB 凭据；快照即训练集（回答“数据如何变成数据集”） |
| D5 | 全量用户进入推理，清洗只作用于训练集 | 冷启动用户也有推荐（兜底），且导出侧不丢用户 |
| D6 | 导入 fail-closed + run_id 审计 + 覆盖告警 + 灰度 | 避免坏模型污染线上缓存（P4） |
| D7 | 曝光打点先行于任何“效果验证” | 无曝光数据则无法回答“推荐是否有效”，也无真负样本 |
| D8 | 门禁用 MovieLens 轨道与相对回退判断，禁用 sim 失真指标 | sim `ctr_auc=1.0` 类指标只证明流水线通，不证明有效 |
