# 答辩/汇报要点（platform 侧）

> 范围：本仓库 `edurec-platform`。`edurec-engine` 是独立仓库，本文只描述两仓的
> **接口契约**与推荐链路，不替它承诺未实现的模型能力。

## 1. 一句话定位

edurec-platform 是教育资源推荐系统的**服务侧与数据治理侧**：把用户、资源、行为、评分、
曝光、知识图谱、学习路径建好，把「算推荐」交给独立仓库 edurec-engine，
自己只做**数据下链（快照导出）**与**结果上链（推荐列表导入 + 缓存读服务）**。

平台**不训练、不推理**；模型权重只留在 engine。两仓交叉接口是**版本化文件契约**，
不是网络调用（唯一的可选在线调用是知识生成服务，见 §6）。

## 2. 架构边界

```
┌──────────┐  HTTP/REST  ┌──────────────┐  文件交接   ┌────────────────┐
│ Frontend │ ◄─────────► │   Backend    │ ◄─────────► │  edurec-engine │
│ Vue 3    │             │ Go + Gin     │  快照/结果   │  离线训练/推理   │
└──────────┘             └──────┬───────┘             └────────────────┘
                                │
                         ┌──────┴───────┐
                         │ MySQL Redis  │
                         └──────────────┘
```

- 推荐主链路：`export_snapshot` → engine 训练/推理 → `recommendations.json` →
  管理员导入 → `Recommendation` 缓存表 → `GET /recommendations`。
- 平台读推荐只查表，**绝不触发 engine 调用**；未命中按评分热门兜底并写缓存。
- 可选在线：`kg_server.py` 只负责「标题/描述 → 概念」，不参与推荐。

## 3. 关键决策（对应 `docs/design.md` 决策记录）

| 编号 | 决策 | 一句话理由 |
|---|---|---|
| 1 / 2 / 3 / 12 | 离线批量 + 文件交接 + 独立 engine | 训练/全量推理是重活，按周期跑；serving 只消费预计算结果 |
| 12 | 平台不做在线推理 | 模型不确定性不进入用户请求路径；模型可随便换 |
| 12 | 未命中本地兜底，不回源 engine | 读路径只有「查表 + FindByIDs」，尾延迟可控 |
| 24 | 资源统一抽象 + JSON metadata | 课程/视频/文章混在同一 ID 空间 |
| 28–32 | 多源内容入池 | B 站采集、第三方数据集、统一判重/分类/截断内核 |
| 34 | 契约可执行化 | `contracts/v1` schema + fixtures + 校验器，两侧同源 |
| 35 | 知识图谱数据源 | MOOCCube 是平台本地外部数据，不指向 engine |
| 36 | 概念生成可选在线 | 独立 HTTP 契约，engine 未启动只影响该功能 |

## 4. 数据流（当前命令）

```bash
# platform 仓库根目录
bash scripts/handoff.sh               # 导出 → 校验快照 → 训练 → 推理 → 校验产物 → 导入
bash scripts/handoff.sh --infer-only  # 复用已有模型
```

手动步骤与校验点见 [`data-handoff.md`](data-handoff.md)。
关键事实：

- 快照带 `contract_version=1`、`run_id`、`files_sha256`；engine 加载前校验。
- 推荐主文件是 `{平台 user_id: [平台 resource_id, …]}`，顺序即排名、已去重、
  覆盖全量用户；旁挂信封可选（`generated_at` / `run_id` / 模型 / 分数 / 理由）。
- 导入按平台库中真实存在的用户/资源过滤；单用户为空或缺席 = 保留旧缓存行。
- 结果反映「截至导出时刻」的快照；新行为要等下一轮刷新。

## 5. 推荐模型（engine，简述）

- **语义双塔**：资源正文 + 由行为历史反推的需求文本 → 同一向量空间 → 内积打分。
- **质量融合**：语义 + 热度（+ 评分），权重按平台数据实测，不是论文默认值。
- 实测（MOOCCube）：纯语义 HitRate@10 0.4432 < 纯热门 0.4930；
  **语义 + 热度融合 0.5758** 才超过热门基线。概率式扩展已实现但默认不启用。
- 已知限制：platform 侧数据量小、行为稀疏；模型收益需要在真实数据上继续验证。

## 6. 可观测性与契约

- **运行记录**：`recommendation_runs` 记录 run_id / 快照 run_id / 模型 / 生成时间 /
  导入统计；`recommendations.run_id` 关联到用户。
- **效果**：`GET /admin/recommendation-stats` 提供曝光/点击/CTR；
  `cmd/evaluate_recommendations` 对比实际推荐与热门基线。
- **推荐解释**：信封 `reasons` 落库并在首页卡片透出。
- **契约**：engine `contracts/v1` 是单一来源；platform
  `scripts/check-contract-sync.sh` 校验 fixture 副本；导入测试用同一份 golden fixture。

## 7. 常见追问

| 追问 | 回答要点 |
|---|---|
| 为什么不让 platform 直接调 engine 推理？ | 训练/全量推理代价高、按周期跑；实时推理会把模型超时/加载失败传导到请求路径。batch 是既定接入方式，在线 serving 列为可选演进 |
| 平台里有模型代码吗？ | 没有。无 PyTorch、无 inference 路径；`Recommendation` 是物化结果表 |
| engine 没跑，首页会空吗？ | 不会。未命中按 `avg_rating DESC, id DESC` 兜底并写缓存；文件空/导入失败则缓存不变 |
| 推荐多久更新？ | 反映导出时刻快照；下一轮完整刷新后生效 |
| 怎么证明推荐变好了？ | engine 离线指标 + 与热门/纯文本基线对比；platform 侧有运行记录与 CTR 看板、离线评估命令 |
| 换模型要改 platform 吗？ | 不改。主文件形状与顺序语义不变即可；契约版本只在破坏性变更时升 |
| 两仓怎么保证不漂移？ | `contracts/v1` 单一来源 + golden fixtures + 双侧校验/同步脚本 + 导入契约测试 |
| 知识图谱数据从哪来？ | MOOCCube 是平台本地外部数据（`backend/data/MOOCCube`），不指向 engine 仓库 |
| 概念是实时生成的吗？ | 可选。engine 的 `kg_server.py` 提供 HTTP 接口；未启动只影响自动生成概念 |
| 数据量太小怎么办？ | 先积累真实行为（B 站采集 + 真实用户/模拟用户），再谈模型收益；当前结论以离线指标为准 |

## 8. 仍需注意的短板

1. **模型收益未被 platform 真实数据证明**：数据量小、行为稀疏；MOOCCube 上纯语义也弱于热门，
   靠融合才超过。
2. **完成信号（complete）未建模**：当前有 view/click/favorite/曝光，缺「学完」。
3. **多路召回/学习排序/重排未做**：engine 侧 P2，研究性工作。
4. **batch 时效**：无定时调度、无增量导出；靠人工/外部触发。
5. **导入保护阈值**：`skipped_*` 异常大时目前只回报计数，不拒绝整批。
