# 文档地图

> 先读 4 份核心文档；细节按目录查阅。文档与代码冲突时，以
> [`contracts/`](contracts/) 的契约为准。

## 核心（按顺序）

| 文档 | 回答什么问题 |
|---|---|
| [`architecture.md`](architecture.md) | 系统的工程最终形态：platform / engine 的角色、部署单元、交互契约 |
| [`status.md`](status.md) | 现在能做什么、不能做什么、已知约束与路线图 |
| [`backlog.md`](backlog.md) | 待实现清单（跨仓）、优先级与建议执行顺序 |
| [`constraints.md`](constraints.md) | MUST / MUST NOT 约束与 PR 检查清单 |
| [`contracts/README.md`](contracts/README.md) | 与 engine 的两条契约（C1 离线工件 / C2 在线 KG） |

## 契约与交接

| 文档 | 说明 |
|---|---|
| [`contracts/engine-integration.md`](contracts/engine-integration.md) | C1 接入形态、配置、快照与导入 |
| [`contracts/README.md`](contracts/README.md) | C1/C2 总览与变更流程 |
| [`guides/data-handoff.md`](guides/data-handoff.md) | 一轮刷新的完整步骤与校验 |
| engine 契约的单一来源 | `../edurec-engine/contracts/v1/`（schema + fixtures + 校验器） |

## 设计细节

| 文档 | 说明 |
|---|---|
| [`design/architecture-decisions.md`](design/architecture-decisions.md) | 平台架构、技术栈与决策记录 |
| [`design/api-design.md`](design/api-design.md) | REST API 设计 |
| [`design/recsys-design.md`](design/recsys-design.md) | 推荐系统完善方案与实施状态 |
| [`design/adaptive-learning.md`](design/adaptive-learning.md) | 知识图谱、问卷、学习画像与先修路径 |
| [`design/architecture-review.md`](design/architecture-review.md) | 双仓架构评估、重构记录与路线图 |
| `DESIGN.md`（UI 分支） | 前端设计系统；位于 `feat/ui-redesign` 分支 |

## 操作指南

| 文档 | 说明 |
|---|---|
| [`guides/data-handoff.md`](guides/data-handoff.md) | 快照 → engine → 导入 的完整流程 |
| [`guides/bilibili-import.md`](guides/bilibili-import.md) | B 站采集、字段映射、判重与合规边界 |
| [`guides/dataset-import.md`](guides/dataset-import.md) | 第三方数据集配置化导入 |
| [`guides/demo-walkthrough.md`](guides/demo-walkthrough.md) | 现场演示动线 |
| [`guides/presentation-notes.md`](guides/presentation-notes.md) | 汇报要点 |
| [`guides/qa-prep.md`](guides/qa-prep.md) | 答辩问答准备 |

## 推荐阅读路径

- **第一次接触**：根 [`README.md`](../README.md) → 本文 → `architecture.md` → `status.md`
- **后端/推荐**：`architecture.md` → `contracts/README.md` → `design/recsys-design.md` → `design/api-design.md`
- **前端**：UI 分支 `DESIGN.md` → `design/api-design.md`
- **运维/交接**：`contracts/README.md` → `guides/data-handoff.md` → `constraints.md`

## 文档维护规则

1. 顶层只放核心文档；细节下沉到 `design/`、`guides/`、`contracts/`；
2. 新增文档必须在本页登记；
3. 契约变更先改 engine `contracts/v1`，再同步本仓 fixtures 与文档；
4. 历史计划/存档不放在 `docs/` 顶层，需要时从 git 历史取。
