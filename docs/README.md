# 文档索引

## 架构与契约

| 文档 | 内容 |
|---|---|
| [`design.md`](design.md) | 平台架构、技术栈、决策记录 |
| [`target-architecture.md`](target-architecture.md) | 双仓工程最终形态：角色、部署单元、交互契约与演进路线 |
| [`architecture-constraints.md`](architecture-constraints.md) | 可执行的架构约束与 PR 检查清单 |
| [`architecture-review.md`](architecture-review.md) | 双仓架构评估、目标设计、本轮重构与路线图 |
| [`api-design.md`](api-design.md) | REST API 设计（含推荐读缓存与导入） |
| [`engine-integration.md`](engine-integration.md) | 与 edurec-engine 的边界、数据流、可选在线能力 |
| [`data-handoff.md`](data-handoff.md) | 一轮推荐刷新的完整步骤与契约校验 |
| [`recsys-design.md`](recsys-design.md) | 推荐系统完善方案、现状与剩余路线 |

## 内容与学习

| 文档 | 内容 |
|---|---|
| [`bilibili-import.md`](bilibili-import.md) | B 站采集、字段映射、判重与合规边界 |
| [`dataset-import.md`](dataset-import.md) | 第三方数据集导入（配置驱动字段映射） |
| [`adaptive-learning.md`](adaptive-learning.md) | 知识图谱、初始问卷、学习画像与先修路径 |

## 演示与答辩

| 文档 | 内容 |
|---|---|
| [`demo-walkthrough.md`](demo-walkthrough.md) | 现场演示动线与命令 |
| [`presentation-notes.md`](presentation-notes.md) | 汇报要点与常见追问 |
| [`qa-prep.md`](qa-prep.md) | 答辩问答准备 |

> 契约的单一来源在 engine 仓库的 `contracts/v1/`；platform 的 fixture 副本由
> `scripts/check-contract-sync.sh` 维护。
