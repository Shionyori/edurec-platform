# 知识图谱与自适应学习

面向「前端知识点导航 + 后端自适应推荐」的完整链路：

```
学科（21 个：数学/物理/化学…）
        │
        ▼
知识点（72 个：高等数学、数学分析、机器学习…）──先修边──► 概念层（MOOCCube + 生成）
        │                                                        │
        └──────────── 资源关联（380+）◄──────────────────────────┘
                       │
        初始问卷 / 日常行为 / 完成路径
                       ▼
                 学习画像（掌握度）
                       ▼
        目标知识点 → 先修闭包 → 学习路径（done/current/locked）
                       ▼
              只推荐当前一节未掌握的资源
```

## 1. 三层知识图谱

### 学科层（主页面）

- 节点：`disciplines`，21 个主要学科（数学、物理学、化学、天文学、地理学、生物学、
  计算机科学、人工智能、经济学、管理学、法学、教育学、心理学、文学、历史学、哲学、
  外国语言、医学、工学、艺术学、环境科学）。
- 每个学科带关键词（JSON 数组），既用于资源匹配，也作为**无资源时自动爬取的搜索词**。
- 学科页展示该学科下的知识点与资源；资源为 0 时进入页面会自动触发 B 站抓取。

### 知识点层

- 节点：`knowledge_points`，72 个知识点（挂在 `discipline_code` 下）。
- 边：`knowledge_point_edges`，19 条先修关系（高等数学 → 数学分析 → 实变函数……）。
- 资源关联：`resource_knowledge_points`，按标题/标签/描述与知识点关键词的加权匹配
  （标题 1.0、标签 0.6、描述 0.3，阈值 0.5），当前 380+ 条。
- 层级由先修边拓扑推导，**有环直接报错**（知识点层必须严格无环）。

### 概念层（MOOCCube + engine 生成）

- 离线数据来源：平台本地的 `backend/data/MOOCCube`（不入库；MOOCCube 是公开数据集，
  由使用者下载/挂载）。engine 侧生成能力在 `src/kg/`，平台不直接读 engine 目录。
- 规模：425 个概念、1027 条先修边（数学 + 计算机科学技术）。
- 概念没有学科标签，导入时通过「课程 → 概念」反推：
  课程名按学科关键词归类 → 概念按所属课程投票 → 票数最多的学科即归属。
  当前 322 个概念能归入知识点。
- MOOCCube 原始先修图**含环**，概念层级用 SCC 缩点的容错算法（环内同级、环外递增）。

### 数据源路径与重建

路径走配置 `knowledge.source_dir`（默认相对 `backend/`，也支持环境变量
`KNOWLEDGE_SOURCE_DIR` 覆盖）。当前本地指向外部仓库：

```yaml
knowledge:
  source_dir: "data/MOOCCube"
```

换目录只改这一处路径（或用环境变量 `KNOWLEDGE_SOURCE_DIR`），**不改代码**：

```bash
bash scripts/set-kg-source.sh --show
bash scripts/set-kg-source.sh data/MOOCCube
```

图谱规则调整后整体重建（只清图谱派生表，资源/用户/行为不动）：

```powershell
.\scripts\rebuild-kg.ps1              # 全量重建
.\scripts\rebuild-kg.ps1 -DryRun      # 只统计
.\scripts\rebuild-kg.ps1 -KeepOld     # 增量
```

## 2. 学习画像

掌握度（0-1）取以下三者的较大值：

| 来源 | 规则 |
|---|---|
| 初始问卷 | 自评 0/1/2/3 → 0 / 0.33 / 0.67 / 1 |
| 日常行为 | 行为次数 × 0.1，**上限 0.6**（仅看视频不应算已掌握） |
| 完成路径 | 完成某一节后该知识点掌握度置 1 |

掌握度 **≥ 0.8** 视为已掌握：生成路径时直接标 `done`，不再作为待学内容。

## 3. 学习路径算法

1. **先修闭包**：从目标知识点沿先修边反向收集所有前置节点；
2. **拓扑排序**：闭包内按入度出队，同层按 `level`、`id` 稳定排序；有环报错；
3. **生成步骤**：已掌握 → `done`，第一个未掌握 → `current`，其余 → `locked`；
4. **逐节解锁**：只能完成 `current` 步骤，完成后下一节变 `current`；
   完成 `locked` 步骤会被拒绝；全部完成时路径状态置 `completed`。

## 4. API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/disciplines` | 学科列表（主页面，含知识点/资源数） |
| GET | `/api/v1/disciplines/:code` | 学科详情：知识点 + 资源 |
| POST | `/api/v1/disciplines/:code/crawl` | 学科无资源时自动爬取 |
| GET | `/api/v1/disciplines/:code/crawl` | 爬取状态 |
| GET | `/api/v1/knowledge-points` | 知识点列表（资源数、概念数） |
| GET | `/api/v1/knowledge-points/:code` | 知识点先修图 + 概念子图 + 资源 |
| GET | `/api/v1/knowledge-points/:code/resources` | 知识点资源分页 |
| GET | `/api/v1/learning-profile` | 当前用户知识画像 |
| POST | `/api/v1/learning-profile/questionnaire` | 提交问卷 `{answers:{code:0-3}}` |
| POST | `/api/v1/learning-paths` | 生成路径 `{target:"machine-learning"}` |
| GET | `/api/v1/learning-paths/:id` | 路径详情 |
| POST | `/api/v1/learning-paths/:id/complete` | 完成当前节 `{index:1}` |
| GET | `/api/v1/learning-paths/:id/resources` | 当前节资源 |

生成路径示例（问卷：高等数学=3、线性代数=2、概率论=1）：

```json
{
  "target_name": "机器学习",
  "steps": [
    {"name":"高等数学","status":"done","mastery":1},
    {"name":"线性代数","status":"current","mastery":0.67},
    {"name":"数据结构与算法","status":"locked","mastery":0},
    {"name":"概率论与数理统计","status":"locked","mastery":0.33},
    {"name":"机器学习","status":"locked","mastery":0}
  ],
  "current_index": 1,
  "status": "active"
}
```

## 5. 前端页面

| 页面 | 路径 | 说明 |
|---|---|---|
| 学科导航 | `/knowledge` | 21 个学科卡片；无资源的学科标记「自动抓取」 |
| 学科页 | `/discipline/:code` | 学科下的知识点与资源；资源为 0 时自动爬取 |
| 知识点详情 | `/knowledge/:code` | SVG 先修图、概念标签、资源、生成学习路径 |
| 初始问卷 | `/questionnaire` | 0-3 自评，回填已有掌握度 |
| 学习路径 | `/learning-path/:id` | 进度、步骤状态、当前节资源、完成解锁 |

**进入学科页时**，如果该学科资源数为 0，前端会自动调用
`POST /disciplines/:code/crawl`：后端按学科关键词去 B 站搜索 1 页（最多 2 个关键词），
导入资源并按知识点关键词建立关联，完成后刷新页面；有并发去重与 30 分钟冷却。
已有资源时不会重复爬取，必要时可手动点「重新抓取资源」。

图谱视图用纯 SVG + `layoutKnowledgeGraph` 纯函数实现（按 level 分列、同层居中），
不引入图形库。详情页默认只画**当前目标的先修闭包**，保证最右端只有这一个学习目标；
右上角「显示全部学科」可切换到整张图。

## 6. 实时概念生成（engine 侧）

概念不再只靠离线导入：engine 提供 `scripts/kg_server.py`，平台在搜索时实时调用。

```bash
cd <engine 仓库根目录>
.venv/bin/python scripts/kg_server.py --host 127.0.0.1 --port 8090
```

engine 接口（标准库 HTTP，无额外依赖）：

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | 服务状态 |
| POST | `/knowledge/generate` | `{title, description, point_code}` → `{concepts:[{name, page_rank}]}` |
| POST | `/knowledge/generate_batch` | 批量生成 |

平台配置 `knowledge.generator_url`（默认 `http://127.0.0.1:8090`），**只通过 HTTP 调用，
不执行 Python**；生成算法、Wikifier 调用、venv 依赖都留在 engine。

- **搜索即构建**：搜索栏输入关键词 → `POST /knowledge-generation` 匹配最近的知识点并异步生成；
  前端每 2s 轮询 `GET /knowledge-points/:code/generate`，完成后显示「补充 N 个概念 · 查看知识点」。
- **详情页重新生成**：知识点详情「重新生成概念」走同一套接口。
- **批量重建全部学科**：

```powershell
cd backend
CONFIG_PATH=configs/config.local.yaml go run ./cmd/generate_concepts -all
```

- **去重**：同一学科同一概念名只更新 pageRank；新概念 `source=generated`，
  MOOCCube 导入的概念保持 `source=mooc`。
- **质量**：engine 侧按 pageRank 阈值 + 停用词过滤；带描述生成明显更好
  （数学分析 → 积分 / 导数 / 极限 / 连续函数）。

## 7. 关键帧与封面

资源详情页还展示 B 站关键帧（雪碧图 CSS 裁切），详见
[bilibili-import.md](../guides/bilibili-import.md#关键帧与封面资源特征)。

## 8. 已知限制

- 离线概念层只覆盖少量学科；现已用 generate.py 补齐，但概念质量受 Wikifier 影响，
  仍会出现少量泛词（如「数学」「学习」），后续可在 engine 过滤规则里继续加强。
- 行为侧掌握度是简单线性估计（0.1/次，上限 0.6），后续可换成引擎侧的掌握度模型。
- 学习路径尚未合并进首页推荐流；首页仍走 engine 推荐，路径当前节资源在路径页展示。
- 概念归属依赖课程名关键词，课程名缺失时该概念不归属任何学科。
- 学科自动爬取依赖 B 站关键词搜索质量：冷门学科可能抓不到，或抓到的课程相关性一般；
  已做标题/标签/描述关键词匹配，仍建议人工抽查。
