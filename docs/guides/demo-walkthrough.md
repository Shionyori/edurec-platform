# 现场演示动线（platform 侧）

> 适用仓库：`edurec-platform`。engine 是独立仓库，演示时默认在 `../edurec-engine`。
> 本文不绑定某台机器的路径/密码；所有命令都可从仓库根目录执行。

## 0. 一句话总纲

「platform 不做模型推理：个性化推荐来自独立仓库 edurec-engine 的离线结果，
平台只负责数据下链、结果上链，以及在线读缓存与兜底。所以引擎没跑时网站也不会空。」

## 1. 演示前准备（10 分钟）

```bash
# 1) 起容器 + 前后端
bash scripts/start.sh

# 2) 准备内容（二选一）
bash scripts/bilibili.sh        # 真实 B 站教育视频（推荐；首次较慢）
# 或：bash scripts/seed.sh       # 纯模拟数据，仅离线演示/测试

# 3) 准备用户行为（真实内容 + 模拟用户，解决冷启动）
bash scripts/mock-users.sh

# 4) 跑一轮推荐（导出 → 校验 → 训练 → 推理 → 校验 → 导入）
bash scripts/handoff.sh
```

前置检查：

```bash
docker ps                      # MySQL / Redis 容器在跑
curl -s localhost:8080/api/v1/health || true
cd backend && go test ./... >/dev/null && echo "backend ok"
```

账号（由 `demo_seed` / `mock_users` 创建）：`demo_admin / demo123456`（管理员），
`demo1 / demo123456`（普通用户）；以实际脚本输出为准。

## 2. 演示动线（约 12 分钟）

### 第 1 幕 · 首页：推荐来自缓存表（2 分钟）

- 打开 `/`，登录 `demo1`。
- 说：每条推荐都来自 `Recommendation` 表里 engine 产出的有序 ID 列表；
  平台按数组顺序返回，不重排、不打分。
- 看：卡片上的推荐理由（来自信封 `reasons`）。

### 第 2 幕 · 个性化 vs 热门兜底（2 分钟）

- 新注册一个用户（或用一个没有缓存行的账号），刷新首页。
- 说：没有缓存行不会空白——后端按 `avg_rating DESC, id DESC` 兜底并顺手写缓存；
  下一轮导入会用 engine 结果覆盖。
- 决策 #12：服务始终有返回，陈旧但可用。

### 第 3 幕 · 搜索与在线采集（2 分钟）

- 打开搜索页，搜一个本地没有的关键词。
- 说：本地无结果时才在线抓 B 站；结果落库后与普通资源同池，推荐链路不区分来源。
- 注意：在线抓取受 B 站风控，失败不影响其他功能。

### 第 4 幕 · 资源详情 + 反馈埋点（2 分钟）

- 打开一个资源详情，停留/点击/收藏/评分。
- 说：`view` / `click` / `favorite` 落 `user_behaviors`，评分落 `ratings`；
  这些就是下一轮 `export_snapshot` 的训练输入。
- 曝光在独立 `resource_impressions` 表，不污染行为正样本。

### 第 5 幕 · 学科 → 知识点 → 学习路径（2 分钟）

- 进入一个学科页，展示知识点、先修关系、当前可学的一节。
- 说：知识点层严格无环；概念层来自 MOOCCube（平台本地数据），
  原始图有环时用 SCC 缩点容错。

### 第 6 幕 · 管理后台 + 数据闭环（2 分钟）

- 打开 `/admin`，展示推荐运行记录（run_id / 快照 / 模型 / 导入统计）与效果看板
  （曝光/点击/CTR）。
- 终端演示链路（可选）：

```bash
cd backend
CONFIG_PATH=configs/config.yaml go run ./cmd/export_snapshot
# → data/snapshots/<run_id>/（contract_version=1 + sha256）

cd ..
bash scripts/handoff.sh --infer-only   # 用已有模型跑完导出→推理→导入
```

- 说：快照进、排名 ID 出；契约 schema/fixtures/校验器在 engine 的 `contracts/v1`，
  平台用同一份 fixture 做导入测试，`handoff.sh` 交接前会校验。

### 第 7 幕 · 知识生成（可选，1 分钟）

```bash
# engine 仓库
.venv/bin/python scripts/kg_server.py --port 8090
```

- 在平台搜索/知识点详情触发「自动生成概念」。
- 说：这是可选的在线能力，独立契约；engine 未启动只影响该功能。

## 3. 可能被追问

| 问题 | 回答 |
|---|---|
| 平台里有模型吗？ | 没有。无 PyTorch、无 inference；推荐是物化结果 |
| 引擎没跑会怎样？ | 未命中按评分兜底并写缓存；导入失败缓存不变 |
| 推荐多久更新？ | 反映导出时刻快照；下一轮刷新后生效 |
| 怎么证明有效？ | engine 离线指标 + 基线；平台运行记录/CTR/离线评估 |
| 换模型要改平台吗？ | 不改；主文件形状与顺序语义不变即可 |
| 两仓怎么防漂移？ | `contracts/v1` 单一来源 + fixtures + 同步/校验脚本 |

## 4. 出问题时的降级

- 在线抓取失败 → 用已有本地资源继续演示，不影响推荐与学习路径。
- engine 不在场 → 跳过第 6 幕的 `handoff.sh`，只演示「读缓存 + 兜底」。
- 没有真实数据 → `bash scripts/seed.sh` 用模拟数据演示，但明确说明「仅测试」。
