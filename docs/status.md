# 当前设计状态与路线图

> 更新：2026-11-12
> 这是“项目现在能做什么、不能做什么”的唯一摘要；细节见 [`architecture.md`](architecture.md)
> 与 [`design/`](design/)。

## 1. 一句话状态

platform 已具备完整的内容治理、用户反馈、推荐消费与评估闭环；
engine 已从“纯离线”演进为“离线主链路 + 可选在线 KG”；
两者通过 C1 工件与 C2 HTTP 两条契约交互，工程边界已加固。

## 2. 已实现

| 领域 | 状态 |
|---|---|
| 认证/用户/资源/分类/评分/行为/曝光 | ✅ |
| 推荐主链路：快照导出 → engine 训练/推理 → 导入缓存 → 读缓存服务 | ✅ |
| 推荐可追溯与解释：`run_id`、旁挂信封、理由、运行记录、CTR 看板 | ✅ |
| 契约可执行化：`contracts/v1` schema + fixtures + 校验器 + 两侧同步 | ✅ |
| 知识图谱/学科导航/自适应学习/关键帧 | ✅ |
| 外部内容：B 站采集、第三方数据集配置化导入 | ✅ |
| 在线 KG：鉴权、限额、并发、熔断、请求 ID、readiness | ✅（本轮） |
| 架构约束与自动边界测试 | ✅（本轮） |

## 3. 已知约束（不是 bug，是当前工程边界）

1. **`platform-api` 单副本**：概念生成任务状态在进程内存（`ConceptGenerator.jobs`）；
   外置到 Redis/DB 前不能水平扩展（约束 E4）。
2. **工件交接仍依赖本地目录/共享卷**：生产目标的对象存储 URI + manifest 尚未落地（P2）。
3. **在线推荐不提供**：推荐仍是离线批量；请求时实时推荐不在当前范围（C3）。
4. **KG 生成不自动重试**：生成昂贵且可能非确定性；由用户/调度器在冷却后重试。
5. **UI 改版在独立分支**：`feat/ui-redesign`；本架构分支不包含 UI 资产。

## 4. 当前验证基线

| 检查 | 基线 |
|---|---|
| engine pytest | 190 passed（含契约/架构边界/KG 服务测试） |
| platform backend | `go test ./...` + `go vet ./...` 通过 |
| frontend | 166–168 passed（取决于是否含 UI 分支用例）、`vue-tsc`、eslint 通过 |
| 契约 | fixture 自校验 + 两侧同步通过 |

## 5. 路线图

完整待办、Owner、验收标准与建议顺序见 [`backlog.md`](backlog.md)。摘要：

| 阶段 | 目标 | 验收 |
|---|---|---|
| P1（当前） | 在线 KG 边界硬化、架构约束、文档分层 | `verify-all.sh --with-frontend` 全绿 |
| P2 | 工件存储抽象（对象存储 URI + manifest + 保留策略） | 不依赖相邻目录完成一轮刷新 |
| P3 | 任务状态外置 + platform 多副本 | 两个副本轮询同一任务状态一致 |
| P4 | 如确需在线推荐：新增 `engine-serving` 与独立契约 | 单独 ADR、压测、成本评估 |

## 6. 从哪里继续读

- 架构与部署形态：[`architecture.md`](architecture.md)
- 必须遵守的规则：[`constraints.md`](constraints.md)
- 跨仓契约：[`contracts/README.md`](contracts/README.md)
- 详细设计与决策：[`design/architecture-decisions.md`](design/architecture-decisions.md)
- 操作指南：[`guides/data-handoff.md`](guides/data-handoff.md)
