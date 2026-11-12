# 契约总览

platform 与 engine 之间只有两条契约；除此之外不共享数据库、目录或代码。

| 契约 | 方向 | 传输 | 权威文档（engine 仓库） | 本仓说明 |
|---|---|---|---|---|
| **C1 离线工件** | platform ⇄ engine | 快照 CSV / 推荐 JSON | [platform-contract.md](https://github.com/Shionyori/edurec-engine/blob/main/docs/contracts/platform-contract.md) | [`engine-integration.md`](engine-integration.md) |
| **C2 在线 KG** | platform → engine | HTTP/JSON（仅内网） | [kg-service-contract.md](https://github.com/Shionyori/edurec-engine/blob/main/docs/contracts/kg-service-contract.md) | 本文 §2 |

契约的可执行版本在 engine 仓库的 `contracts/v1/`；platform 的 fixture 副本由
`scripts/check-contract-sync.sh` 维护。

## 1. C1：离线工件（主链路）

```
platform 导出快照 → engine 训练/推理 → 推荐结果 → platform 导入缓存
```

- platform 只认「平台 ID 进、平台 ID 出」，与模型实现无关；
- 主文件只能是 `{user_id: [resource_id, …]}`，元信息放旁挂信封；
- 导入幂等、按用户覆盖；失败保留旧缓存；
- 一轮刷新的完整步骤见 [`../guides/data-handoff.md`](../guides/data-handoff.md)。

## 2. C2：在线 KG（可选旁路）

- 只服务「标题/描述 → 概念」；不参与推荐主链路；
- 生产必须启用 Bearer token；platform 侧有超时、并发上限与熔断；
- 不自动重试生成；KG 不可用只影响自动生成概念；
- 平台调用方：`backend/internal/service/concept_generator.go`。

## 3. 变更流程

1. 先改 engine 的 `contracts/v1` schema/fixtures 与权威文档；
2. 同步 platform 的 fixture 副本：`bash scripts/check-contract-sync.sh --update`；
3. 更新本目录与相关实现；
4. 跑 `bash scripts/verify-all.sh --with-frontend`。

## 4. 不在当前契约内

- 在线推荐（C3）：明确暂不引入；
- engine 访问 platform 数据库/HTTP：禁止；
- platform 执行 engine 代码/加载模型：禁止。
