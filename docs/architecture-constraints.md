# 架构约束（可执行）

> 配套设计：[`target-architecture.md`](target-architecture.md)
> 适用范围：`edurec-platform` 与 `edurec-engine` 两个仓库的全部改动
> 约束级别：**MUST / MUST NOT 为合并门禁；SHOULD 需在 PR 说明理由**

约束不是建议清单，而是把目标架构变成可检查的规则。每条都写明「怎么检查」；
新增交互方式、改变部署形态、引入新的在线依赖，必须先补一条约束与对应测试。

---

## 1. 仓库与依赖

| ID | 约束 | 理由 | 检查方式 |
|---|---|---|---|
| A1 | **MUST** 两仓不共享代码、不共享数据库；跨仓只通过契约文件/HTTP | 避免隐式耦合，保证 engine 可独立复现 | 代码评审；engine `tests/test_architecture.py`；platform 不 import engine 路径 |
| A2 | **MUST NOT** platform 加载模型权重、执行 Python 推理、读取 engine 目录 | platform 是在线业务系统，不是模型运行时 | 评审 + 依赖检查；`knowledge.source_dir` 不指向 engine |
| A3 | **MUST NOT** engine 访问 platform 数据库或调用 platform HTTP 接口 | 单向数据流；引擎无平台凭证 | 评审；engine 离线链路无网络 import |
| A4 | **MUST** engine 离线链路（`src/engine`、train/infer 入口）不 import HTTP/服务端模块，也不 import `src/kg` | 离线交付必须可在无网络环境运行 | **自动**：engine `tests/test_architecture.py::test_offline_engine_has_no_network_or_kg_imports` |
| A5 | **MUST** 只有 `scripts/kg_server.py` 是网络入口；离线脚本不得启动 HTTP 服务 | 保持部署单元边界 | **自动**：`test_only_kg_server_wrapper_is_a_network_entrypoint` |
| A6 | **MUST** KG 生产路径（server 之外）不 import `src/engine` | 两个能力域不共享实现 | **自动**：`test_kg_production_path_does_not_import_engine` |

---

## 2. 交互与契约

| ID | 约束 | 理由 | 检查方式 |
|---|---|---|---|
| B1 | **MUST** 批量交接只走 C1：快照进、推荐 ID 出；不得新增平台↔engine 的数据库/HTTP 旁路 | 契约是唯一耦合面 | 评审；`docs/platform-contract.md` 变更流程 |
| B2 | **MUST** 工件不可变、带 `run_id`、带校验和、原子写出 | 可复现、可追溯、防半份文件 | engine 校验器 + `write_handoff`；`handoff.sh` 校验 |
| B3 | **MUST** 导入幂等；失败保留旧缓存；不交付半份工件 | 在线服务始终有可用结果 | platform 导入测试 + 契约测试 |
| B4 | **MUST** 推荐工件使用平台原始 ID，禁止引擎内部编号 | 平台不感知模型内部空间 | 契约校验器 / 导入统计 |
| B5 | **MUST** 契约变更先改 engine `contracts/v1`，再同步 platform fixtures | 单一来源，避免两侧漂移 | `scripts/check-contract-sync.sh` |
| B6 | **MUST** 读取方对未知 `contract_version` 明确失败或降级，不得猜测 | 防止静默错误 | engine 校验器；platform KG 客户端版本检查 |
| B7 | **MUST** 在线 KG 调用带 `Authorization`、`X-Request-ID`、超时、并发上限、熔断 | 保护在线服务与平台资源 | platform `concept_generator_test.go`；engine `test_kg_server.py` |
| B8 | **MUST NOT** 对 KG 生成做自动无限重试 | 生成昂贵且可能非确定性；重试会放大故障 | 评审 + 客户端实现（当前无重试） |

---

## 3. 离线 / 在线边界

| ID | 约束 | 理由 | 检查方式 |
|---|---|---|---|
| C1 | **MUST** 离线推荐是主链路；KG 不可用不影响推荐、搜索、学习路径 | 旁路能力不能拖垮主业务 | 降级矩阵评审；前端/服务测试 |
| C2 | **MUST** `engine-batch` 不接收任何入站 HTTP 流量 | 它是 Job，不是服务 | 部署清单评审（无 Service/Ingress） |
| C3 | **MUST** `engine-kg` 只在内网暴露；生产启用服务 token | 防止公网滥用 | 部署清单 + 配置检查（host/token） |
| C4 | **MUST** KG 服务无状态、不落库、不写 platform | 可水平扩展、失败可丢弃 | 代码评审 |
| C5 | **MUST** KG 请求体、批量条数、并发数有硬上限 | 防止单请求打满 CPU/内存 | engine `ServiceConfig` 默认值 + 测试 |
| C6 | **MUST** KG 提供 liveness 与 readiness 两个探针；readiness 检查模型就绪标记 | 部署时正确摘流 | engine `/health`、`/ready` + 测试 |

---

## 4. 数据与安全

| ID | 约束 | 理由 | 检查方式 |
|---|---|---|---|
| D1 | **MUST** 快照/日志不含个人信息；只含 ID、行为、评分、资源正文 | 数据最小化 | 快照列定义 + 评审 |
| D2 | **MUST** 密钥（JWT、KG token、数据库密码）通过环境变量/secret manager 注入，不写仓库 | 防泄露 | 配置评审；`config.yaml` 中 token 默认为空 |
| D3 | **MUST** 在线入口做输入大小与类型校验，不抓取用户提供的任意 URL | 防 SSRF/资源耗尽 | engine `_read_json` + 测试 |
| D4 | **MUST** 日志不打印完整用户描述/画像原文 | 降低隐私暴露面 | 评审；KG 日志只记录 request_id/路径/耗时 |

---

## 5. 运行时与部署

| ID | 约束 | 理由 | 检查方式 |
|---|---|---|---|
| E1 | **MUST** `platform-api`、`engine-kg`、`engine-batch` 分别部署、分别配置资源与扩缩容 | 失败域与资源画像不同 | 部署清单评审 |
| E2 | **MUST** `engine-batch` 可重复执行且幂等；失败不产生可被导入的半成品 | 调度器可能重试 | 工件原子写 + 出口校验 |
| E3 | **MUST** 工件存储/共享卷是唯一交接介质；生产不依赖相邻目录 | 可移植、可审计 | 部署评审；P2 路线图 |
| E4 | **MUST** 在任务状态外置到 Redis/DB 之前，`platform-api` 只能单副本 | 进程内 `jobs` map 无法跨副本一致 | 见 A11（下方） |
| E5 | **SHOULD** 为离线任务与 KG 服务分别设置资源配额与超时，避免互相争抢 | 稳定性 | 部署清单评审 |

---

## 6. 变更管理

| ID | 约束 | 理由 | 检查方式 |
|---|---|---|---|
| F1 | **MUST** 新增交互方式/部署单元/外部依赖，先补架构文档 + 契约 + 测试，再实现 | 防止架构漂移 | PR 模板/评审 |
| F2 | **MUST** 合并前运行 `bash scripts/verify-all.sh --with-frontend` | 统一门禁 | CI/本地脚本 |
| F3 | **MUST** 契约变更同时更新 `docs/platform-contract.md` 或 `docs/kg-service-contract.md` | 文档与实现一致 | 评审 |
| F4 | **SHOULD** 每个 PR 只做一个逻辑变更；破坏性变更单独提交并标注 | 可回滚、可审计 | 评审 |
| F5 | **MUST NOT** 把 `docs/` 里的设计文档当作可选项；与代码冲突时以文档中的约束为准并先修文档 | 文档是架构的一部分 | 评审 |

---

## 7. 当前已落实的自动检查

| 检查 | 位置 | 覆盖约束 |
|---|---|---|
| 离线纯度 / KG 边界 | `edurec-engine/tests/test_architecture.py` | A4, A5, A6 |
| KG 服务鉴权/限额/就绪/请求 ID | `edurec-engine/tests/test_kg_server.py` | B7, C5, C6 |
| KG 客户端鉴权/版本/熔断 | `edurec-platform/backend/internal/service/concept_generator_test.go` | B6, B7, B8 |
| 契约 fixture 同步 | `edurec-platform/scripts/check-contract-sync.sh` | B5 |
| 契约 schema/fixtures 自校验 | `edurec-engine/contracts/v1/validate.py` + `tests/test_contract.py` | B2, B4, B6 |
| 全链路门禁 | `edurec-platform/scripts/verify-all.sh` | F2 |

## 8. 尚未自动化的约束（路线图）

- A1/A2/A3：目前靠评审；可在 CI 增加「禁止路径引用」检查（如 grep engine 路径）。
- E3：工件存储抽象尚未落地（P2）。
- E4：任务状态外置尚未落地（P3）。
- D2：密钥扫描尚未接入 CI；建议增加 secret scanning。

---

## 9. PR 检查清单（复制到评审）

- [ ] 是否新增了跨仓交互？若是，契约与文档是否先更新？
- [ ] 是否让离线链路依赖了在线服务/网络？
- [ ] 在线调用是否有超时、并发上限、熔断、鉴权、request id？
- [ ] 新增字段/接口是否保持向后兼容？破坏性变更是否升 `contract_version`？
- [ ] 是否引入 PII 或把密钥写进仓库？
- [ ] 是否更新了 `docs/target-architecture.md` / 约束文档？
- [ ] `verify-all.sh --with-frontend` 是否通过？
