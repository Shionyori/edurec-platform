# 待实现清单（Backlog）

> 更新：2026-11-12
> 这是**跨仓总清单**：platform、engine、部署与研究的待办都在这里。
> engine 仓库有一份只含 engine 自身的子集：`edurec-engine/docs/backlog.md`。
> 关联：[`architecture.md`](architecture.md)、[`constraints.md`](constraints.md)、[`status.md`](status.md)

## 状态图例

| 状态 | 含义 |
|---|---|
| **待实现** | 设计已完成，只差编码/部署 |
| **待设计** | 方向已定，方案还需评审 |
| **研究** | 需要实验/数据验证，不承诺交付时间 |
| **已完成** | 本轮或之前已闭环，仅作对照 |

---

## 0. 本轮已闭环（不计入待办）

- C1 契约可执行化：schema + fixtures + 校验器 + 两侧同步；
- C2 在线 KG 边界：鉴权、限额、并发、熔断、请求 ID、readiness；
- 推荐导入批量化 + 事务化；handler→repository 分层修复；
- 离线/在线架构边界测试、架构文档与约束；
- 文档分层（core / contracts / design / guides）。

---

## 1. 生产就绪（P1，优先）

| ID | 事项 | 状态 | Owner | 验收标准 | 依赖 |
|---|---|---|---|---|---|
| SYS-01 | **工件存储抽象**：快照/推荐以对象存储 URI（S3/MinIO）交接，增加 manifest 与保留策略 | 待实现 | platform + engine | `handoff.sh` 不依赖相邻目录；`run_id` 前缀不可变；可按天清理 | 无 |
| SYS-02 | **部署单元与镜像**：`platform-api`、`engine-batch`、`engine-kg` 各自镜像/入口；batch 无入站端口，KG 仅内网 | 待实现 | platform + engine | 三者在 dev/staging 可独立启动；KG 有 liveness/readiness；batch 为 Job | SYS-01（可选） |
| SYS-03 | **CI 门禁**：`verify-all --with-frontend` + 文档链接检查 + secret scanning | 待实现 | both | PR 在任一检查失败时被阻断 | 无 |
| SYS-04 | **可观测性**：batch 运行摘要、导入统计、KG 延迟/错误/熔断指标与告警 | 待设计 | both | 结构化日志 + 基础指标；刷新失败/KG 熔断可告警 | SYS-02 |
| SYS-05 | **密钥管理**：JWT / KG token / DB 密码走 secret manager，支持轮换 | 待设计 | platform | 仓库无明文密钥；token 可双值过渡 | 无 |

---

## 2. 规模化与正确性（P2）

| ID | 事项 | 状态 | Owner | 验收标准 | 依赖 |
|---|---|---|---|---|---|
| SYS-06 | **platform 任务状态外置 + 多副本**：概念生成任务状态从进程内存迁到 Redis/DB | 待实现 | platform | 两个 `platform-api` 副本轮询同一任务状态一致；无进程内 map | 无 |
| PF-01 | `export_snapshot` 流式写出，避免全量行累积内存 | 待实现 | platform | 内存峰值与行数解耦；meta 计数/校验和不变 | 无 |
| PF-02 | 移除 handler 对 `repository.*Query` DTO 的依赖，改为 service 入参 | 待实现 | platform | `handler/` 不再 import repository（中间件管理权限除外） | 无 |
| PF-03 | 平台兜底推荐排除「用户已交互」资源 | 待实现 | platform | fallback 注入行为仓储并过滤 seen；有单测 | 无 |
| PF-04 | 推荐缓存新鲜度/TTL：读取信封 `generated_at`，超期走兜底并告警 | 待设计 | platform | 可配置 TTL；接口能反映结果新鲜度 | 无 |
| PF-05 | 版本化数据库迁移（golang-migrate）替代 AutoMigrate | 待实现 | platform | 空库可迁移、重复迁移幂等、可回滚 | 无 |
| PF-06 | 全局请求 ID 中间件：入站 `X-Request-ID` 注入 context、日志与响应，KG 调用继承 | 待实现 | platform | 所有请求/日志带 request id；现有 generation handler 逻辑可移除 | 无 |
| PF-07 | 推荐导入 dry-run / 差异报告（可选） | 待设计 | platform | 导入前可见将更新/跳过的用户与资源数 | PF-01 |

---

## 3. 推荐能力（P3，研究/迭代）

| ID | 事项 | 状态 | Owner | 验收标准 | 依赖 |
|---|---|---|---|---|---|
| RES-01 | 多路召回（语义 + 热门 + 协同/内容）与显式去重 | 研究 | engine | 各单路 vs 融合指标可对比；出口去重 100% | 无 |
| RES-02 | 多目标排序（pClick / pComplete / pSatisfy） | 研究 | engine | 精排 AUC/融合指标优于单信号 | RES-01 |
| RES-03 | 重排：多样性 / 难度梯度 / 新鲜度 | 研究 | engine | 去重 100%；多样性/新鲜度可度量 | RES-02 |
| RES-04 | 接入真 transformer 编码器并重调质量权重 | 研究 | engine | `encoder_kind=sentence_transformer` 可跑；指标与权重网格重跑 | 模型/网络环境 |
| RES-05 | 高斯双塔处置：补齐批量推理，或移入 research | 待设计 | engine | 不再存在「能训练、不能交付」的 checkpoint | 无 |
| RES-06 | ANN（faiss）索引 | 待设计 | engine | 资源量增长后接口不变、延迟可控 | 规模触发 |
| RES-07 | 完成信号（complete）采集与建模 | 待设计 | platform + engine | `complete` 进入快照与训练；多目标评估 | 无 |
| RES-08 | 配置分组（Data/Encoder/Recall/Quality）与编码器命名清理 | 待实现 | engine | facade 保持兼容；YAML 分组；旧名保留别名 | 无 |
| RES-09 | 需求文本构造统一到 `BundleAccess` | 待实现 | engine | 三处实现收敛为一处；有等价性测试 | 无 |
| RES-10 | batch 运行摘要（run_id/行数/耗时/校验结果）结构化输出 | 待实现 | engine | 调度器可采集；失败可定位 | 无 |

---

## 4. 前端

| ID | 事项 | 状态 | Owner | 验收标准 | 依赖 |
|---|---|---|---|---|---|
| FE-01 | E2E 冒烟：登录 → 首页推荐 → 详情 → 收藏 | 待实现 | frontend | 1–2 条 Playwright 用例进 CI | SYS-03 |
| FE-02 | 拆分 `SearchPage` / `ResourceDetailPage` 的数据与交互逻辑为 composable | 待设计 | frontend | 页面只保留模板与交互；现有测试不退化 | 无 |
| FE-03 | MSW handlers 按资源域拆分，减少与后端响应漂移 | 待设计 | frontend | handlers 可维护；关键响应复用契约 fixture | 无 |
| FE-04 | 合并 `feat/ui-redesign`（独立 UI 分支） | 待合并 | frontend | UI 分支与架构分支分别合并；`.gitignore` 冲突解决 | 无 |

---

## 5. 明确不做 / 暂不做

| 事项 | 决定 | 原因 |
|---|---|---|
| 在线推荐 serving（C3） | 暂不做 | 当前数据量与 batch 形态足够；引入需新部署单元、独立契约与成本评估 |
| Monorepo | 暂不做 | 双仓边界清晰；只有跨仓改动频率达到每周 ≥1 次再评估 |
| 实时流式训练/推理 | 暂不做 | 离线 batch 已满足刷新周期 |
| 复杂深度排序模型 | 暂不做 | 先完成多路召回 + 浅层多目标 |
| 用户发表评论 | 产品待定 | 当前评论只读（B 站采集） |

---

## 6. 建议执行顺序

1. **SYS-03 CI 门禁**（成本低，先锁住质量）；
2. **PF-01 / PF-05 / PF-06 / PF-03**（平台正确性与内存/迁移）；
3. **SYS-01 工件存储 → SYS-02 部署单元**（生产形态）；
4. **SYS-06 任务状态外置**（多副本前提）；
5. **SYS-04 / SYS-05 可观测性与密钥**；
6. 推荐研究线 `RES-01 → RES-02 → RES-03`，与工程线并行。
