# 答辩问答准备（platform 侧）

> 事实以当前 worktree 为准。引用文件路径便于现场翻代码。
> engine 的内部实现不替它承诺；本文只描述两仓契约与 platform 行为。

## A. 边界与架构

**Q1. 两个仓库什么关系？**
platform（Go + Vue）负责数据治理、在线服务、评估与运营；edurec-engine（Python + PyTorch）
负责离线训练与全量推理。两仓不共享数据库与代码，交叉接口是**版本化文件契约**：
快照进、排名 ID 出。唯一可选在线调用是 engine 的 `kg_server.py`（概念生成），不参与推荐。

**Q2. 为什么不做实时推理？**
训练与全量推理代价高，按周期批量执行；serving 只消费预计算结果，模型超时/加载失败
不会进入用户请求路径。实时服务化列在演进方向，不是当前能力。

**Q3. 平台里有模型代码吗？**
没有。platform 无 PyTorch、无 inference 路径；`Recommendation` 是物化结果表。
见 `docs/contracts/engine-integration.md`、`docs/design/architecture-decisions.md` 决策 #12。

**Q4. engine 没启动，平台会怎样？**
推荐照常：未命中缓存按 `avg_rating DESC, id DESC` 兜底并写缓存；导入失败则缓存不变。
只有「自动生成概念」不可用。见 `docs/contracts/engine-integration.md` §6。

**Q5. 两仓怎么防止契约漂移？**
engine 的 `contracts/v1/` 是单一来源：JSON Schema + golden fixtures + 校验器。
platform 的 `scripts/check-contract-sync.sh` 比对 fixture 副本；
`recommendation_import_contract_test.go` 用同一份样本测导入；`concept_generator_test.go`
用同一份知识生成响应样本。`handoff.sh` 在快照与产物交接前都跑校验。

## B. 数据与契约

**Q6. 快照里有什么？**
`users.csv`（只有 user_id）、`resources.csv`（含 description/difficulty/duration）、
`behaviors.csv`、`ratings.csv`、`categories.csv`、`meta.json`
（`contract_version=1`、`run_id`、行数、`files_sha256`）。

**Q7. 契约版本怎么演进？**
`contract_version=1`。新增可选字段/列不升版本；删除字段、改类型、改键/顺序语义必须升版本，
并两侧同步。engine 读到不认识的版本明确报错退出。

**Q8. 快照损坏/拷贝中断怎么办？**
engine 加载前比对 `meta.json` 的 `files_sha256`，不匹配直接 `ValueError`；
`handoff.sh` 也会先跑校验器，失败即中止。

**Q9. 推荐文件格式为什么这么严格？**
platform 不做容错解析：顶层只能是 `{user_id: [resource_id,…]}`，值必须是非负整数数组，
无 BOM、合法 UTF-8，先写 `.tmp` 再原子重命名。违反任一条整份导入失败（HTTP 500），
避免写入半份错误数据。校验器：`contracts/v1/validate.py recommendations`。

**Q10. 去重/排序/覆盖由谁负责？**
engine 出口负责：按相关性降序、自行去重、覆盖快照全量用户（冷启动走热门兜底）。
platform 端到端保序、不去重、不重排、不截断（展示层 limit 除外）。
见 `docs/contracts/platform-contract.md`（engine 仓库）。

**Q11. 用户/资源 ID 对不上怎么办？**
platform 轨道上 engine 原样透传平台整数 ID。导入时按平台库中真实存在的用户/资源过滤：
用户不存在或过滤后无资源 → 跳过该用户、保留旧缓存；资源不存在 → 只剔除该资源并计数。
只有 `--data-source platform` 的产物可以导入 platform；sim/MOOCube 产物不得导入。

**Q12. 空数组/用户缺席是什么意思？**
「本次不更新该用户」，platform 保留旧缓存行。空数组表达不了「清空」。

## C. 推荐服务

**Q13. 读路径做了什么？**
`GET /api/v1/recommendations`：查 `Recommendation` 缓存行 → 按数组顺序 `FindByIDs` →
返回；未命中按评分热门兜底并写缓存。不调用 engine。

**Q14. 为什么用 MySQL 表而不是 Redis？**
推荐结果是需要持久、审计、按用户批量覆盖写的数据；落库支持 SQL 统计与离线核对。
Redis 在本项目只承载 refresh token。

**Q15. 导入是原子的吗？**
不是跨用户原子：逐用户「删旧行 + 写新行」，中途失败前面已写入的不回滚。
文件本身完整时通常不会走到这里；这条用于解释「部分用户更新了、部分没有」。

**Q16. 冷启动怎么处理？**
两层：engine 为快照里无行为用户输出热门兜底；platform 在缓存未命中时本地兜底。
新用户注册后还有兴趣引导（`users.interests`）。

**Q17. 时效性多差？**
结果反映「截至导出时刻」的快照；新行为/新资源在下一轮刷新后生效。
真实生成时间看信封 `generated_at`，不是导入时刻。

## D. 评估与可观测性

**Q18. 怎么证明推荐有效？**
engine 侧：离线 HitRate/NDCG/MRR 与纯文本/热门基线对比。
platform 侧：`recommendation_runs` 记录 run_id / 快照 / 模型 / 生成时间 / 导入统计；
`GET /admin/recommendation-stats` 给曝光/点击/CTR；`cmd/evaluate_recommendations`
对比实际推荐与热门基线；信封 `reasons` 落库并展示。

**Q19. 推荐结果可追溯吗？**
可以：每次导入写 `recommendation_runs`，`recommendations.run_id` 关联到用户；
信封带 `run_id` / `snapshot_run_id` / 模型版本。

**Q20. 曝光会不会污染行为正样本？**
不会。曝光在独立 `resource_impressions` 表，`user_behaviors` 只记录 view/click/favorite。

## E. 知识图谱与学习路径

**Q21. 知识图谱数据从哪来？**
学科/知识点是平台自有种子数据；概念层从 MOOCCube 导入。MOOCCube 是平台本地外部数据
（`backend/data/MOOCCube`），**不指向 engine 仓库**；为空时只建学科层。

**Q22. 概念是实时的吗？**
可选：engine 的 `kg_server.py` 提供 `/knowledge/generate`；平台只在「自动生成概念」时调用，
失败只影响该功能。契约见 engine `docs/contracts/kg-service-contract.md`。

**Q23. 学习路径与推荐什么关系？**
路径页按先修关系逐节解锁，只推荐当前未掌握内容；首页推荐流仍走 engine 结果。
两者目前是两个入口，不混在同一条排序里。

**Q24. 知识图谱有环怎么办？**
知识点层要求严格无环（有环报错）；MOOCCube 概念图原始数据含环，导入时用 SCC 缩点
（环内同级、环外递增）容错。

## F. 工程与安全

**Q25. 建表用迁移吗？**
当前用 GORM `AutoMigrate`；决策 #9 的 golang-migrate 尚未落地。这是已知工程偏差。

**Q26. 为什么 `source_url` 没有唯一索引？**
当前导入靠单进程串行保证不重复；并发导入可能产生重复行，已列为已知限制。

**Q27. 快照有隐私数据吗？**
`users.csv` 只有 `user_id`，不含 username/email；这是有意的隐私最小化。
交接物只含用户 ID → 资源 ID → 分数/解释这类派生数据。

**Q28. 为什么文档里还有 PowerShell 脚本？**
平台支持 Windows 开发（`.ps1`）与 Linux/macOS（`.sh`），两者功能等价；
新增脚本应同时提供或明确平台。文档命令以 bash 为主。

## G. 已知短板（主动说）

1. 模型收益尚未在 platform 真实数据上证明：数据量小、行为稀疏；MOOCCube 上纯语义弱于热门，
   靠融合才超过。
2. 完成信号（complete）未建模；多路召回/学习排序/重排未做（engine P2）。
3. batch 刷新靠人工/外部触发，无定时任务、无增量导出。
4. 导入没有保护阈值：`skipped_*` 异常大时只回报计数，不拒绝整批。
5. 前端仍以桌面为主；移动端只做了基础适配。

## H. 现场可跑的命令

```bash
# 一键刷新推荐（platform 根目录；engine 默认 ../edurec-engine）
bash scripts/handoff.sh

# 只校验契约 fixture 是否与 engine 同步
bash scripts/check-contract-sync.sh

# 后端测试
cd backend && go test ./... && go vet ./...

# 前端测试
cd frontend && pnpm test && pnpm type-check && pnpm lint
```
