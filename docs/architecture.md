# 目标架构：platform 与 engine 的工程最终形态

> 状态：设计基线（2026-11-12）
> 关联文档：[`architecture-constraints.md`](constraints.md)（可执行约束）、
> [`architecture-review.md`](design/architecture-review.md)（现状评估与已落地重构）
> engine 侧组件视图：`edurec-engine/docs/architecture.md`

---

## 0. 结论先行

1. **两个仓库，不等于两个进程。** 工程最终形态是 **三类可部署单元**：
   - `platform-api`（Go + Vue 静态资源）：在线业务系统，唯一事实来源；
   - `engine-batch`：离线训练/推理 **Job**，不是常驻服务，不接收入站流量；
   - `engine-kg`：可选的在线知识生成服务，独立部署、独立失败、独立扩缩容。
2. **engine 的定位从「纯离线」修正为「离线为主 + 可选在线」**。离线推荐仍是唯一主链路；
   在线 KG 是旁路能力，必须满足独立部署、鉴权、限额、熔断与降级约束。
3. **交互只有两条契约**：离线工件契约（C1）与在线 KG 契约（C2）。
   目前不引入在线推荐（C3）；若未来引入，必须作为新的可部署单元和独立契约评审。
4. **platform 不得因为 KG 在线化而变成「会跑模型的平台」**：模型/算法/权重始终在 engine。
5. **当前平台的任务状态在进程内存**（`ConceptGenerator.jobs`）。在把它外置到 Redis/DB 之前，
   `platform-api` 只能单副本运行；这是本设计明确写死的约束，而不是隐含假设。

---

## 1. 角色与职责

| 角色 | 部署形态 | 职责 | 明确不做 |
|---|---|---|---|
| `platform-api` | 常驻服务，可水平扩展（见约束 A11） | 业务数据治理、用户/行为/反馈、推荐结果消费、KG 调用、运营看板 | 不训练模型、不加载模型权重、不直接执行 Python、不直连 engine 数据库/目录 |
| `engine-batch` | 定时/触发的 Job（CronJob、Argo、CI Job、手动） | 消费快照、训练、全量推理、产出推荐工件、出口契约校验 | 不提供 HTTP、不接收用户请求、不写 platform 数据库 |
| `engine-kg` | 常驻服务，内部网络 | 标题/描述 → 概念，无状态；可选在线能力 | 不落库、不写 platform、不参与推荐主链路、不对外网暴露 |
| `artifact store` | 对象存储 / 共享卷 / 文件系统（按环境） | 快照与推荐工件的不可变交接 | 不承载业务查询 |
| `scheduler` | 平台侧或独立调度器 | 触发 `export → batch → import` 一轮刷新 | 不参与在线请求 |

> **仓库边界**：`edurec-engine` 与 `edurec-platform` 不共享代码、不共享数据库。
> 仓库边界由契约、fixture 与 CI 边界测试固化，而不是靠目录约定。

---

## 2. 部署拓扑

### 2.1 本地开发（当前）

```
platform-api :8080 ──文件──> ../edurec-engine/model/...
       │
       └──HTTP(可选)──> engine-kg :8090（本地进程）
```

`scripts/start.sh`、`scripts/handoff.sh` 面向该形态；文件系统就是“工件存储”。

### 2.2 生产目标

```
                    ┌──────────────┐
 Browser ──HTTPS──> │   Ingress    │
                    └──────┬───────┘
                           ▼
                    ┌──────────────┐        ┌────────────┐
                    │ platform-api │──────► │ MySQL/Redis│
                    └──┬────────┬──┘        └────────────┘
                       │        │
             C1 工件    │        │ C2 HTTP（内部网络，带 token）
                       ▼        ▼
                ┌────────────┐  ┌────────────┐
                │ artifact   │  │ engine-kg  │  Deployment，内部 Service
                │ store      │  └────────────┘
                └──┬──────┬──┘
                   │      │
       读快照/写结果 │      │ 读推荐结果
                   ▼      ▼
                ┌──────────────┐
                │ engine-batch │  Job（CronJob/Argo/CI），无入站端口
                └──────────────┘
```

关键网络规则：

- `engine-kg` **只**在集群内网暴露，不挂公网 Ingress；
- `engine-batch` 没有任何入站端口；
- `platform-api` 不直接调用 `engine-batch`，只与 artifact store 和导入接口交互；
- `engine` 不反向调用 `platform`。

---

## 3. 交互契约

### 3.1 C1：离线工件契约（主链路）

权威规范：engine 仓库 `docs/contracts/platform-contract.md`，可执行版本 `contracts/v1/`。

| 项 | 约定 |
|---|---|
| 方向 A | platform → engine：数据快照（CSV + `meta.json`） |
| 方向 B | engine → platform：`recommendations.json` + 可选旁挂信封 |
| 传输 | 本地共享目录 / 对象存储前缀；生产推荐对象存储 |
| 不可变性 | 每次刷新使用新的 `run_id` 前缀；已发布工件只读 |
| 完整性 | `contract_version` + `files_sha256`；写出原子重命名；两侧校验 |
| 幂等性 | 导入按用户覆盖；同一工件重复导入结果一致 |
| 身份 | 平台原始用户/资源 ID；禁止引擎内部编号 |
| 失败语义 | 校验失败即中止；导入失败保留旧缓存；绝不交付半份工件 |
| 数据最小化 | 快照只含 ID、行为、评分、资源正文；不含个人信息 |

**最终形态的增强项**（未在本轮实现，属路线图）：

- 工件以对象存储 URI 标识：`s3://.../snapshots/<run_id>/`、
  `s3://.../recommendations/<run_id>/`；
- 增加一份 `handoff.json` 清单（或复用 `meta.json`）记录输入/输出 URI、校验和、模型版本；
- 工件保留策略与清理任务（按 `run_id` 和天数）。

### 3.2 C2：在线 KG 契约（可选旁路）

权威规范：engine 仓库 `docs/contracts/kg-service-contract.md`。

| 项 | 约定 |
|---|---|
| 接口 | `GET /health`、`GET /ready`、`POST /knowledge/generate`、`POST /knowledge/generate_batch` |
| 状态 | 无状态；不落库；不写 platform |
| 鉴权 | 生产必须启用 Bearer token（`KG_SERVICE_TOKEN` ↔ `KNOWLEDGE_GENERATOR_TOKEN`） |
| 关联 | 透传 `X-Request-ID`；响应带 `X-Contract-Version` |
| 限额 | 请求体 ≤1 MiB、批量 ≤50 条、并发 ≤4（均可配置） |
| 超时 | platform 客户端超时（默认 180s）；engine 单请求并发信号量 |
| 熔断 | platform 连续 3 次 5xx/429/传输失败后熔断 30s，期间快速失败 |
| 重试 | **不自动重试生成**：生成昂贵且可能非确定性；由用户/调度器在冷却后重试 |
| 降级 | KG 不可用只影响“自动生成概念”；推荐、搜索、学习路径不受影响 |
| 版本 | `contract_version=1`；平台拒绝未知版本 |

### 3.3 C3：在线推荐（明确暂不引入）

如果未来需要“请求时实时推荐”，必须：

1. 作为新的可部署单元 `engine-serving`，而不是塞进 `engine-batch`；
2. 定义独立契约（请求/响应、超时、限流、特征来源）；
3. 明确 online/offline 特征一致性方案；
4. 单独评审成本与收益——当前数据量下 batch 已足够。

---

## 4. 数据与状态边界

| 数据 | 所有者 | engine 是否可写 | platform 是否可写 |
|---|---|---|---|
| 用户/资源/行为/评分 | platform MySQL | 否（只读快照） | 是 |
| 推荐缓存 `recommendations` | platform MySQL | 否（通过导入接口间接写） | 是 |
| 模型权重/编码器缓存 | engine 工件目录 | 是 | 否 |
| 概念生成任务状态 | platform（当前进程内存） | 否 | 是 |
| KG 服务的中间文件 | engine 临时目录 | 是（请求结束即删） | 否 |

**约束**：`ConceptGenerator.jobs` 是进程内状态。多副本部署前必须外置到 Redis/DB
（或引入任务队列），否则状态轮询会落到不同副本而失真。见约束 A11。

---

## 5. 可靠性与降级矩阵

| 故障 | 影响 | 系统行为 |
|---|---|---|
| `engine-kg` 宕机 | 自动生成概念不可用 | 熔断/超时快速失败；其余功能正常；任务状态 `failed` |
| `engine-kg` 变慢 | 生成任务变慢 | 并发上限 + 超时；不阻塞推荐/搜索请求 |
| 快照校验失败 | 本轮刷新中止 | 旧推荐继续服务；`handoff.sh` 非零退出 |
| `engine-batch` 失败 | 本轮刷新中止 | 旧推荐继续服务；告警；不导入半成品 |
| 推荐导入失败 | 缓存不更新 | 旧缓存保留；接口继续返回旧结果或兜底 |
| `platform-api` 宕机 | 业务不可用 | 与 engine 无关；恢复后从缓存/DB 继续 |
| 工件存储不可用 | 刷新链路中断 | 在线服务不受影响（读缓存） |

**原则**：在线链路永远不因离线/旁路依赖失败而整体不可用。

---

## 6. 安全与数据治理

- **网络**：`engine-kg` 仅内网；`engine-batch` 无入站；artifact store 通过最小权限凭证访问。
- **认证**：`platform → engine-kg` 使用服务 token（生产建议 mTLS 或服务网格）；
  token 通过 secret manager / 环境变量注入，不写仓库。
- **授权**：推荐导入是管理员接口；快照导出与导入由运维/调度器执行。
- **输入**：KG 请求体大小、批量条数、字段长度都有硬上限；不抓取任意 URL。
- **数据最小化**：快照不含个人信息；日志不打印用户画像原文；KG 日志不记录完整描述。
- **密钥轮换**：token 支持双值过渡（新 token 生效后旧 token 保留一个部署周期）。

---

## 7. 可观测性

| 维度 | 要求 |
|---|---|
| 关联 ID | `run_id` / `snapshot_run_id`（离线）；`X-Request-ID`（在线） |
| 日志 | 结构化 JSON；离线任务输出输入/输出 URI、行数、耗时、校验结果；KG 输出 request_id、路径、耗时、状态 |
| 指标 | 离线：任务耗时、行数、导入/跳过计数；在线：KG QPS/延迟/错误率/熔断状态 |
| 告警 | 刷新失败、导入跳过率异常、KG 错误率/熔断、工件校验失败 |
| 健康 | `engine-kg`：`/health`（liveness）、`/ready`（readiness） |

---

## 8. 版本与兼容策略

三个版本号不要混用：

| 版本 | 载体 | 变更含义 |
|---|---|---|
| `contract_version` | 快照 `meta.json`、推荐信封、KG 响应 | 交接/接口形状 |
| 包版本 | engine/platform 的发布版本 | 代码与部署 |
| 模型版本 | 信封 `model.version`、checkpoint | 算法与权重 |

规则：

1. **加法兼容**：新增可选字段、提高质量、调整排序不升 `contract_version`；
2. **破坏性变更**：删字段/改类型/改路径/改错误语义 → 升版本，两侧同步，保留一个版本的兼容期；
3. **未知版本**：读取方必须明确失败或降级，不得猜测；
4. **fixture 单一来源**：engine `contracts/v1` 是唯一来源，platform 副本由
   `scripts/check-contract-sync.sh` 维护；
5. **变更流程**：契约变更先改 engine 的 schema/fixtures，再同步 platform，最后跑 `verify-all.sh`。

---

## 9. 演进路线

| 阶段 | 目标 | 验收 |
|---|---|---|
| P0（当前） | 双仓 + 文件交接 + 可选 KG | 现有 `handoff.sh` / 契约测试 |
| P1（本轮） | 在线 KG 边界硬化：鉴权、限额、并发、熔断、请求 ID、readiness；架构约束与边界测试 | `verify-all.sh` 通过；新增 KG 边界测试 |
| P2 | 工件存储抽象与调度器：对象存储 URI、manifest、保留策略 | 不依赖相邻目录也能完成一轮刷新 |
| P3 | platform 任务状态外置（Redis/DB）+ 多副本 | 两个 platform-api 副本轮询同一任务状态一致 |
| P4 | 如确需在线推荐，新增 `engine-serving` 与独立契约 | 单独的 ADR、压测、成本评估 |

---

## 10. 关键决策摘要（ADR 风格）

1. **ADR-001 一个 engine 仓库，两个可部署单元**：离线 batch 与在线 KG 共享代码与模型资产，
   但进程、镜像、资源配额、扩缩容、失败域分开。
2. **ADR-002 离线主链路不可依赖在线能力**：离线训练/推理不得 import HTTP 服务；
   由 `tests/test_architecture.py` 守护。
3. **ADR-003 在线 KG 失败必须可降级**：超时、并发上限、熔断、未知契约版本拒绝；
   KG 不可用只影响自动生成概念。
4. **ADR-004 交接只走工件**：platform 与 engine 不共享数据库/目录；生产用不可变工件 +
   校验和 + 原子写；导入幂等。
5. **ADR-005 版本分离**：contract / package / model 三个版本独立演进，破坏性变更升契约版本。
6. **ADR-006 多副本前先外置任务状态**：进程内 `jobs` map 是已知的单副本约束。
