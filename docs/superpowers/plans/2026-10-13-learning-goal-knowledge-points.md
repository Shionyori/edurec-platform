# 学习目标 · 知识点推荐层 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在推荐链条最前面加一层「学习目标 → 知识点 → 每组资源」，让用户从"选方向/搜资源"变成"说出目标"，第一轮交付只读但完整的一条链路。

**Architecture:** 知识点不是新算法，而是对现有推荐信号的一次**分桶**。用户目标经确定性**子串匹配**命中知识点；每个知识点用其**标签**过滤整个资源池（`ResourceRepo.List` 已有能力），再用「engine 推荐序 > avg_rating > view_count」排序取前 6。知识点本轮由 `resources.tags` 聚合而来（零 LLM、零外部依赖），导入契约固定，将来换 LLM 打标时消费方不动。平台仍然不做任何推理。

**Tech Stack:** 后端 Go 1.26 · Gin · GORM(MySQL) · Viper；前端 Vue 3 `<script setup>` · Element Plus · Tailwind v4 · Pinia · Axios · Vitest + MSW。

**Spec:** `docs/superpowers/specs/2026-10-13-learning-goal-knowledge-points-design.md`

## Global Constraints

- **平台侧不做推理。** 个性化结果只来自 engine 离线产出；本功能不得引入任何模型、embedding 或向量库。
- **零新增外部依赖。** 后端不得新增 LLM/HTTP 客户端依赖；前端不得新增 npm 依赖（`el-collapse` 等均出自已装的 element-plus）。
- **配置标签用 `mapstructure:"..."`，不是 `yaml:"..."`。** 本仓 Viper 用 mapstructure 反序列化。
- **本仓没有 testify、没有 mock 框架、没有 sqlmock/sqlite。** 后端测试用标准库 `testing` + 测试文件内手写 fake，包名 `service_test`，且**不是表驱动**。
- **本仓 `internal/handler/` 没有任何测试。** 因此校验逻辑必须下沉到 service 才能被测；handler 只做绑定、序列化与错误映射。（这是对 spec §12「handler 测试」的有意调整。）
- **响应统一走 `response.OK(c, gin.H{...})` / `response.Error(c, httpStatus, code, message)`；错误统一走 `apperror.*`。**
- **中文注释**，与全仓一致。前端代码风格：无分号、单引号、行宽 100（`.prettierrc.json`）。
- **提交粒度**：每个 Task 一次提交，提交后**停下等待用户确认**再继续（`CLAUDE.md`）。提交信息格式 `<type>(<scope>): <subject>`。
- 上限常量：知识点数 `N=8`（服务端最大 20）、组内资源 `M=6`（服务端最大 20）、目标长度 2–100 字符、资源候选池 100（`ResourceListQuery.PageSize` 上限）、推荐信号 50。

## Review Focus

以下五类输入/失效模式是本 spec 隐含、但最可能咬到真实使用者的。每一条都在它所属的 Task 里配了测试；实现时不要绕过。

1. **别名表为空的英文目标 → 零命中兜底。** 首轮知识点由中文标签聚合，`aliases` 全空，所以任何英文目标（"machine learning"）都必然走兜底分支。期望行为：兜底成单一分组 + 明确文案，**不是**空页面、不是 500。
2. **子串匹配不是语义匹配。** "我不想学机器学习了" 仍会命中知识点「机器学习」。这是已知且接受的误召回，必须用测试把行为**钉死**（而不是让人以为它坏了）。
3. **快速连续提交两次目标 → 慢响应覆盖新结果。** 先发的请求若后返回，会把用户刚输入的目标结果冲掉。期望：只有最后一次提交的结果被采用。
4. **`resources.tags` 是 `null` / 非字符串数组 / 损坏 JSON。** 爬虫与数据集导入的标签字段来自外部，必须跳过而不是 panic 或把整批聚合打断。
5. **拉丁文大小写。** 目标里写 "Machine Learning"、知识点别名写 "machine learning"，应当命中；中文无此问题但英文别名有。

---

## File Structure

**后端（新增）**

| 文件 | 职责 |
|---|---|
| `backend/internal/model/knowledge_point.go` | 知识点表模型 |
| `backend/internal/repository/knowledge_point.go` | 知识点数据访问（`ListAll` / `UpsertByName`） |
| `backend/internal/service/knowledge_point_import.go` | 标签聚合**纯函数** + 落库编排 |
| `backend/internal/service/knowledge_point.go` | `PlanForGoal`：匹配 / 分桶 / 排序 / 兜底 |
| `backend/internal/handler/learning_plan.go` | HTTP 绑定与响应整形 |
| `backend/cmd/import_knowledge_points/main.go` | 导入命令（`--from-tags` / `--file` / `--preview` / `--dry-run`） |

**后端（修改）**

| 文件 | 改动 |
|---|---|
| `backend/internal/database/mysql.go:42` | `AutoMigrate` 追加 `&model.KnowledgePoint{}` |
| `backend/internal/config/config.go` | 新增 `KnowledgePointsConfig` + `Config` 字段 + 默认值 + 校验 |
| `backend/internal/router/router.go` | 注册 repo / service / handler 与路由 |
| `backend/configs/config.yaml` | 新增 `knowledge_points:` 段 |

**前端（新增）**

| 文件 | 职责 |
|---|---|
| `frontend/src/api/learningPlan.ts` | `getLearningPlan(goal, knowledgePoints)` |
| `frontend/src/pages/home/GoalInput.vue` | 提问框（校验 + emit） |
| `frontend/src/pages/home/KnowledgePointPanel.vue` | 单个知识点折叠面板 |
| `frontend/src/pages/home/LearningPlanView.vue` | 计划编排：加载 / 空 / 错态 / 兜底文案 / 曝光 |

**前端（修改）**

| 文件 | 改动 |
|---|---|
| `frontend/src/types/index.ts` | 新增 `KnowledgePointSummary` / `LearningPlanGroup` / `LearningPlan` |
| `frontend/src/api/impression.ts` | `ImpressionScene` 联合类型加 `'learning-plan'` |
| `frontend/src/pages/home/index.vue` | 提问框置顶 + 推荐流/计划双状态切换 + URL 同步 |
| `frontend/src/mocks/handlers.ts` | 新增 `GET /learning-plan` handler |
| `frontend/src/pages/home/__tests__/HomePage.test.ts` | 扩展双状态用例 |

---

## 分支与里程碑

| 里程碑 | 分支 | Tasks | 提交信息 |
|---|---|---|---|
| M1 | `feat/knowledge-point-import` | 1–4b | 见表内各 Task |
| M2 | `feat/learning-plan-api` | 5–6 | 同上 |
| M3 | `feat/frontend-learning-plan` | 7–10 | 同上 |
| M4 | `feat/frontend-learning-plan-impression` | 11 | 同上 |
| M5 | `docs/recsys-boundary-revision` | 12 | 同上 |

每个 Task 完成后**停下等用户确认**，这是 `CLAUDE.md` 的硬要求。

---

### Task 1: 知识点模型与数据访问

**Files:**
- Create: `backend/internal/model/knowledge_point.go`
- Create: `backend/internal/repository/knowledge_point.go`
- Modify: `backend/internal/database/mysql.go:42`

**Interfaces:**
- Consumes: 无
- Produces:
  - `model.KnowledgePoint{ gorm.Model; Name string; Description string; Aliases string; CategoryID uint; Category model.Category; Source string; SortOrder int }`
  - `repository.KnowledgePointRepository` 接口，方法 `ListAll() ([]model.KnowledgePoint, error)` 与 `UpsertByName(points []model.KnowledgePoint) (created int, updated int, err error)`
  - `repository.NewKnowledgePointRepository(db *gorm.DB) *KnowledgePointRepo`

> **为什么这个 Task 没有单测**：本仓 `internal/repository/` 下**没有任何测试文件**（全仓测试只存在于 `service/`、`config/`、`middleware/`、`util/`、`cmd/`），仓库层一贯靠编译与上层集成来验证。不要为此发明新范式；本 Task 的闸门是 `go build` + `go vet`，真正的行为验证在 Task 3 与 Task 5。

- [ ] **Step 1: 建分支**

```bash
cd /home/shionyori/project/edurec-platform
git checkout -b feat/knowledge-point-import
```

- [ ] **Step 2: 写模型**

创建 `backend/internal/model/knowledge_point.go`：

```go
package model

import "gorm.io/gorm"

// 知识点来源，用于将来区分产出方（换成 LLM 打标时下游不用改）
const (
	KnowledgePointSourceTag  = "tag"  // 由 resources.tags 聚合而来
	KnowledgePointSourceSeed = "seed" // 人工预置
	KnowledgePointSourceLLM  = "llm"  // 离线 LLM 打标
)

// KnowledgePoint 学习知识点：学习目标被拆解后的一块主题，挂载若干真实资源。
//
// 本轮知识点由资源标签聚合而来（Source = "tag"），Name 与 resources.tags 里的
// 字符串按原值对应。Aliases 既参与「目标文本 → 知识点」的匹配，**也参与资源检索**：
// 服务层用 AnyTags（OR 语义）把 Name 与别名一起作为过滤条件，因此只带别名标签的
// 资源也能被查到（大小写变体与 synonyms 归并进来的写法都落在别名里）。
type KnowledgePoint struct {
	gorm.Model
	Name        string   `gorm:"size:128;uniqueIndex;not null" json:"name"`
	Description string   `gorm:"size:512" json:"description"`
	Aliases     string   `gorm:"type:json" json:"aliases"` // JSON 数组字符串，仅用于目标匹配
	CategoryID  uint     `gorm:"index" json:"category_id"`
	Category    Category `gorm:"foreignKey:CategoryID" json:"category"`
	Source      string   `gorm:"size:32;index" json:"source"`
	SortOrder   int      `gorm:"default:0" json:"sort_order"`
}
```

- [ ] **Step 3: 写数据访问**

创建 `backend/internal/repository/knowledge_point.go`：

```go
package repository

import (
	"errors"
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// KnowledgePointRepository 知识点数据访问接口
type KnowledgePointRepository interface {
	ListAll() ([]model.KnowledgePoint, error)
	UpsertByName(points []model.KnowledgePoint) (created int, updated int, err error)
}

type KnowledgePointRepo struct {
	db *gorm.DB
}

func NewKnowledgePointRepository(db *gorm.DB) *KnowledgePointRepo {
	return &KnowledgePointRepo{db: db}
}

// ListAll 载入全部知识点（含分类）。知识点量级很小（默认上限 60 条），
// 而目标匹配需要拿全量候选，所以不分页。
func (r *KnowledgePointRepo) ListAll() ([]model.KnowledgePoint, error) {
	points := make([]model.KnowledgePoint, 0)
	if err := r.db.Preload("Category").Order("sort_order ASC, id ASC").Find(&points).Error; err != nil {
		return nil, fmt.Errorf("查询知识点失败: %w", err)
	}
	return points, nil
}

// UpsertByName 按 name 幂等写入：已存在则更新展示字段（不新建行，避免重复导入
// 把知识点刷成一堆同义行），不存在则新建。整体在一个事务里，中途失败不留半批。
func (r *KnowledgePointRepo) UpsertByName(points []model.KnowledgePoint) (int, int, error) {
	created, updated := 0, 0
	err := r.db.Transaction(func(tx *gorm.DB) error {
		for i := range points {
			var existing model.KnowledgePoint
			err := tx.Where("name = ?", points[i].Name).First(&existing).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				if err := tx.Create(&points[i]).Error; err != nil {
					return fmt.Errorf("创建知识点 %q 失败: %w", points[i].Name, err)
				}
				created++
			case err != nil:
				return fmt.Errorf("查询知识点 %q 失败: %w", points[i].Name, err)
			default:
				existing.Description = points[i].Description
				existing.Aliases = points[i].Aliases
				existing.CategoryID = points[i].CategoryID
				existing.Source = points[i].Source
				existing.SortOrder = points[i].SortOrder
				if err := tx.Save(&existing).Error; err != nil {
					return fmt.Errorf("更新知识点 %q 失败: %w", points[i].Name, err)
				}
				updated++
			}
		}
		return nil
	})
	return created, updated, err
}
```

- [ ] **Step 4: 挂到 AutoMigrate**

修改 `backend/internal/database/mysql.go`，在 `RecommendationRun{}` 之后追加一行：

```go
	return db.AutoMigrate(
		&model.User{},
		&model.Admin{},
		&model.Category{},
		&model.Resource{},
		&model.UserBehavior{},
		&model.ResourceImpression{},
		&model.Rating{},
		&model.ResourceComment{},
		&model.CommentFetchState{},
		&model.Recommendation{},
		&model.RecommendationRun{},
		&model.KnowledgePoint{},
	)
```

- [ ] **Step 5: 编译与静态检查**

Run:
```bash
cd /home/shionyori/project/edurec-platform/backend && go build ./... && go vet ./...
```
Expected: 无输出（成功）。

- [ ] **Step 6: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add backend/internal/model/knowledge_point.go \
        backend/internal/repository/knowledge_point.go \
        backend/internal/database/mysql.go
git commit -m "feat(backend): 新增知识点模型与数据访问"
```

**提交后停下，等待用户确认。**

---

### Task 2: 知识点配置项

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/configs/config.yaml`
- Test: `backend/internal/config/config_test.go`

**Interfaces:**
- Consumes: 无
- Produces:
  - `config.KnowledgePointsConfig{ TagStoplist []string; MinResourceCount int; MaxKnowledgePoints int }`
  - `config.Config.KnowledgePoints KnowledgePointsConfig`（mapstructure: `knowledge_points`）
  - `(KnowledgePointsConfig) KnowledgePointsOrDefault() KnowledgePointsConfig`
  - 常量 `config.DefaultMinResourceCount = 2`、`config.DefaultMaxKnowledgePoints = 60`、`config.DefaultTagStoplist []string`

- [ ] **Step 1: 写失败的测试**

追加到 `backend/internal/config/config_test.go`。**注意该文件是外部测试包 `package config_test`**（已核实），所以所有导出标识符都要带 `config.` 前缀；文件已 import 了 `testing` 与 `config`，无需新增 import：

```go
func TestKnowledgePointsOrDefault(t *testing.T) {
	t.Run("零值回退到默认", func(t *testing.T) {
		got := config.KnowledgePointsConfig{}.KnowledgePointsOrDefault()
		if got.MinResourceCount != config.DefaultMinResourceCount {
			t.Errorf("MinResourceCount = %d, want %d", got.MinResourceCount, config.DefaultMinResourceCount)
		}
		if got.MaxKnowledgePoints != config.DefaultMaxKnowledgePoints {
			t.Errorf("MaxKnowledgePoints = %d, want %d", got.MaxKnowledgePoints, config.DefaultMaxKnowledgePoints)
		}
		if len(got.TagStoplist) != len(config.DefaultTagStoplist) {
			t.Errorf("TagStoplist 长度 = %d, want %d", len(got.TagStoplist), len(config.DefaultTagStoplist))
		}
	})

	t.Run("显式配置不被覆盖", func(t *testing.T) {
		got := config.KnowledgePointsConfig{
			MinResourceCount:   5,
			MaxKnowledgePoints: 10,
			TagStoplist:        []string{"噪音"},
		}.KnowledgePointsOrDefault()
		if got.MinResourceCount != 5 || got.MaxKnowledgePoints != 10 {
			t.Errorf("显式值被覆盖: %+v", got)
		}
		// 显式给了停用词就用它，不能和默认值合并——否则用户无法「清空」默认表
		if len(got.TagStoplist) != 1 || got.TagStoplist[0] != "噪音" {
			t.Errorf("TagStoplist = %v, want [噪音]", got.TagStoplist)
		}
	})

	t.Run("负数回退到默认", func(t *testing.T) {
		got := config.KnowledgePointsConfig{MinResourceCount: -1}.KnowledgePointsOrDefault()
		if got.MinResourceCount != config.DefaultMinResourceCount {
			t.Errorf("MinResourceCount = %d, want %d", got.MinResourceCount, config.DefaultMinResourceCount)
		}
	})
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/shionyori/project/edurec-platform/backend && go test ./internal/config/ -run TestKnowledgePointsOrDefault -v`
Expected: 编译失败 — `undefined: KnowledgePointsConfig`。

- [ ] **Step 3: 写实现**

在 `backend/internal/config/config.go` 中，紧跟 `EngineConfig` 之后加：

```go
// DefaultTagStoplist 默认标签停用词：这些词在 B 站教育视频标签里高频出现，
// 但没有区分度（几乎每个视频都挂），聚合成知识点只会制造噪音。
var DefaultTagStoplist = []string{
	"教程", "示例", "合集", "全集", "讲解", "课程", "入门", "系列", "视频", "学习",
}

const (
	// DefaultMinResourceCount 标签至少出现在多少个资源上才算知识点
	DefaultMinResourceCount = 2
	// DefaultMaxKnowledgePoints 聚合出的知识点上限
	DefaultMaxKnowledgePoints = 60
)

// KnowledgePointsConfig 学习目标·知识点层配置
type KnowledgePointsConfig struct {
	// TagStoplist 从资源标签聚合知识点时要丢弃的标签
	TagStoplist []string `mapstructure:"tag_stoplist"`
	// MinResourceCount 标签至少要出现在多少个资源上才算知识点
	MinResourceCount int `mapstructure:"min_resource_count"`
	// MaxKnowledgePoints 聚合出的知识点上限
	MaxKnowledgePoints int `mapstructure:"max_knowledge_points"`
}

// KnowledgePointsOrDefault 返回生效的知识点配置，未配置项回退到默认值。
// 显式给了 TagStoplist 就完全采用它（不合并默认值），否则用户没法清空默认停用词。
func (k KnowledgePointsConfig) KnowledgePointsOrDefault() KnowledgePointsConfig {
	out := k
	if out.MinResourceCount <= 0 {
		out.MinResourceCount = DefaultMinResourceCount
	}
	if out.MaxKnowledgePoints <= 0 {
		out.MaxKnowledgePoints = DefaultMaxKnowledgePoints
	}
	if len(out.TagStoplist) == 0 {
		out.TagStoplist = DefaultTagStoplist
	}
	return out
}
```

在 `Config` 结构体末尾（`Datasets` 之后）加一行：

```go
	KnowledgePoints KnowledgePointsConfig     `mapstructure:"knowledge_points"`
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/shionyori/project/edurec-platform/backend && go test ./internal/config/ -v`
Expected: PASS（含既有的其它 config 测试）。

- [ ] **Step 5: 写进 config.yaml**

在 `backend/configs/config.yaml` 的 `content_rules:` 段之前插入：

```yaml
# 学习目标·知识点层（cmd/import_knowledge_points）。
# 本轮知识点由 resources.tags 聚合而来：某个标签至少出现在 min_resource_count
# 个资源上、且不在 tag_stoplist 里，才会成为一个知识点。
# 留空/不写各项 → 用代码内默认值（见 config.DefaultTagStoplist 等）。
knowledge_points:
  # 无区分度的高频标签，聚合成知识点只会制造噪音
  tag_stoplist:
    - 教程
    - 示例
    - 合集
    - 全集
    - 讲解
    - 课程
    - 入门
    - 系列
    - 视频
    - 学习
  # 标签至少要出现在多少个资源上才算知识点
  min_resource_count: 2
  # 聚合出的知识点上限
  max_knowledge_points: 60
```

- [ ] **Step 6: 确认配置能加载**

Run:
```bash
cd /home/shionyori/project/edurec-platform/backend && go vet ./... && go test ./internal/config/ -v
```
Expected: 全部 PASS。

- [ ] **Step 7: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add backend/internal/config/config.go backend/internal/config/config_test.go backend/configs/config.yaml
git commit -m "feat(backend): 新增知识点聚合配置项"
```

**提交后停下，等待用户确认。**

---

### Task 3: 标签聚合（纯函数）

**Files:**
- Create: `backend/internal/service/knowledge_point_import.go`
- Test: `backend/internal/service/knowledge_point_import_test.go`

**Interfaces:**
- Consumes: `model.Resource`（`Tags string` 为 JSON 数组字符串、`CategoryID uint`）
- Produces:
  - `service.TagAggregate{ Name string; Count int; CategoryID uint }`
  - `service.AggregateKnowledgePointsFromTags(resources []model.Resource, stoplist []string, minCount, max int) []TagAggregate`
  - `service.ParseResourceTags(raw string) []string`
  - 常量 `service.MinKnowledgePointNameRunes = 2`

- [ ] **Step 1: 写失败的测试**

创建 `backend/internal/service/knowledge_point_import_test.go`：

```go
package service_test

import (
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

func resourceWithTags(id uint, categoryID uint, tags string) model.Resource {
	r := model.Resource{Tags: tags, CategoryID: categoryID}
	r.ID = id
	return r
}

func TestParseResourceTags(t *testing.T) {
	t.Run("正常数组", func(t *testing.T) {
		got := service.ParseResourceTags(`["机器学习","回归"]`)
		if len(got) != 2 || got[0] != "机器学习" {
			t.Errorf("ParseResourceTags = %v", got)
		}
	})

	// 标签来自爬虫与数据集导入，是外部数据：null、类型不对、JSON 损坏都不能 panic，
	// 也不能漏出空字符串标签。断言 len==0 而不是「逐个查空」——后者在返回空切片时
	// 恒真，等于没断言。
	t.Run("空与损坏一律返回空", func(t *testing.T) {
		for _, raw := range []string{"", "  ", "null", "not json", `{"a":1}`, `[1,2]`, `["ok",3]`} {
			if got := service.ParseResourceTags(raw); len(got) != 0 {
				t.Errorf("ParseResourceTags(%q) = %v, want 空切片", raw, got)
			}
		}
	})

	t.Run("数组里的空白标签被剔除", func(t *testing.T) {
		got := service.ParseResourceTags(`["机器学习","  ",""]`)
		if len(got) != 1 || got[0] != "机器学习" {
			t.Errorf("got %v, want [机器学习]", got)
		}
	})
}

func TestAggregateKnowledgePointsFromTags(t *testing.T) {
	t.Run("按词频聚合，低于阈值与停用词被剔除", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["机器学习","教程"]`),
			resourceWithTags(2, 1, `["机器学习","回归"]`),
			resourceWithTags(3, 2, `["机器学习"]`),
			resourceWithTags(4, 2, `["深度学习"]`),
		}

		got := service.AggregateKnowledgePointsFromTags(resources, []string{"教程"}, 2, 60)

		if len(got) != 1 {
			t.Fatalf("聚合出 %d 个知识点，want 1: %+v", len(got), got)
		}
		if got[0].Name != "机器学习" || got[0].Count != 3 {
			t.Errorf("got %+v, want 机器学习 x3", got[0])
		}
		// 出现最多的分类成为知识点的归属分类（这里分类 1 与 2 各 1 次，取 id 小的）
		if got[0].CategoryID != 1 {
			t.Errorf("CategoryID = %d, want 1", got[0].CategoryID)
		}
	})

	t.Run("同一资源内重复标签只计一次", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["机器学习","机器学习"]`),
			resourceWithTags(2, 1, `["机器学习"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, nil, 2, 60)
		if len(got) != 1 || got[0].Count != 2 {
			t.Fatalf("got %+v, want 机器学习 x2（重复标签不该把词频刷到 3）", got)
		}
	})

	t.Run("短标签被剔除", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["C"]`),
			resourceWithTags(2, 1, `["C"]`),
		}
		if got := service.AggregateKnowledgePointsFromTags(resources, nil, 2, 60); len(got) != 0 {
			t.Errorf("单字符标签不该成为知识点: %+v", got)
		}
	})

	t.Run("损坏标签不打断整批聚合", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `null`),
			resourceWithTags(2, 1, `not json`),
			resourceWithTags(3, 1, `["机器学习"]`),
			resourceWithTags(4, 1, `["机器学习"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, nil, 2, 60)
		if len(got) != 1 || got[0].Name != "机器学习" {
			t.Fatalf("got %+v, want 仅 机器学习", got)
		}
	})

	t.Run("超出上限时按词频截断，同频按名字稳定排序", func(t *testing.T) {
		// 注意：fixture 里的词必须是 2 个 rune 以上。单个汉字只占 1 个 rune，
		// 会被 MinKnowledgePointNameRunes=2 过滤掉（见上一个用例的 "C"），
		// 用单字当 fixture 会让这个用例恒为 0 个候选、断言永不成立。
		resources := []model.Resource{
			resourceWithTags(1, 1, `["甲甲","乙乙","丙丙"]`),
			resourceWithTags(2, 1, `["甲甲","乙乙","丙丙"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, nil, 2, 2)
		if len(got) != 2 {
			t.Fatalf("got %d 个，want 2", len(got))
		}
		// 三次词频相同 → 按名字升序，保证多次运行结果一致（map 迭代本身无序）
		if got[0].Name > got[1].Name {
			t.Errorf("同频未按名字升序: %+v", got)
		}
	})

	t.Run("无分类的资源不影响聚合", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 0, `["机器学习"]`),
			resourceWithTags(2, 0, `["机器学习"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, nil, 2, 60)
		if len(got) != 1 || got[0].CategoryID != 0 {
			t.Fatalf("got %+v, want CategoryID=0", got)
		}
	})
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/shionyori/project/edurec-platform/backend && go test ./internal/service/ -run 'TestParseResourceTags|TestAggregateKnowledgePointsFromTags' -v`
Expected: 编译失败 — `undefined: service.ParseResourceTags`。

- [ ] **Step 3: 写实现**

创建 `backend/internal/service/knowledge_point_import.go`：

```go
package service

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// MinKnowledgePointNameRunes 知识点名的最小长度。单字符（如 "C"）会命中几乎
// 所有学习目标，作为知识点既无意义又会污染匹配，必须在聚合与匹配两侧都挡掉。
const MinKnowledgePointNameRunes = 2

// ParseResourceTags 解析 resources.tags 的 JSON 数组字符串。
// 标签属于外部数据（爬虫 / 数据集导入），null、类型不符、JSON 损坏一律当作
// 空标签处理，绝不让单个脏资源打断整批聚合。
func ParseResourceTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil
	}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			out = append(out, tag)
		}
	}
	return out
}

// TagAggregate 一个标签的聚合结果，即一个候选知识点
type TagAggregate struct {
	Name       string
	Count      int
	CategoryID uint
}

// AggregateKnowledgePointsFromTags 从资源标签聚合知识点候选。
//
// stoplist 内的标签、长度不足 MinKnowledgePointNameRunes 的标签、出现次数
// 少于 minCount 的标签都会被丢弃。结果按词频降序、同频按名字升序排列——
// map 迭代本身无序，不显式定序的话每次运行产出的知识点顺序都不同。
func AggregateKnowledgePointsFromTags(
	resources []model.Resource,
	stoplist []string,
	minCount, max int,
) []TagAggregate {
	stop := make(map[string]struct{}, len(stoplist))
	for _, s := range stoplist {
		stop[s] = struct{}{}
	}

	type accumulator struct {
		count       int
		categoryHit map[uint]int
	}
	accs := make(map[string]*accumulator)

	for i := range resources {
		seen := make(map[string]struct{})
		for _, tag := range ParseResourceTags(resources[i].Tags) {
			if utf8.RuneCountInString(tag) < MinKnowledgePointNameRunes {
				continue
			}
			if _, blocked := stop[tag]; blocked {
				continue
			}
			// 同一资源里重复挂同一个标签只算一次，否则词频会被单个资源刷高
			if _, dup := seen[tag]; dup {
				continue
			}
			seen[tag] = struct{}{}

			acc, ok := accs[tag]
			if !ok {
				acc = &accumulator{categoryHit: make(map[uint]int)}
				accs[tag] = acc
			}
			acc.count++
			if resources[i].CategoryID != 0 {
				acc.categoryHit[resources[i].CategoryID]++
			}
		}
	}

	out := make([]TagAggregate, 0, len(accs))
	for name, acc := range accs {
		if acc.count < minCount {
			continue
		}
		out = append(out, TagAggregate{
			Name:       name,
			Count:      acc.count,
			CategoryID: dominantCategory(acc.categoryHit),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	if max > 0 && len(out) > max {
		out = out[:max]
	}
	return out
}

// dominantCategory 取出现次数最多的分类；并列时取 id 最小的，保证结果确定。
func dominantCategory(hits map[uint]int) uint {
	var bestID uint
	bestCount := 0
	for id, count := range hits {
		if count > bestCount || (count == bestCount && id < bestID) {
			bestID, bestCount = id, count
		}
	}
	return bestID
}

// KnowledgePointImportService 把标签聚合结果落成知识点行
type KnowledgePointImportService struct {
	resources repository.ResourceRepository
	points    repository.KnowledgePointRepository
}

func NewKnowledgePointImportService(
	resources repository.ResourceRepository,
	points repository.KnowledgePointRepository,
) *KnowledgePointImportService {
	return &KnowledgePointImportService{resources: resources, points: points}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/shionyori/project/edurec-platform/backend && go test ./internal/service/ -run 'TestParseResourceTags|TestAggregateKnowledgePointsFromTags' -v`
Expected: 全部 PASS。

- [ ] **Step 5: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add backend/internal/service/knowledge_point_import.go \
        backend/internal/service/knowledge_point_import_test.go
git commit -m "feat(backend): 标签聚合知识点候选"
```

**提交后停下，等待用户确认。**

---

### Task 4: 导入命令

**Files:**
- Create: `backend/cmd/import_knowledge_points/main.go`

**Interfaces:**
- Consumes: `service.AggregateKnowledgePointsFromTags`、`service.NewKnowledgePointImportService`、`repository.NewResourceRepository`、`repository.NewKnowledgePointRepository`、`config.KnowledgePointsConfig.KnowledgePointsOrDefault`、`database.InitMySQL`
- Produces: 可执行命令 `go run ./cmd/import_knowledge_points`

- [ ] **Step 1: 加落库编排到 service（导入命令只做 IO）**

在 `backend/internal/service/knowledge_point_import.go` 文件末尾追加。把编排放在 service 而不是 `main`，是为了让它可被测试，且让命令层保持"解析参数 → 打印"的薄壳：

```go
// KnowledgePointImportResult 一次导入的统计
type KnowledgePointImportResult struct {
	Candidates int
	Created    int
	Updated    int
}

// ImportFromTags 从库内资源的标签聚合知识点并幂等落库。
// write 为 false 时只计算不写库（--dry-run）。
func (s *KnowledgePointImportService) ImportFromTags(
	cfg config.KnowledgePointsConfig,
	write bool,
) (*KnowledgePointImportResult, []TagAggregate, error) {
	effective := cfg.KnowledgePointsOrDefault()

	resources, err := s.loadAllResources()
	if err != nil {
		return nil, nil, apperror.Internal(err)
	}

	candidates := AggregateKnowledgePointsFromTags(
		resources, effective.TagStoplist, effective.MinResourceCount, effective.MaxKnowledgePoints,
	)

	result := &KnowledgePointImportResult{Candidates: len(candidates)}
	if !write || len(candidates) == 0 {
		return result, candidates, nil
	}

	points := make([]model.KnowledgePoint, 0, len(candidates))
	for i, c := range candidates {
		points = append(points, model.KnowledgePoint{
			Name:       c.Name,
			CategoryID: c.CategoryID,
			// Aliases 是 MySQL JSON 列，Go 零值 "" 不是合法 JSON，直接落库会报
			// Error 3140（Invalid JSON text: The document is empty）并回滚整批。
			// 与 service/resource.go 里 tags 默认 "[]"、metadata 默认 "{}" 同一惯例。
			Aliases:   "[]",
			Source:    model.KnowledgePointSourceTag,
			SortOrder: i,
		})
	}

	created, updated, err := s.points.UpsertByName(points)
	if err != nil {
		return nil, nil, apperror.Internal(err)
	}
	result.Created, result.Updated = created, updated
	return result, candidates, nil
}

// loadAllResources 翻页取回全部资源。
//
// 必须翻页：ResourceRepo.List 的 PageSize 上限是 100（repository/resource.go:66-68），
// 只取一页会让资源数超过 100 的库**静默丢掉后面的标签**，聚合出的知识点因人而异却毫无提示。
func (s *KnowledgePointImportService) loadAllResources() ([]model.Resource, error) {
	const pageSize = 100
	all := make([]model.Resource, 0)
	for page := 1; ; page++ {
		list, err := s.resources.List(repository.ResourceListQuery{Page: page, PageSize: pageSize})
		if err != nil {
			return nil, err
		}
		all = append(all, list.Items...)
		if len(list.Items) < pageSize || int64(len(all)) >= list.Total {
			return all, nil
		}
	}
}
```

在同文件 import 块补上 `config`、`apperror`：

```go
import (
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)
```

- [ ] **Step 2: 编译**

Run: `cd /home/shionyori/project/edurec-platform/backend && go build ./...`
Expected: 成功。

- [ ] **Step 3: 写命令**

创建 `backend/cmd/import_knowledge_points/main.go`：

```go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

const prefix = "[import_knowledge_points]"

// defaultPreviewLimit --preview 未指定 --limit 时展示的知识点条数。
// 预览是用来核对聚合质量的，看几十条就够，全量刷屏反而找不着要点。
const defaultPreviewLimit = 30

func main() {
	fromTags := flag.Bool("from-tags", false, "从 resources.tags 聚合知识点（本轮的唯一来源）")
	filePath := flag.String("file", "", "从 JSON 文件导入知识点（手工/LLM 产出，格式见 spec §8.3；本轮未实现）")
	// 注意：--limit 只截断「打印」，不影响落库范围（落库范围由 max_knowledge_points 配置决定）。
	// 文案必须说清这一点——import_dataset 的 --limit 是真截断处理范围，两者语义不同。
	limit := flag.Int("limit", 0, "打印候选知识点时最多展示多少条；--preview 未指定时默认 30")
	// 注意：--from-tags 必须读 resources 表才能聚合，所以是「只读库、不写库」，
	// 与 import_dataset 的 --preview（解析本地文件、完全不连库）语义不同。
	preview := flag.Bool("preview", false, "只读数据库、不写库（优先于 --dry-run）")
	dryRun := flag.Bool("dry-run", false, "连库走完整聚合，但不写库")
	flag.Parse()

	if !*fromTags && *filePath == "" {
		flag.Usage()
		log.Fatalf("%s 必须指定 --from-tags 或 --file 之一", prefix)
	}
	if *fromTags && *filePath != "" {
		log.Fatalf("%s --from-tags 与 --file 互斥，只能选一个", prefix)
	}
	// --file 本轮未实现：在连库之前就拦掉，避免先建立数据库连接再报错
	if *filePath != "" {
		log.Fatalf("%s --file 导入本轮尚未实现，请用 --from-tags", prefix)
	}

	limitSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "limit" {
			limitSet = true
		}
	})
	effectiveLimit := *limit
	if !limitSet && *preview {
		effectiveLimit = defaultPreviewLimit
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("%s 加载配置失败: %v", prefix, err)
	}

	if *preview {
		// 注意：--from-tags 必须读 resources 表才能聚合，所以 --preview「只读库、不写库」，
		// 与 import_dataset 的 --preview（解析本地文件、完全不连库）语义不同。
		fmt.Printf("%s --preview：只读数据库、不写库\n", prefix)
	}

	db, err := database.InitMySQL(cfg.Database)
	if err != nil {
		log.Fatalf("%s MySQL 初始化失败: %v", prefix, err)
	}

	importer := service.NewKnowledgePointImportService(
		repository.NewResourceRepository(db),
		repository.NewKnowledgePointRepository(db),
	)

	result, candidates, err := importer.ImportFromTags(cfg.KnowledgePoints, !*dryRun && !*preview)
	if err != nil {
		log.Fatalf("%s 聚合失败: %v", prefix, err)
	}

	printCandidates(candidates, effectiveLimit)

	if *dryRun {
		fmt.Printf("%s --dry-run：未写库\n", prefix)
	}
	fmt.Printf("%s 聚合出 %d 个候选知识点，新增 %d，更新 %d\n",
		prefix, result.Candidates, result.Created, result.Updated)
	if result.Candidates == 0 {
		fmt.Printf("%s 提示：一个知识点都没聚合出来，检查 resources 表是否有数据、"+
			"以及 min_resource_count 是否过高（当前 %d）\n",
			prefix, cfg.KnowledgePoints.KnowledgePointsOrDefault().MinResourceCount)
	}
}

// printCandidates 打印候选知识点，供人工检视聚合质量（spec §8.4 的阶段 0 闸门）
func printCandidates(candidates []service.TagAggregate, limit int) {
	shown := candidates
	if limit > 0 && len(shown) > limit {
		shown = shown[:limit]
	}
	for i, c := range shown {
		encoded, err := json.Marshal(c)
		if err != nil {
			log.Fatalf("%s 序列化第 %d 个候选失败: %v", prefix, i+1, err)
		}
		fmt.Printf("%s [%d/%d] %s\n", prefix, i+1, len(shown), encoded)
	}
	if len(shown) < len(candidates) {
		fmt.Printf("%s 上面只展示前 %d 个（--limit 可改，共 %d 个）\n",
			prefix, len(shown), len(candidates))
	}
}
```

> **对 spec §8.1 的有意收窄**：`--file` 与「`--from-tags` 先写出 `data/knowledge_points.json` 再导入」本轮**不做**。
> 理由是 YAGNI 加本仓的测试纪律：本轮只有 `--from-tags` 一个产出方，写一个没有第二个产出方、
> 因而无法端到端验证的读取器，只会留下一段没被测过的代码。**缝依然在**——聚合逻辑是独立纯函数
> （Task 3），落库是独立 service 方法（本 Task），将来接 LLM 打标时只需新增一个产出同结构切片的函数。
> Task 12 会把这个收窄回写进 spec。

- [ ] **Step 4: 编译与静态检查**

Run:
```bash
cd /home/shionyori/project/edurec-platform/backend && go build ./... && go vet ./...
```
Expected: 无输出（成功）。

- [ ] **Step 5: 阶段 0 闸门 — 人工检视聚合质量**

先确认服务与数据库已启动（`bash scripts/start.sh`，或已有容器在跑），然后：

```bash
cd /home/shionyori/project/edurec-platform/backend
go run ./cmd/import_knowledge_points --from-tags --preview
```

**这是 spec §8.4 的闸门，必须由用户人工判断产出质量**，而不是自动通过。检视要点：
- 候选项是否**像知识点**（"线性代数""监督学习"），还是**像标签云**（"吴彦祖""B站"）？
- 若噪音占多数，把噪音词加进 `configs/config.yaml` 的 `knowledge_points.tag_stoplist` 后重跑。
- 若加停用词也救不回来，**停下**：说明该换知识点来源（LLM 打标或手工预置），此时应回到 spec 讨论，不要继续 Task 5。

用户确认质量可接受后，真导入一次：

```bash
go run ./cmd/import_knowledge_points --from-tags
```

Expected: 输出形如 `[import_knowledge_points] 聚合出 N 个候选知识点，新增 N，更新 0`，且库中 `knowledge_points` 表有行。

- [ ] **Step 6: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add backend/cmd/import_knowledge_points/main.go \
        backend/internal/service/knowledge_point_import.go
git commit -m "feat(backend): 新增知识点导入命令"
```

**提交后停下，等待用户确认。**

---

### Task 4b: 知识点质量修补（大小写归并 · 停用词 · 别名归并）

> **为什么有这个 Task**：Task 4 的人工闸门实测 60 个候选知识点，上半段是可用的真知识点
> （机器学习 35 / 人工智能 28 / 深度学习 21 / 编程 19 / Python 18），但暴露出三类问题：
> ① 拉丁文大小写未归一把同一个知识点拆成多条（`GO`/`Go`/`go`/`Golang`/`GOLANG`/`Go语言`/`go语言`/`GO语言`
> 共 8 条）。**实测修正**：归并后是 1 条、计数 17、排名第 6（原为第 12）。注意别把 8 种写法的标签出现次数相加得 38——
> 12 个资源同时挂了多种写法，在「同一资源只算一次」的不变式下正确值是去重后的**资源数 17**，38 是重复计数；
> ② 平台徽章/人名/元信息词混入（科技猎手、吴恩达、视频教程…）；③ 语义重叠（机器学习 / 机器学习入门 / 机器学习算法）。
> ① 是缺陷，② 是配置问题，③ 用 `aliases` 解决。用户已就此拍板：修这三处，不换知识点来源。

**Files:**
- Modify: `backend/internal/service/knowledge_point_import.go`
- Modify: `backend/internal/service/knowledge_point_import_test.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/internal/repository/resource.go`
- Modify: `backend/configs/config.yaml`

**Interfaces:**
- Consumes: `model.Resource`、`repository.ResourceListQuery`、`config.KnowledgePointsConfig`
- Produces:
  - `service.AggregateOptions{ Stoplist []string; MinCount int; Max int; Synonyms []SynonymGroup }`
  - `service.SynonymGroup{ Canonical string; Aliases []string }`
  - `service.AggregateKnowledgePointsFromTags(resources []model.Resource, opts AggregateOptions) []TagAggregate` —**签名变更**（原为四个位置参数）
  - `service.TagAggregate{ Name string; Count int; CategoryID uint; Aliases []string }` —**新增 `Aliases`**
  - `config.KnowledgePointsConfig.Synonyms []SynonymGroup`（mapstructure: `synonyms`）
  - `config.SynonymGroup{ Canonical string `mapstructure:"canonical"`; Aliases []string `mapstructure:"aliases"` }`
  - `repository.ResourceListQuery.AnyTags []string`（OR 语义，与既有的 `Tags` AND 语义并存）

- [ ] **Step 1: 写失败的测试 — 大小写归并与别名归并**

在 `backend/internal/service/knowledge_point_import_test.go` 中，**更新既有用例的调用形式**（签名变了），并追加：

```go
func TestAggregate_CaseFoldingAndSynonyms(t *testing.T) {
	t.Run("拉丁文大小写归并到同一条，展示名取出现最多的写法", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["Python"]`),
			resourceWithTags(2, 1, `["Python"]`),
			resourceWithTags(3, 1, `["python"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{MinCount: 2, Max: 60})

		if len(got) != 1 {
			t.Fatalf("got %d 条，want 1（Python/python 应归并）: %+v", len(got), got)
		}
		if got[0].Name != "Python" {
			t.Errorf("展示名 = %q, want Python（出现次数多的写法胜出）", got[0].Name)
		}
		if got[0].Count != 3 {
			t.Errorf("Count = %d, want 3（归并后应累加）", got[0].Count)
		}
		if len(got[0].Aliases) != 1 || got[0].Aliases[0] != "python" {
			t.Errorf("Aliases = %v, want [python]（折叠进来的原始写法要留作别名）", got[0].Aliases)
		}
	})

	t.Run("同频时展示名取字典序最小，保证确定", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["Go"]`),
			resourceWithTags(2, 1, `["go"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{MinCount: 2, Max: 60})
		// "Go" < "go"（大写字母字节序在前）
		if len(got) != 1 || got[0].Name != "Go" {
			t.Fatalf("got %+v, want 展示名 Go", got)
		}
	})

	t.Run("synonyms 把别名归到规范名", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["Golang"]`),
			resourceWithTags(2, 1, `["Go语言"]`),
			resourceWithTags(3, 1, `["Go"]`),
			resourceWithTags(4, 1, `["go"]`),
		}
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{
			MinCount: 2, Max: 60,
			Synonyms: []service.SynonymGroup{{Canonical: "Go", Aliases: []string{"Golang", "Go语言", "go"}}},
		})

		if len(got) != 1 {
			t.Fatalf("got %d 条，want 1（四种写法应归到 Go）: %+v", len(got), got)
		}
		if got[0].Name != "Go" || got[0].Count != 4 {
			t.Errorf("got %+v, want Go 计数 4", got[0])
		}
		// 别名要去重、且不含规范名本身；sort.Strings 后顺序确定。
		// 字节序：'G'(0x47) < 'g'(0x67)，且 "Golang" 的第 3 字节 'l'(0x6C) < "Go语言" 的第 3 字节 0xE8
		want := []string{"Golang", "Go语言", "go"}
		if len(got[0].Aliases) != len(want) {
			t.Fatalf("Aliases = %v, want %v", got[0].Aliases, want)
		}
		for i := range want {
			if got[0].Aliases[i] != want[i] {
				t.Errorf("Aliases = %v, want %v", got[0].Aliases, want)
				break
			}
		}
	})

	t.Run("synonyms 的别名也受停用词与长度规则约束", func(t *testing.T) {
		resources := []model.Resource{
			resourceWithTags(1, 1, `["Golang"]`),
			resourceWithTags(2, 1, `["Golang"]`),
		}
		// 别名 "Golang" 被停用 → 不应有任何知识点产出
		got := service.AggregateKnowledgePointsFromTags(resources, service.AggregateOptions{
			MinCount: 2, Max: 60, Stoplist: []string{"Golang"},
			Synonyms: []service.SynonymGroup{{Canonical: "Go", Aliases: []string{"Golang"}}},
		})
		if len(got) != 0 {
			t.Errorf("停用词应作用于别名，got %+v", got)
		}
	})
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/shionyori/project/edurec-platform/backend && go test ./internal/service/ -run TestAggregate -v`
Expected: 编译失败 — 参数类型不符 / `service.AggregateOptions` 未定义。

- [ ] **Step 3: 改聚合实现**

在 `backend/internal/service/knowledge_point_import.go` 中：

1. 新增选项结构与字段：

```go
// AggregateOptions 标签聚合的配置。用结构体而非散参：字段还会增加，
// 散参每次都要改所有调用点。
type AggregateOptions struct {
	// Stoplist 丢弃这些标签（平台徽章、人名、元信息词等无区分度的噪音）
	Stoplist []string
	// MinCount 标签至少要出现多少次才算知识点
	MinCount int
	// Max 知识点上限，<=0 表示不限
	Max int
	// Synonyms 把别名归到规范名（列表形式，见 config.SynonymGroup——用 map 会被 Viper 转小写，丢展示名）。
	// 拉丁文即便不在表里，也会按小写再归并一轮，避免同一个词因大小写拆成多条。
	Synonyms []SynonymGroup
}
```

2. `TagAggregate` 增加 `Aliases []string`：

```go
type TagAggregate struct {
	Name       string
	Count      int
	CategoryID uint
	// Aliases 折叠进这个知识点的其它写法（大小写变体 + synonyms 里配的别名），
	// 去重、不含 Name 本身、已排序。落库进 knowledge_points.aliases，
	// 并参与资源检索（见 ResourceListQuery.AnyTags）。
	Aliases []string
}
```

3. 把 `AggregateKnowledgePointsFromTags` 改为按「规范键」聚合：

```go
// AggregateKnowledgePointsFromTags 从资源标签聚合知识点候选。
//
// 归并规则（两级）：
//  1. synonyms 里配了别名 → 归到规范名（"Golang" → "Go"）
//  2. 其余按小写归并（"python" → "Python"），避免同一个词因大小写拆成多条
//
// 展示名取该组内出现次数最多的原始写法；同频取字典序最小的，保证多次运行结果一致。
func AggregateKnowledgePointsFromTags(resources []model.Resource, opts AggregateOptions) []TagAggregate {
	stop := make(map[string]struct{}, len(opts.Stoplist))
	for _, s := range opts.Stoplist {
		stop[s] = struct{}{}
	}

	// 别名（已小写）→ 规范名。规范名保留配置里的大小写（见 config.SynonymGroup）。
	aliasToCanonical := make(map[string]string)
	for _, syn := range opts.Synonyms {
		for _, alias := range syn.Aliases {
			aliasToCanonical[strings.ToLower(strings.TrimSpace(alias))] = syn.Canonical
		}
	}
	type group struct {
		canonical   string // 配了 synonyms 时是规范名；否则为空，由展示名兜底
		spellings   map[string]int
		categoryHit map[uint]int
	}
	groups := make(map[string]*group) // key = 归并键（小写后的规范名）

	for i := range resources {
		seen := make(map[string]struct{})
		for _, tag := range ParseResourceTags(resources[i].Tags) {
			if utf8.RuneCountInString(tag) < MinKnowledgePointNameRunes {
				continue
			}
			if _, blocked := stop[tag]; blocked {
				continue
			}
			lowered := strings.ToLower(tag)
			key := lowered
			canonical := ""
			if c, ok := aliasToCanonical[lowered]; ok {
				canonical = c
				key = strings.ToLower(c)
			}
			// 同一资源里重复挂同一个标签只算一次，否则词频会被单个资源刷高
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}

			g, ok := groups[key]
			if !ok {
				g = &group{canonical: canonical, spellings: make(map[string]int), categoryHit: make(map[uint]int)}
				groups[key] = g
			}
			g.spellings[tag]++
			if resources[i].CategoryID != 0 {
				g.categoryHit[resources[i].CategoryID]++
			}
		}
	}

	out := make([]TagAggregate, 0, len(groups))
	for _, g := range groups {
		name := g.canonical
		total := 0
		if name == "" {
			name, total = dominantSpelling(g.spellings)
		} else {
			for _, c := range g.spellings {
				total += c
			}
		}
		if total < opts.MinCount {
			continue
		}
		out = append(out, TagAggregate{
			Name:       name,
			Count:      total,
			CategoryID: dominantCategory(g.categoryHit),
			Aliases:    aliasesOf(name, g.spellings),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	if opts.Max > 0 && len(out) > opts.Max {
		out = out[:opts.Max]
	}
	return out
}

// dominantSpelling 取出现次数最多的原始写法；同频取字典序最小的，保证结果确定。
func dominantSpelling(spellings map[string]int) (string, int) {
	best, bestCount, total := "", -1, 0
	for spelling, count := range spellings {
		total += count
		if count > bestCount || (count == bestCount && spelling < best) {
			best, bestCount = spelling, count
		}
	}
	return best, total
}

// aliasesOf 列出折叠进该知识点的其它写法：不含规范名本身，去重后排序。
func aliasesOf(canonical string, spellings map[string]int) []string {
	out := make([]string, 0, len(spellings))
	for spelling := range spellings {
		if spelling != canonical {
			out = append(out, spelling)
		}
	}
	sort.Strings(out)
	return out
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/shionyori/project/edurec-platform/backend && go test ./internal/service/ -run TestAggregate -v`
Expected: 全部 PASS。

- [ ] **Step 5: 让导入把别名落库**

在 `ImportFromTags` 里，把构造 `model.KnowledgePoint` 的那段改为带上别名（`Aliases` 现在是动态值，不再是固定 `"[]"`）：

```go
	points := make([]model.KnowledgePoint, 0, len(candidates))
	for i, c := range candidates {
		// Aliases 是 MySQL JSON 列，必须永远是合法 JSON：没有别名也要写 "[]"。
		// Go 零值 "" 会被 MySQL 以 Error 3140 拒绝并回滚整批。
		aliasesJSON, err := json.Marshal(c.Aliases)
		if err != nil {
			aliasesJSON = []byte("[]")
		}
		points = append(points, model.KnowledgePoint{
			Name:       c.Name,
			CategoryID: c.CategoryID,
			Aliases:    string(aliasesJSON),
			Source:     model.KnowledgePointSourceTag,
			SortOrder:  i,
		})
	}
```

并把 `ImportFromTags` 里调用聚合的那行改为传 `AggregateOptions`：

```go
	candidates := AggregateKnowledgePointsFromTags(resources, AggregateOptions{
		Stoplist: effective.TagStoplist,
		MinCount: effective.MinResourceCount,
		Max:      effective.MaxKnowledgePoints,
		Synonyms: effective.Synonyms,
	})
```

- [ ] **Step 6: 加 synonyms 配置项**

在 `backend/internal/config/config.go` 的 `KnowledgePointsConfig` 里追加：

```go
	// Synonyms 把别名归到规范名。用「列表 + 显式 canonical 字段」而不是 map：
	// **Viper 会把 map 的键统一转成小写**，若用 map，规范名 "Go" 到 Go 侧会变成 "go"，
	// 展示名就丢掉大小写了（实测踩到）。列表形式没有这个问题。
	Synonyms []SynonymGroup `mapstructure:"synonyms"`
}

// SynonymGroup 一组同义词：Aliases 里的写法都归到 Canonical。
type SynonymGroup struct {
	Canonical string   `mapstructure:"canonical"`
	Aliases   []string `mapstructure:"aliases"`
}
```

在 `backend/internal/config/config_test.go` 追加：

```go
func TestKnowledgePointsSynonyms(t *testing.T) {
	t.Run("默认无同义词", func(t *testing.T) {
		// 复合字面量出现在 if 的初始化语句里必须加括号，否则 Go 语法歧义
		if got := (config.KnowledgePointsConfig{}).KnowledgePointsOrDefault(); len(got.Synonyms) != 0 {
			t.Errorf("Synonyms = %v, want 空", got.Synonyms)
		}
	})

	t.Run("显式配置原样保留", func(t *testing.T) {
		syn := []config.SynonymGroup{{Canonical: "Go", Aliases: []string{"Golang"}}}
		got := config.KnowledgePointsConfig{Synonyms: syn}.KnowledgePointsOrDefault()
		if len(got.Synonyms) != 1 || got.Synonyms[0].Canonical != "Go" || got.Synonyms[0].Aliases[0] != "Golang" {
			t.Errorf("Synonyms = %v, want 原样保留", got.Synonyms)
		}
	})
}
```

在 `backend/configs/config.yaml` 的 `knowledge_points:` 段补齐停用词与同义词：

```yaml
knowledge_points:
  # 无区分度的高频标签，聚合成知识点只会制造噪音。
  # 三类：平台徽章（B 站自动挂）、创作者/机构名、元信息词。
  tag_stoplist:
    - 教程
    - 示例
    - 合集
    - 全集
    - 讲解
    - 课程
    - 入门
    - 系列
    - 视频
    - 学习
    # 平台徽章
    - 科技猎手
    - 知识分享官
    - 野生技术协会
    - 活动作品
    # 创作者 / 机构
    - 吴恩达
    - 李宏毅
    - 尚硅谷
    - 黑马程序员
    # 元信息词
    - 视频教程
    - 教学视频
    - 快速入门
    - 自学
    - 程序员
    - 代码
    - IT
  # 标签至少要出现在多少个资源上才算知识点
  min_resource_count: 2
  # 聚合出的知识点上限
  max_knowledge_points: 60
  # 同义词归并：canonical 是规范名（会成为知识点展示名），aliases 是别名。
  # 别名会写进知识点的 aliases 并参与资源检索（OR 语义），所以带别名标签的资源不会丢。
  # 用列表而不是 map：Viper 会把 map 的键统一转小写，规范名 "Go" 会变成 "go"。
  synonyms:
    - canonical: Go
      aliases: [Go语言, GO语言, go语言, Golang, GOLANG, go, GO]
    - canonical: 人工智能
      aliases: [AI, ai, 人工智能入门]
    - canonical: 机器学习
      aliases: [机器学习入门, 机器学习算法]
    - canonical: 编程
      aliases: [编程语言]
    - canonical: 计算机
      aliases: [计算机技术]
```

- [ ] **Step 7: 给资源检索加 OR 语义（别名能查到资源）**

在 `backend/internal/repository/resource.go` 的 `ResourceListQuery` 里追加字段：

```go
	// AnyTags 任一命中即算（OR 语义），与 Tags 的全命中（AND）不同。
	// 知识点用它：知识点名与它的别名是「同一个东西的不同写法」，
	// 若按 AND 处理，只带别名标签的资源就查不出来了。
	AnyTags []string
```

在 `List` 的 tag 过滤之后（`for _, tag := range query.Tags { ... }` 那段之后）追加：

```go
	if len(query.AnyTags) > 0 {
		conds := make([]string, 0, len(query.AnyTags))
		args := make([]any, 0, len(query.AnyTags))
		for _, tag := range query.AnyTags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			tagJSON, _ := json.Marshal(tag)
			conds = append(conds, "JSON_CONTAINS(tags, ?)")
			args = append(args, string(tagJSON))
		}
		if len(conds) > 0 {
			db = db.Where("("+strings.Join(conds, " OR ")+")", args...)
		}
	}
```

- [ ] **Step 8: 全量验证**

Run:
```bash
cd /home/shionyori/project/edurec-platform/backend
gofmt -l internal/service/knowledge_point_import.go internal/config/config.go internal/repository/resource.go
go build ./... && go vet ./... && go test ./...
```
Expected: gofmt 无输出；编译/vet/测试全部通过。

- [ ] **Step 9: 重跑闸门并人工复核**

```bash
cd /home/shionyori/project/edurec-platform/backend
go run ./cmd/import_knowledge_points --from-tags --preview
```

**闸门复查要点**（把输出交给用户判断，不要自行认定通过）：
- `GO`/`Go`/`go`/`Golang`/`GOLANG`/`Go语言`/`go语言`/`GO语言` 是否已归并为**一条**（预期计数 17、排名第 6 左右），且**展示名保留大小写**（是 `Go` 而不是 `go`——这正是 Viper 键小写陷阱要验的点）
- 科技猎手 / 吴恩达 / 视频教程 之类是否已消失
- 候选总数是否明显少于 60（归并与停用都会减条目）

**仍不执行真正的导入**（`--from-tags` 不带 `--preview`/`--dry-run`）——写库时机由用户决定。

- [ ] **Step 10: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add backend/internal/service/knowledge_point_import.go \
        backend/internal/service/knowledge_point_import_test.go \
        backend/internal/config/config.go \
        backend/internal/config/config_test.go \
        backend/internal/repository/resource.go \
        backend/configs/config.yaml
git commit -m "feat(backend): 知识点聚合支持大小写归并与别名合并"
```

**提交后停下，等待用户确认。**

---

### Task 5: 目标匹配与分桶（核心）

**Files:**
- Create: `backend/internal/service/knowledge_point.go`
- Test: `backend/internal/service/knowledge_point_test.go`

**Interfaces:**
- Consumes: `repository.KnowledgePointRepository.ListAll`、`repository.ResourceRepository.List`、`RecommendationService.Get(userID uint, limit int) (*RecommendationResult, error)`、`apperror.BadRequest/Internal`、**`repository.ResourceListQuery.AnyTags`（Task 4b 新增的 OR 语义 tag 过滤）**
- 注意：本 Task 依赖 Task 4b 已落地。若 `AnyTags` 不存在，编译会报 `unknown field AnyTags`——那说明 Task 4b 没做完，先回去补。
- Produces:
  - `service.KnowledgePointGroup{ KnowledgePoint model.KnowledgePoint; Total int64; Resources []model.Resource }`
  - `service.LearningPlan{ Goal string; Matched bool; Groups []KnowledgePointGroup }`
  - `service.KnowledgePointService` 与 `service.NewKnowledgePointService(points repository.KnowledgePointRepository, resources repository.ResourceRepository, recommendations *RecommendationService) *KnowledgePointService`
  - `(*KnowledgePointService) PlanForGoal(userID uint, goal string, kpLimit int) (*LearningPlan, error)`
  - 常量 `service.DefaultKnowledgePointLimit = 8`、`service.MaxKnowledgePointLimit = 20`、`service.DefaultResourcePerPoint = 6`

- [ ] **Step 1: 写失败的测试**

创建 `backend/internal/service/knowledge_point_test.go`。fake 写法沿用 `recommendation_test.go` 的手写风格：

```go
package service_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

type fakeKnowledgePointRepository struct {
	points []model.KnowledgePoint
	err    error
}

func (f *fakeKnowledgePointRepository) ListAll() ([]model.KnowledgePoint, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.points, nil
}

func (f *fakeKnowledgePointRepository) UpsertByName(points []model.KnowledgePoint) (int, int, error) {
	return len(points), 0, nil
}

// fakePlanResourceRepository 按「查询里带的标签」返回资源，并记录收到的查询
type fakePlanResourceRepository struct {
	byTag  map[string][]model.Resource
	totals map[string]int64
	queries []repository.ResourceListQuery
}

func (f *fakePlanResourceRepository) List(q repository.ResourceListQuery) (*repository.ResourceListResult, error) {
	f.queries = append(f.queries, q)
	// 读 AnyTags 而不是 Tags：实现用 AnyTags（OR）查知识点资源。若这里读 Tags，
	// 所有知识点分组都会因取不到资源而被剔除，测试会成批假失败。
	if len(q.AnyTags) == 0 {
		items := f.byTag[""]
		return &repository.ResourceListResult{Items: items, Total: int64(len(items))}, nil
	}
	tag := q.AnyTags[0]
	items := f.byTag[tag]
	total := f.totals[tag]
	if total == 0 {
		total = int64(len(items))
	}
	return &repository.ResourceListResult{Items: items, Total: total}, nil
}

func (f *fakePlanResourceRepository) Create(*model.Resource) error { return nil }
func (f *fakePlanResourceRepository) FindByID(uint) (*model.Resource, error) {
	return nil, repository.ErrNotFound
}
// FindByIDs 必须真的按 id 返回资源：RecommendationService.Get 靠它把缓存的
// 推荐 ID 列表还原成资源顺序，返回 nil 会让「engine 推荐序优先」这条测试假通过。
func (f *fakePlanResourceRepository) FindByIDs(ids []uint) ([]model.Resource, error) {
	byID := make(map[uint]model.Resource)
	for _, items := range f.byTag {
		for _, r := range items {
			byID[r.ID] = r
		}
	}
	out := make([]model.Resource, 0, len(ids))
	for _, id := range ids {
		if r, ok := byID[id]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakePlanResourceRepository) ListTopByCategories([]uint, int) ([]model.Resource, error) {
	return nil, nil
}
func (f *fakePlanResourceRepository) FindBySourceURLs([]string) ([]model.Resource, error) {
	return nil, nil
}
func (f *fakePlanResourceRepository) Update(*model.Resource) error { return nil }
func (f *fakePlanResourceRepository) Delete(uint) error            { return nil }

func point(id uint, name, aliases string, categoryID uint) model.KnowledgePoint {
	p := model.KnowledgePoint{Name: name, Aliases: aliases, CategoryID: categoryID}
	p.ID = id
	if categoryID != 0 {
		p.Category = model.Category{Name: "人工智能"}
		p.Category.ID = categoryID
	}
	return p
}

func resource(id uint, rating float32, views uint) model.Resource {
	r := model.Resource{AvgRating: rating, ViewCount: views}
	r.ID = id
	return r
}

func newPlanService(
	points []model.KnowledgePoint,
	resources *fakePlanResourceRepository,
	recRepo repository.RecommendationRepository,
) *service.KnowledgePointService {
	return service.NewKnowledgePointService(
		&fakeKnowledgePointRepository{points: points},
		resources,
		service.NewRecommendationService(recRepo, resources, &fakeUserRepository{}),
	)
}

func TestPlanForGoal_MatchByName(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"机器学习": {resource(1, 4.5, 100)},
		"线性代数": {resource(2, 4.0, 50)},
	}}
	svc := newPlanService([]model.KnowledgePoint{
		point(1, "机器学习", `[]`, 1),
		point(2, "线性代数", `[]`, 1),
	}, resources, &fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我想入门机器学习", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if !plan.Matched {
		t.Fatal("Matched = false, want true")
	}
	if len(plan.Groups) != 1 || plan.Groups[0].KnowledgePoint.Name != "机器学习" {
		t.Fatalf("groups = %+v, want 仅 机器学习", plan.Groups)
	}
}

// 首轮知识点由中文标签聚合，aliases 全空，因此英文目标必然走兜底。
// 这条测试把「兜底而不是报错/空页」钉死。
func TestPlanForGoal_FallsBackWhenNoAliasMatches(t *testing.T) {
	resources := &fakePlanResourceRepository{
		byTag:  map[string][]model.Resource{"": {resource(9, 4.0, 10)}},
		totals: map[string]int64{},
	}
	svc := newPlanService([]model.KnowledgePoint{point(1, "机器学习", `[]`, 1)}, resources,
		&fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "machine learning", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if plan.Matched {
		t.Error("Matched = true, want false（英文目标应走兜底）")
	}
	if len(plan.Groups) != 1 {
		t.Fatalf("兜底应返回单一分组，got %d", len(plan.Groups))
	}
	if len(plan.Groups[0].Resources) == 0 {
		t.Error("兜底分组不该是空的——绝不能返回空白页")
	}
}

func TestPlanForGoal_MatchByAliasIgnoreCase(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"机器学习": {resource(1, 4.5, 100)},
	}}
	svc := newPlanService([]model.KnowledgePoint{
		point(1, "机器学习", `["Machine Learning"]`, 1),
	}, resources, &fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我想学 machine learning", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	// 必须断言 Matched 与分组名：兜底分支恒好返回 1 个分组，只断言 len(Groups)!=1
	// 的话，把别名解析整个删掉这条测试照样通过（会静默走兜底）。
	if !plan.Matched {
		t.Fatal("应命中知识点而非走兜底：Matched = false")
	}
	if len(plan.Groups) != 1 || plan.Groups[0].KnowledgePoint.Name != "机器学习" {
		t.Fatalf("拉丁文别名应忽略大小写命中「机器学习」，got %+v", plan.Groups)
	}
}

func TestPlanForGoal_MatchByCategoryWhenNameMisses(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"机器学习": {resource(1, 4.5, 100)},
	}}
	// 知识点名与目标文本没有公共子串，但分类名「人工智能」在目标里
	svc := newPlanService([]model.KnowledgePoint{
		point(1, "机器学习", `[]`, 1),
	}, resources, &fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我想入门人工智能", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	// 同上：只断言分组数会被兜底分支蒙混过关，必须断言命中而非兜底。
	if !plan.Matched {
		t.Fatal("应按分类名命中而非走兜底：Matched = false")
	}
	if len(plan.Groups) != 1 || plan.Groups[0].KnowledgePoint.Name != "机器学习" {
		t.Fatalf("应按分类名命中「机器学习」，got %+v", plan.Groups)
	}
}

func TestPlanForGoal_ShortNameIsNotMatched(t *testing.T) {
	// 必须给 "C" 配一条真实资源：否则「不匹配」与「匹配到但无资源被剔除」两条路径
	// 都落到兜底、Matched 都是 false，断言无法区分。配上资源后，一旦 minMatchRunes
	// 守卫被删，"C" 就会命中并产出分组，Matched 变 true，测试才会失败。
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"":  {resource(9, 4, 1)},
		"C": {resource(8, 3, 5)},
	}}
	svc := newPlanService([]model.KnowledgePoint{point(1, "C", `[]`, 0)}, resources,
		&fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我想学 C 语言和 Python", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	// 单字知识点名会命中一切目标，必须挡掉；命中不了就兜底
	if plan.Matched {
		t.Error("单字符知识点名不该被匹配")
	}
}

// 别名必须真的进入资源检索条件：否则「把 Golang 归并到 Go」只是改了个名字，
// 只带别名标签的资源会查不出来。替身只读 AnyTags[0]，所以断言要看发出的查询本身。
func TestPlanForGoal_QueriesResourcesByAliasesToo(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"机器学习": {resource(1, 4.5, 100)},
	}}
	// 这里给一张命中的推荐缓存：缓存命中时 Get 只走 FindByIDs，不再自己发
	// resources.List 冷启动兜底查询。若用空缓存，替身记录的 queries[0] 会是 engine
	// 那条兜底查询（AnyTags 为空），要断言的规划查询就退到 queries[1]。
	recRepo := &fakeRecommendationRepository{found: &model.Recommendation{
		UserID: 7, ResourceIDs: `[1]`, UpdatedAt: 1,
	}}
	svc := newPlanService([]model.KnowledgePoint{
		point(1, "机器学习", `["机器学习入门","机器学习算法"]`, 1),
	}, resources, recRepo)

	if _, err := svc.PlanForGoal(7, "机器学习", 8); err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if len(resources.queries) == 0 {
		t.Fatal("没有发出资源查询")
	}
	want := []string{"机器学习", "机器学习入门", "机器学习算法"}
	got := resources.queries[0].AnyTags
	if len(got) != len(want) {
		t.Fatalf("AnyTags = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AnyTags = %v, want %v", got, want)
			break
		}
	}
}

func TestPlanForGoal_SkipsKnowledgePointWithoutResources(t *testing.T) {
	resources := &fakePlanResourceRepository{
		byTag:  map[string][]model.Resource{"机器学习": {}, "线性代数": {resource(2, 4.0, 50)}},
		totals: map[string]int64{"机器学习": 0, "线性代数": 1},
	}
	svc := newPlanService([]model.KnowledgePoint{
		point(1, "机器学习", `[]`, 1),
		point(2, "线性代数", `[]`, 1),
	}, resources, &fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我想入门机器学习和线性代数", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if len(plan.Groups) != 1 || plan.Groups[0].KnowledgePoint.Name != "线性代数" {
		t.Fatalf("无资源的知识点应被剔除，got %+v", plan.Groups)
	}
}

func TestPlanForGoal_RanksByEngineRecommendationOrder(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"机器学习": {
			resource(1, 5.0, 1000), // 评分最高
			resource(2, 3.0, 10),   // 评分最低，但在 engine 推荐里排第一
		},
	}}
	// 字段名是 found（见 recommendation_test.go 里 fakeRecommendationRepository 的定义）
	recRepo := &fakeRecommendationRepository{found: &model.Recommendation{
		UserID: 7, ResourceIDs: `[2,1]`, UpdatedAt: 1,
	}}
	svc := newPlanService([]model.KnowledgePoint{point(1, "机器学习", `[]`, 1)}, resources, recRepo)

	plan, err := svc.PlanForGoal(7, "机器学习", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	got := plan.Groups[0].Resources
	if len(got) != 2 || got[0].ID != 2 {
		t.Fatalf("engine 推荐序优先，want 资源 2 在前，got %+v", got)
	}
}

func TestPlanForGoal_CapsResourcesPerPoint(t *testing.T) {
	items := make([]model.Resource, 0, 10)
	for i := 1; i <= 10; i++ {
		items = append(items, resource(uint(i), float32(i), uint(i)))
	}
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{"机器学习": items}}
	svc := newPlanService([]model.KnowledgePoint{point(1, "机器学习", `[]`, 1)}, resources,
		&fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "机器学习", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if len(plan.Groups[0].Resources) != service.DefaultResourcePerPoint {
		t.Errorf("组内资源 = %d, want %d", len(plan.Groups[0].Resources), service.DefaultResourcePerPoint)
	}
	if plan.Groups[0].Total != 10 {
		t.Errorf("Total = %d, want 10（总数不受展示上限影响）", plan.Groups[0].Total)
	}
}

func TestPlanForGoal_RejectsBadGoal(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{}}
	svc := newPlanService(nil, resources, &fakeRecommendationRepository{})

	// 断言具体错误码（复用 auth_test.go 里的 assertErrorCode）：
	// 只断言 err != nil 的话，任何错误都能通过，测不出「该是 400 而不是 500」。
	t.Run("空目标", func(t *testing.T) {
		_, err := svc.PlanForGoal(7, "   ", 8)
		assertErrorCode(t, err, apperror.CodeBadRequest)
	})

	t.Run("超长目标", func(t *testing.T) {
		_, err := svc.PlanForGoal(7, strings.Repeat("学", 101), 8)
		assertErrorCode(t, err, apperror.CodeBadRequest)
	})
}

func TestPlanForGoal_RepositoryErrorIsInternal(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{}}
	svc := service.NewKnowledgePointService(
		&fakeKnowledgePointRepository{err: errors.New("db down")},
		resources,
		service.NewRecommendationService(&fakeRecommendationRepository{}, resources, &fakeUserRepository{}),
	)
	_, err := svc.PlanForGoal(7, "机器学习", 8)
	// 名称说的是 IsInternal：只断言 err != nil 的话，BadRequest 也能通过。
	assertErrorCode(t, err, apperror.CodeInternal)
}

// 已接受的已知行为：子串匹配不是语义匹配。把行为钉死，避免将来被当成 bug 反复「修」。
func TestPlanForGoal_SubstringMatchIgnoresNegation(t *testing.T) {
	resources := &fakePlanResourceRepository{byTag: map[string][]model.Resource{
		"机器学习": {resource(1, 4.5, 100)},
	}}
	svc := newPlanService([]model.KnowledgePoint{point(1, "机器学习", `[]`, 1)}, resources,
		&fakeRecommendationRepository{})

	plan, err := svc.PlanForGoal(7, "我不想学机器学习了", 8)
	if err != nil {
		t.Fatalf("PlanForGoal 报错: %v", err)
	}
	if !plan.Matched {
		t.Error("已知行为：子串匹配不识别否定，仍应命中「机器学习」")
	}
}
```

> 注意：`fakeRecommendationRepository` 与 `fakeUserRepository` 已存在于 `recommendation_test.go`（同一个 `service_test` 包），**不要重复定义**。
> 已核实的字段名：`fakeRecommendationRepository.found *model.Recommendation`、`.findErr error`；`fakeUserRepository.users []*model.User`、`.nextID uint`。
> 另注意 `fakeResourceRepository` 也已存在，所以本 Task 的资源替身取名 `fakePlanResourceRepository` 以避免重名。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/shionyori/project/edurec-platform/backend && go test ./internal/service/ -run TestPlanForGoal -v`
Expected: 编译失败 — `undefined: service.NewKnowledgePointService`。

- [ ] **Step 3: 写实现**

创建 `backend/internal/service/knowledge_point.go`：

```go
package service

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

const (
	// DefaultKnowledgePointLimit 一次规划返回的知识点数上限
	DefaultKnowledgePointLimit = 8
	// MaxKnowledgePointLimit 调用方可要求的上限，防止把整页塞满
	MaxKnowledgePointLimit = 20
	// DefaultResourcePerPoint 每个知识点组内展示的资源数上限
	DefaultResourcePerPoint = 6
	// resourceCandidateLimit 组内候选池大小（ResourceListQuery.PageSize 的上限）
	resourceCandidateLimit = 100
	// recSignalLimit 取多少条 engine 推荐结果当排序信号
	recSignalLimit = 50
	// defaultFallbackResourceLimit 兜底分组的资源条数
	defaultFallbackResourceLimit = 12
	// minGoalRunes / maxGoalRunes 学习目标的字符数区间
	minGoalRunes = 2
	maxGoalRunes = 100
	// minMatchRunes 命中词的最小长度：短词（尤其单字符）会命中几乎任何目标文本，
	// 让匹配失去意义。中文与拉丁文都按 rune 计。
	minMatchRunes = 2
	// fallbackGroupName 兜底分组的展示名
	fallbackGroupName = "为你找到的相关资源"
)

// KnowledgePointGroup 一个知识点及其推荐资源
type KnowledgePointGroup struct {
	KnowledgePoint model.KnowledgePoint
	Total          int64 // 该知识点关联的资源总数（不受组内上限影响）
	Resources      []model.Resource
}

// LearningPlan 一个学习目标下的知识点规划结果
type LearningPlan struct {
	Goal    string
	Matched bool // false 表示未匹配到知识点，Groups 为兜底单一分组
	Groups  []KnowledgePointGroup
}

// KnowledgePointService 学习目标 → 知识点 → 分组资源。
//
// 它不引入任何新模型：知识点是「分桶维度」，组内排序复用平台既有信号
// （engine 推荐序 > 平均分 > 浏览量）。平台依然不做推理。
type KnowledgePointService struct {
	points          repository.KnowledgePointRepository
	resources       repository.ResourceRepository
	recommendations *RecommendationService
}

func NewKnowledgePointService(
	points repository.KnowledgePointRepository,
	resources repository.ResourceRepository,
	recommendations *RecommendationService,
) *KnowledgePointService {
	return &KnowledgePointService{points: points, resources: resources, recommendations: recommendations}
}

// PlanForGoal 把学习目标规划成若干知识点，每个知识点挂一组资源。
//
// 校验放在这里而不是 handler：本仓 internal/handler/ 没有任何测试，
// 放在 service 才能被单测覆盖（见 Global Constraints）。
func (s *KnowledgePointService) PlanForGoal(userID uint, goal string, kpLimit int) (*LearningPlan, error) {
	goal = strings.TrimSpace(goal)
	if utf8.RuneCountInString(goal) < minGoalRunes {
		return nil, apperror.BadRequest("请输入学习目标")
	}
	if utf8.RuneCountInString(goal) > maxGoalRunes {
		return nil, apperror.BadRequest("学习目标过长，请精简到 100 字以内")
	}
	if kpLimit <= 0 {
		kpLimit = DefaultKnowledgePointLimit
	}
	if kpLimit > MaxKnowledgePointLimit {
		kpLimit = MaxKnowledgePointLimit
	}

	points, err := s.points.ListAll()
	if err != nil {
		return nil, apperror.Internal(err)
	}

	matched := matchKnowledgePoints(goal, points, kpLimit)
	if len(matched) == 0 {
		return s.fallback(userID, goal)
	}

	ranks := s.recommendationRanks(userID)

	groups := make([]KnowledgePointGroup, 0, len(matched))
	for i := range matched {
		// 用 AnyTags（OR）而不是 Tags（AND）：知识点名与它的别名是同一个知识点的
		// 不同写法，只按规范名查会漏掉只带别名标签的资源（如只挂 "Golang" 的视频）。
		list, err := s.resources.List(repository.ResourceListQuery{
			Page:     1,
			PageSize: resourceCandidateLimit,
			AnyTags:  resourceTagsOf(&matched[i]),
			Sort:     "rating",
		})
		if err != nil {
			return nil, apperror.Internal(err)
		}
		// 决策 #6：没有资源的知识点整体剔除。点开却没内容比不展示更伤。
		if list.Total == 0 {
			continue
		}

		items := list.Items
		sortByPreference(items, ranks)
		if len(items) > DefaultResourcePerPoint {
			items = items[:DefaultResourcePerPoint]
		}
		groups = append(groups, KnowledgePointGroup{
			KnowledgePoint: matched[i],
			Total:          list.Total,
			Resources:      items,
		})
	}

	if len(groups) == 0 {
		return s.fallback(userID, goal)
	}
	return &LearningPlan{Goal: goal, Matched: true, Groups: groups}, nil
}

// knowledgePointMatch 命中的知识点与其命中词长度
type knowledgePointMatch struct {
	point      model.KnowledgePoint
	matchRunes int
}

// matchKnowledgePoints 三级匹配：知识点名/别名 → 分类名 → 无。
//
// 中文按子串匹配，不需要分词：「我想入门机器学习」直接包含「机器学习」。
// 这是确定性算法，可测、可复现，代价是不识别否定与语义（见 spec §14 风险表）。
func matchKnowledgePoints(goal string, points []model.KnowledgePoint, limit int) []model.KnowledgePoint {
	lowered := strings.ToLower(goal)

	matches := make([]knowledgePointMatch, 0)
	for i := range points {
		if n, ok := bestMatch(lowered, &points[i]); ok {
			matches = append(matches, knowledgePointMatch{point: points[i], matchRunes: n})
		}
	}
	if len(matches) == 0 {
		for i := range points {
			if points[i].Category.ID == 0 {
				continue
			}
			name := strings.ToLower(strings.TrimSpace(points[i].Category.Name))
			if utf8.RuneCountInString(name) < minMatchRunes {
				continue
			}
			if strings.Contains(lowered, name) {
				matches = append(matches, knowledgePointMatch{
					point: points[i], matchRunes: utf8.RuneCountInString(name),
				})
			}
		}
	}

	// 命中词越长越具体：「机器学习」应压过「学习」这种更泛的词
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].matchRunes != matches[j].matchRunes {
			return matches[i].matchRunes > matches[j].matchRunes
		}
		if matches[i].point.SortOrder != matches[j].point.SortOrder {
			return matches[i].point.SortOrder < matches[j].point.SortOrder
		}
		return matches[i].point.ID < matches[j].point.ID
	})

	if len(matches) > limit {
		matches = matches[:limit]
	}
	out := make([]model.KnowledgePoint, 0, len(matches))
	for i := range matches {
		out = append(out, matches[i].point)
	}
	return out
}

// bestMatch 返回该知识点在目标文本中的最长命中词长度；未命中返回 false。
//
// 长度小于 minMatchRunes 的词直接跳过：单字符知识点名（如 "C"）会命中几乎所有
// 目标文本，让它参与匹配等于让匹配失效。拉丁文按小写比较，中文无影响。
func bestMatch(loweredGoal string, point *model.KnowledgePoint) (int, bool) {
	best := 0
	candidates := append([]string{point.Name}, parseStringJSON(point.Aliases)...)
	for _, cand := range candidates {
		cand = strings.ToLower(strings.TrimSpace(cand))
		if utf8.RuneCountInString(cand) < minMatchRunes {
			continue
		}
		if strings.Contains(loweredGoal, cand) {
			if n := utf8.RuneCountInString(cand); n > best {
				best = n
			}
		}
	}
	return best, best > 0
}

// parseStringJSON 解析 JSON 字符串数组；为空或损坏返回空切片。
func parseStringJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

// resourceTagsOf 列出检索该知识点资源时用到的全部标签写法：规范名 + 别名。
// 别名由 Task 4b 的聚合产出（大小写变体 + synonyms 归并进来的写法）。
func resourceTagsOf(point *model.KnowledgePoint) []string {
	out := make([]string, 0, 1+len(point.Aliases))
	out = append(out, point.Name)
	return append(out, parseStringJSON(point.Aliases)...)
}

// sortByPreference 组内排序：命中 engine 推荐缓存的资源优先（按缓存顺序），
// 其余按平均分、再按浏览量。
func sortByPreference(items []model.Resource, ranks map[uint]int) {
	sort.SliceStable(items, func(i, j int) bool {
		ri, iRanked := ranks[items[i].ID]
		rj, jRanked := ranks[items[j].ID]
		switch {
		case iRanked && jRanked:
			if ri != rj {
				return ri < rj
			}
		case iRanked != jRanked:
			return iRanked
		}
		if items[i].AvgRating != items[j].AvgRating {
			return items[i].AvgRating > items[j].AvgRating
		}
		return items[i].ViewCount > items[j].ViewCount
	})
}

// recommendationRanks 取当前用户的 engine 推荐顺序，作为组内排序的第一优先级。
// 取不到（缓存未命中、仓储出错）时返回空表，排序自动退化为纯评分+热度，
// 不影响主流程——兜底不是错误路径。
func (s *KnowledgePointService) recommendationRanks(userID uint) map[uint]int {
	ranks := make(map[uint]int)
	if s.recommendations == nil {
		return ranks
	}
	result, err := s.recommendations.Get(userID, recSignalLimit)
	if err != nil {
		return ranks
	}
	for i := range result.List {
		ranks[result.List[i].ID] = i
	}
	return ranks
}

// fallback 未匹配到知识点（或命中的知识点全无资源）时的兜底：
// 按目标做关键词搜索，返回单一分组。绝不返回空白页。
func (s *KnowledgePointService) fallback(userID uint, goal string) (*LearningPlan, error) {
	list, err := s.resources.List(repository.ResourceListQuery{
		Page:     1,
		PageSize: defaultFallbackResourceLimit,
		Keyword:  goal,
		Sort:     "rating",
	})
	if err != nil {
		return nil, apperror.Internal(err)
	}

	items := list.Items
	if len(items) > 0 {
		sortByPreference(items, s.recommendationRanks(userID))
	}

	return &LearningPlan{
		Goal:    goal,
		Matched: false,
		Groups: []KnowledgePointGroup{{
			KnowledgePoint: model.KnowledgePoint{Name: fallbackGroupName},
			Total:          list.Total,
			Resources:      items,
		}},
	}, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/shionyori/project/edurec-platform/backend && go test ./internal/service/ -run TestPlanForGoal -v`
Expected: 全部 PASS。

- [ ] **Step 5: 跑全量后端测试**

Run: `cd /home/shionyori/project/edurec-platform/backend && go test ./... && go vet ./...`
Expected: 全部 PASS，无 vet 警告。

- [ ] **Step 6: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add backend/internal/service/knowledge_point.go backend/internal/service/knowledge_point_test.go
git commit -m "feat(backend): 学习目标匹配知识点并分桶返回资源"
```

**提交后停下，等待用户确认。**

---

### Task 6: HTTP 接口与路由

**Files:**
- Create: `backend/internal/handler/learning_plan.go`
- Modify: `backend/internal/router/router.go`

**Interfaces:**
- Consumes: `service.KnowledgePointService.PlanForGoal`、`handler.toResourceListItem`、`handler.categorySummary`、`response.OK`、`handleError`、`queryPositiveInt`、`middleware.UserID`
- Produces: `GET /api/v1/learning-plan?goal=<string>&knowledge_points=<int>`

- [ ] **Step 1: 写 handler**

创建 `backend/internal/handler/learning_plan.go`：

```go
package handler

import (
	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/middleware"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type LearningPlanHandler struct {
	plan *service.KnowledgePointService
}

func NewLearningPlanHandler(plan *service.KnowledgePointService) *LearningPlanHandler {
	return &LearningPlanHandler{plan: plan}
}

type knowledgePointSummary struct {
	ID          uint             `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Category    *categorySummary `json:"category"`
}

type learningPlanGroup struct {
	KnowledgePoint knowledgePointSummary `json:"knowledge_point"`
	Total          int64                 `json:"total"`
	Resources      []resourceListItem    `json:"resources"`
}

// Get 学习目标 → 知识点规划（GET /api/v1/learning-plan）
//
// 参数校验（空目标、超长目标）在 service 里，这里只做绑定与错误映射。
func (h *LearningPlanHandler) Get(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
		return
	}

	limit, ok := queryPositiveInt(c, "knowledge_points", service.DefaultKnowledgePointLimit)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	plan, err := h.plan.PlanForGoal(userID, c.Query("goal"), limit)
	if err != nil {
		handleError(c, err)
		return
	}

	groups := make([]learningPlanGroup, 0, len(plan.Groups))
	for i := range plan.Groups {
		g := &plan.Groups[i]

		items := make([]resourceListItem, 0, len(g.Resources))
		for j := range g.Resources {
			items = append(items, toResourceListItem(&g.Resources[j]))
		}

		var category *categorySummary
		if g.KnowledgePoint.Category.ID != 0 {
			category = &categorySummary{
				ID:   g.KnowledgePoint.Category.ID,
				Name: g.KnowledgePoint.Category.Name,
			}
		}

		groups = append(groups, learningPlanGroup{
			KnowledgePoint: knowledgePointSummary{
				ID:          g.KnowledgePoint.ID,
				Name:        g.KnowledgePoint.Name,
				Description: g.KnowledgePoint.Description,
				Category:    category,
			},
			Total:     g.Total,
			Resources: items,
		})
	}

	response.OK(c, gin.H{
		"goal":    plan.Goal,
		"matched": plan.Matched,
		"groups":  groups,
	})
}
```

- [ ] **Step 2: 接线到 router**

修改 `backend/internal/router/router.go`。在 `commentRepo := ...` 之后加：

```go
	knowledgePointRepo := repository.NewKnowledgePointRepository(db)
```

在 `recommendationImportService := ...` 之后加（必须在 `recommendationService` 之后，它依赖后者）：

```go
	knowledgePointService := service.NewKnowledgePointService(
		knowledgePointRepo, resourceRepo, recommendationService,
	)
```

在 `recommendationHandler := ...` 之后加：

```go
	learningPlanHandler := handler.NewLearningPlanHandler(knowledgePointService)
```

在 `protected.GET("/recommendations", recommendationHandler.Get)` 之后加：

```go
	protected.GET("/learning-plan", learningPlanHandler.Get)
```

- [ ] **Step 3: 编译与全量测试**

Run: `cd /home/shionyori/project/edurec-platform/backend && go build ./... && go vet ./... && go test ./...`
Expected: 全部成功。

- [ ] **Step 4: 手动验证接口**

启动后端（`CONFIG_PATH=configs/config.yaml go run ./cmd/server`），先登录拿 token：

```bash
TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<你的密码>"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["access_token"])')
```

再依次验证三条路径：

```bash
# 1) 命中知识点
curl -s "localhost:8080/api/v1/learning-plan?goal=我想入门机器学习" -H "Authorization: Bearer $TOKEN" | head -c 800
# 2) 未命中 → 兜底（matched 应为 false，groups 仍有 1 组）
curl -s "localhost:8080/api/v1/learning-plan?goal=zzzz不存在的目标zzzz" -H "Authorization: Bearer $TOKEN" | head -c 400
# 3) 空目标 → 400
curl -s -o /dev/null -w '%{http_code}\n' "localhost:8080/api/v1/learning-plan?goal=" -H "Authorization: Bearer $TOKEN"
```

Expected: (1) 返回若干 groups，每组含 `total` 与 `resources`；(2) HTTP 200 且 `"matched":false`；(3) 输出 `400`。

- [ ] **Step 5: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add backend/internal/handler/learning_plan.go backend/internal/router/router.go
git commit -m "feat(backend): 新增学习计划接口与路由"
```

**提交后停下，等待用户确认。**

---

### Task 7: 前端类型、API 模块与 MSW

**Files:**
- Modify: `frontend/src/types/index.ts`
- Create: `frontend/src/api/learningPlan.ts`
- Modify: `frontend/src/mocks/handlers.ts`

**Interfaces:**
- Consumes: `get` from `./client`；MSW 的 `BASE` / `ok` / `bizError` / `requireUser` / `toPublicResource` / `db`
- Produces:
  - `types.KnowledgePointSummary`、`types.LearningPlanGroup`、`types.LearningPlan`
  - `api.getLearningPlan(goal: string, knowledgePoints?: number): Promise<LearningPlan>`

- [ ] **Step 1: 加类型**

追加到 `frontend/src/types/index.ts`：

```ts
// 学习目标下的一块知识点（本轮由资源标签聚合而来）
export interface KnowledgePointSummary {
  id: number
  name: string
  description: string
  category: Pick<Category, 'id' | 'name'> | null
}

// 一个知识点及其推荐资源；total 是该知识点关联的资源总数（组内最多展示 6 个）
export interface LearningPlanGroup {
  knowledge_point: KnowledgePointSummary
  total: number
  resources: Resource[]
}

// 学习目标 → 知识点规划结果；matched=false 表示未命中知识点，groups 为兜底单一分组
export interface LearningPlan {
  goal: string
  matched: boolean
  groups: LearningPlanGroup[]
}
```

- [ ] **Step 2: 加 API 模块**

创建 `frontend/src/api/learningPlan.ts`：

```ts
import { get } from './client'
import type { LearningPlan } from '@/types'

// 学习目标 → 知识点规划。goal 的空值与长度校验在后端进行（400），
// 前端只做即时提示，不作为唯一防线。
export function getLearningPlan(goal: string, knowledgePoints = 8) {
  return get<LearningPlan>('/learning-plan', { params: { goal, knowledge_points: knowledgePoints } })
}
```

- [ ] **Step 3: 加 MSW handler**

在 `frontend/src/mocks/handlers.ts` 的 `handlers` 数组里，紧跟 `GET /recommendations` 那个 handler 之后插入：

```ts
  // 学习目标 → 知识点（mock：按目标子串匹配标签，无命中即兜底）
  http.get(`${BASE}/learning-plan`, ({ request }) => {
    const user = requireUser(request)
    if (!user) return bizError(10002, '未认证', 401)
    const url = new URL(request.url)
    const goal = (url.searchParams.get('goal') ?? '').trim()
    if (goal.length < 2) return bizError(10001, '请输入学习目标', 400)
    const limit = Math.min(Number(url.searchParams.get('knowledge_points') ?? '8'), 20)

    const tagCount = new Map<string, number>()
    for (const r of db.resources) {
      for (const t of r.tags) tagCount.set(t, (tagCount.get(t) ?? 0) + 1)
    }
    const matchedTags = [...tagCount.keys()]
      .filter((t) => t.length >= 2 && goal.includes(t))
      .sort((a, b) => b.length - a.length)
      .slice(0, limit)

    // 未命中：兜底成单一分组，与后端一致（绝不返回空白）
    if (matchedTags.length === 0) {
      const list = db.resources.filter(
        (r) => r.title.includes(goal) || r.description.includes(goal),
      )
      return ok({
        goal,
        matched: false,
        groups: [
          {
            knowledge_point: { id: 0, name: '为你找到的相关资源', description: '', category: null },
            total: list.length,
            resources: list.map(toPublicResource),
          },
        ],
      })
    }

    return ok({
      goal,
      matched: true,
      groups: matchedTags.map((tag, i) => {
        const list = db.resources.filter((r) => r.tags.includes(tag))
        return {
          knowledge_point: { id: i + 1, name: tag, description: '', category: null },
          total: list.length,
          resources: list.slice(0, 6).map(toPublicResource),
        }
      }),
    })
  }),
```

- [ ] **Step 4: 类型检查与测试**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm type-check && pnpm test`
Expected: 全部通过（MSW 自身测试在 `mocks/__tests__/`，应仍然绿）。

- [ ] **Step 5: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add frontend/src/types/index.ts frontend/src/api/learningPlan.ts frontend/src/mocks/handlers.ts
git commit -m "feat(frontend): 学习计划类型、接口与 mock"
```

**提交后停下，等待用户确认。**

---

### Task 8: 提问框组件

**Files:**
- Create: `frontend/src/pages/home/GoalInput.vue`
- Test: `frontend/src/pages/home/__tests__/GoalInput.test.ts`

**Interfaces:**
- Consumes: `ElMessage`
- Produces: `GoalInput.vue`，props `{ initialGoal?: string; submitLabel?: string; loading?: boolean }`，emits `{ submit: [goal: string] }`

- [ ] **Step 1: 写失败的测试**

创建 `frontend/src/pages/home/__tests__/GoalInput.test.ts`：

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus, { ElMessage } from 'element-plus'
import GoalInput from '../GoalInput.vue'

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<typeof import('element-plus')>('element-plus')
  return { ...actual, ElMessage: { warning: vi.fn(), error: vi.fn(), success: vi.fn() } }
})

function mountInput(props: Record<string, unknown> = {}) {
  return mount(GoalInput, { props, global: { plugins: [ElementPlus] } })
}

describe('GoalInput', () => {
  beforeEach(() => {
    vi.mocked(ElMessage.warning).mockClear()
  })

  it('合法目标提交时 emit 去除首尾空白的目标', async () => {
    const wrapper = mountInput()
    await wrapper.find('input').setValue('  我想入门机器学习  ')
    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('submit')).toEqual([['我想入门机器学习']])
  })

  it('空目标不提交并给出提示', async () => {
    const wrapper = mountInput()
    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('submit')).toBeUndefined()
    expect(ElMessage.warning).toHaveBeenCalled()
  })

  it('超长目标不提交并给出提示', async () => {
    const wrapper = mountInput()
    await wrapper.find('input').setValue('学'.repeat(101))
    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('submit')).toBeUndefined()
    expect(ElMessage.warning).toHaveBeenCalled()
  })

  it('initialGoal 变化时回填输入框（URL 同步场景）', async () => {
    const wrapper = mountInput({ initialGoal: '机器学习' })
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('机器学习')

    await wrapper.setProps({ initialGoal: '线性代数' })
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('线性代数')
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm test GoalInput`
Expected: 失败 — 找不到 `../GoalInput.vue`。

- [ ] **Step 3: 写实现**

创建 `frontend/src/pages/home/GoalInput.vue`：

```vue
<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  initialGoal?: string
  submitLabel?: string
  loading?: boolean
}>()

const emit = defineEmits<{ submit: [goal: string] }>()

// 与后端 service 的 minGoalRunes / maxGoalRunes 对齐
const MIN_GOAL_LENGTH = 2
const MAX_GOAL_LENGTH = 100

const goal = ref(props.initialGoal ?? '')

// 目标同步到 URL 后，浏览器前进/后退或直接带 ?goal= 打开时要回填输入框
watch(
  () => props.initialGoal,
  (next) => {
    goal.value = next ?? ''
  },
)

function submit() {
  const trimmed = goal.value.trim()
  if (trimmed.length < MIN_GOAL_LENGTH) {
    ElMessage.warning('请输入学习目标')
    return
  }
  if (trimmed.length > MAX_GOAL_LENGTH) {
    ElMessage.warning(`学习目标请精简到 ${MAX_GOAL_LENGTH} 字以内`)
    return
  }
  emit('submit', trimmed)
}
</script>

<template>
  <form class="flex flex-col gap-3 sm:flex-row" @submit.prevent="submit">
    <el-input
      v-model="goal"
      size="large"
      maxlength="100"
      placeholder="例如：我想入门机器学习"
      aria-label="学习目标"
    />
    <el-button type="primary" size="large" native-type="submit" :loading="props.loading">
      {{ props.submitLabel ?? '开始规划' }}
    </el-button>
  </form>
</template>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm test GoalInput`
Expected: 全部 PASS。

- [ ] **Step 5: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add frontend/src/pages/home/GoalInput.vue frontend/src/pages/home/__tests__/GoalInput.test.ts
git commit -m "feat(frontend): 新增学习目标提问框组件"
```

**提交后停下，等待用户确认。**

---

### Task 9: 知识点折叠面板组件

**Files:**
- Create: `frontend/src/pages/home/KnowledgePointPanel.vue`
- Test: `frontend/src/pages/home/__tests__/KnowledgePointPanel.test.ts`

**Interfaces:**
- Consumes: `types.LearningPlanGroup`、`ResourceCard.vue`
- Produces: `KnowledgePointPanel.vue`，props `{ group: LearningPlanGroup }`，根节点为 `el-collapse-item`

- [ ] **Step 1: 写失败的测试**

创建 `frontend/src/pages/home/__tests__/KnowledgePointPanel.test.ts`：

```ts
import { describe, expect, it, vi } from 'vitest'
import { RouterLinkStub, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { LearningPlanGroup, Resource } from '@/types'
import KnowledgePointPanel from '../KnowledgePointPanel.vue'

vi.mock('@/api/behavior', () => ({ recordBehavior: vi.fn(() => Promise.resolve(null)) }))

const resource: Resource = {
  id: 1, title: '机器学习入门', description: '面向零基础学习者',
  cover_url: null, type: 'course',
  category: { id: 1, name: '人工智能' },
  tags: ['机器学习'], metadata: {},
  author: '吴恩达', source_url: null,
  avg_rating: 4.5, view_count: 1024,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-15T10:00:00Z',
}

function group(overrides: Partial<LearningPlanGroup> = {}): LearningPlanGroup {
  return {
    knowledge_point: { id: 1, name: '监督学习', description: '', category: null },
    total: 1,
    resources: [resource],
    ...overrides,
  }
}

// el-collapse-item 必须挂在 el-collapse 里才有正确的注入上下文。
// 注意 props 的写法：组件定义里的 props 是「声明」，不是「值」——
// 值要从 mount 的第二个参数传进去，否则 :group 拿到 undefined。
const PanelHarness = {
  components: { KnowledgePointPanel },
  props: ['g'],
  template: '<el-collapse><KnowledgePointPanel :group="g" /></el-collapse>',
}

function mountPanel(g: LearningPlanGroup) {
  return mount(PanelHarness, {
    props: { g },
    global: { plugins: [ElementPlus], stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('KnowledgePointPanel', () => {
  it('渲染知识点名与资源总数', () => {
    const wrapper = mountPanel(group({ total: 12 }))
    expect(wrapper.text()).toContain('监督学习')
    expect(wrapper.text()).toContain('12 个资源')
  })

  it('渲染组内资源卡片', () => {
    const wrapper = mountPanel(group())
    expect(wrapper.text()).toContain('机器学习入门')
  })

  it('total 大于已展示数量时给出「查看全部」入口', () => {
    const wrapper = mountPanel(group({ total: 12 }))
    const link = wrapper.findAllComponents(RouterLinkStub).find((l) => l.text().includes('查看全部'))
    expect(link).toBeTruthy()
    expect(link!.props('to')).toEqual({ name: 'search', query: { keyword: '监督学习' } })
  })

  it('total 未超过已展示数量时不出现「查看全部」', () => {
    const wrapper = mountPanel(group({ total: 1 }))
    expect(wrapper.text()).not.toContain('查看全部')
  })

  it('兜底分组的占位知识点也能正常渲染', () => {
    const wrapper = mountPanel(
      group({
        knowledge_point: { id: 0, name: '为你找到的相关资源', description: '', category: null },
      }),
    )
    expect(wrapper.text()).toContain('为你找到的相关资源')
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm test KnowledgePointPanel`
Expected: 失败 — 找不到 `../KnowledgePointPanel.vue`。

- [ ] **Step 3: 写实现**

创建 `frontend/src/pages/home/KnowledgePointPanel.vue`：

```vue
<script setup lang="ts">
import type { LearningPlanGroup } from '@/types'
import ResourceCard from '@/components/resource/ResourceCard.vue'

const props = defineProps<{ group: LearningPlanGroup }>()

// 兜底分组的 knowledge_point.id 为 0，用名字保证 name 唯一
const panelName = props.group.knowledge_point.id
  ? String(props.group.knowledge_point.id)
  : props.group.knowledge_point.name
</script>

<template>
  <el-collapse-item :name="panelName">
    <template #title>
      <div class="flex flex-1 items-center justify-between pr-3">
        <span class="font-medium text-ink">{{ group.knowledge_point.name }}</span>
        <span class="text-xs text-ink-muted">{{ group.total }} 个资源</span>
      </div>
    </template>

    <div v-if="group.resources.length === 0" class="py-4 text-sm text-ink-muted">
      这个知识点暂时没有可推荐的资源
    </div>
    <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <ResourceCard v-for="r in group.resources" :key="r.id" :resource="r" />
    </div>

    <RouterLink
      v-if="group.total > group.resources.length"
      :to="{ name: 'search', query: { keyword: group.knowledge_point.name } }"
      class="mt-4 inline-block text-sm text-primary hover:underline"
    >
      去资源库查看全部 {{ group.total }} 个
    </RouterLink>
  </el-collapse-item>
</template>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm test KnowledgePointPanel`
Expected: 全部 PASS。

- [ ] **Step 5: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add frontend/src/pages/home/KnowledgePointPanel.vue \
        frontend/src/pages/home/__tests__/KnowledgePointPanel.test.ts
git commit -m "feat(frontend): 新增知识点折叠面板组件"
```

**提交后停下，等待用户确认。**

---

### Task 10: 计划视图与首页改造

**Files:**
- Create: `frontend/src/pages/home/LearningPlanView.vue`
- Modify: `frontend/src/pages/home/index.vue`
- Test: `frontend/src/pages/home/__tests__/LearningPlanView.test.ts`
- Test: `frontend/src/pages/home/__tests__/HomePage.test.ts`

**Interfaces:**
- Consumes: `api.getLearningPlan`、`GoalInput.vue`、`KnowledgePointPanel.vue`、`types.LearningPlan`
- Produces:
  - `LearningPlanView.vue`，props `{ goal: string }`，emits `{ reset: [] }`
  - 首页 `index.vue` 在「推荐流」与「学习计划」两态间切换，`?goal=` 与 URL 同步

- [ ] **Step 1: 写失败的测试 — 计划视图**

创建 `frontend/src/pages/home/__tests__/LearningPlanView.test.ts`：

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { LearningPlan } from '@/types'
import { getLearningPlan } from '@/api/learningPlan'
import LearningPlanView from '../LearningPlanView.vue'

vi.mock('vue-router', () => ({ RouterLink: { template: '<a><slot /></a>' } }))
vi.mock('@/api/learningPlan', () => ({ getLearningPlan: vi.fn() }))
vi.mock('@/api/behavior', () => ({ recordBehavior: vi.fn(() => Promise.resolve(null)) }))

const mockedGetLearningPlan = vi.mocked(getLearningPlan)

const plan: LearningPlan = {
  goal: '我想入门机器学习',
  matched: true,
  groups: [
    {
      knowledge_point: { id: 1, name: '监督学习', description: '', category: null },
      total: 3,
      resources: [],
    },
  ],
}

const fallbackPlan: LearningPlan = {
  goal: 'zzz',
  matched: false,
  groups: [
    {
      knowledge_point: { id: 0, name: '为你找到的相关资源', description: '', category: null },
      total: 1,
      resources: [],
    },
  ],
}

async function mountView(goal = '我想入门机器学习') {
  const wrapper = mount(LearningPlanView, {
    props: { goal },
    global: { plugins: [ElementPlus] },
  })
  await flushPromises()
  return wrapper
}

describe('LearningPlanView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedGetLearningPlan.mockResolvedValue(plan)
  })

  it('按目标拉取计划并渲染知识点分组', async () => {
    const wrapper = await mountView()
    expect(mockedGetLearningPlan).toHaveBeenCalledWith('我想入门机器学习')
    expect(wrapper.text()).toContain('监督学习')
  })

  it('接口失败时展示错误态与重试入口', async () => {
    mockedGetLearningPlan.mockRejectedValue(new Error('加载失败'))
    const wrapper = await mountView()
    expect(wrapper.text()).toContain('加载失败')
    expect(wrapper.text()).toContain('重试')
  })

  it('未命中知识点时展示兜底说明而不是空白', async () => {
    mockedGetLearningPlan.mockResolvedValue(fallbackPlan)
    const wrapper = await mountView('zzz')
    expect(wrapper.text()).toContain('没有找到对应知识点')
    expect(wrapper.text()).toContain('为你找到的相关资源')
  })

  // 竞态：先发的慢响应不能覆盖后发的结果
  it('目标快速变更时只采用最后一次请求的结果', async () => {
    let resolveSlow: (v: LearningPlan) => void = () => {}
    mockedGetLearningPlan
      .mockImplementationOnce(() => new Promise<LearningPlan>((r) => { resolveSlow = r }))
      .mockResolvedValueOnce({ ...plan, groups: [
        { knowledge_point: { id: 2, name: '线性代数', description: '', category: null }, total: 1, resources: [] },
      ] })

    const wrapper = mount(LearningPlanView, { props: { goal: '慢目标' }, global: { plugins: [ElementPlus] } })
    await wrapper.setProps({ goal: '快目标' })
    await flushPromises()

    resolveSlow(plan) // 慢请求此刻才返回
    await flushPromises()

    expect(wrapper.text()).toContain('线性代数')
    expect(wrapper.text()).not.toContain('监督学习')
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm test LearningPlanView`
Expected: 失败 — 找不到 `../LearningPlanView.vue`。

- [ ] **Step 3: 写计划视图实现**

创建 `frontend/src/pages/home/LearningPlanView.vue`：

```vue
<script setup lang="ts">
import { ref, watch } from 'vue'
import { getLearningPlan } from '@/api/learningPlan'
import KnowledgePointPanel from './KnowledgePointPanel.vue'
import type { LearningPlan, LearningPlanGroup } from '@/types'

const props = defineProps<{ goal: string }>()
const emit = defineEmits<{ reset: [] }>()

const plan = ref<LearningPlan | null>(null)
const loading = ref(false)
const error = ref('')

// 请求代次：目标快速变更时，先发的慢响应不得覆盖后发的结果
let generation = 0

async function load(goal: string) {
  const current = ++generation
  loading.value = true
  error.value = ''
  try {
    const data = await getLearningPlan(goal)
    if (current !== generation) return // 已被更新的请求取代，丢弃
    plan.value = data
  } catch (e) {
    if (current !== generation) return
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    if (current === generation) loading.value = false
  }
}

watch(
  () => props.goal,
  (goal) => {
    if (goal) load(goal)
  },
  { immediate: true },
)

// 与 KnowledgePointPanel 的面板 name 保持一致（id 为 0 的兜底组用名字）
function panelKey(group: LearningPlanGroup): string {
  return group.knowledge_point.id ? String(group.knowledge_point.id) : group.knowledge_point.name
}
</script>

<template>
  <div>
    <div v-if="loading" class="mt-8 space-y-3">
      <el-skeleton :rows="4" animated />
    </div>

    <div v-else-if="error" class="py-24 text-center text-sm">
      <p class="text-red-500">{{ error }}</p>
      <el-button class="mt-4" @click="load(props.goal)">重试</el-button>
    </div>

    <template v-else-if="plan">
      <p class="mt-6 text-sm text-ink-secondary">
        <template v-if="plan.matched">
          为「{{ plan.goal }}」规划了 {{ plan.groups.length }} 个知识点，展开可查看推荐资源
        </template>
        <template v-else>没有找到对应知识点，以下是按目标搜索到的资源</template>
      </p>

      <el-empty v-if="plan.groups.every((g) => g.resources.length === 0)" description="暂无相关资源" class="mt-8" />
      <!-- el-collapse 的 model-value 是「展开项 name 的数组」，默认展开第一个知识点 -->
      <el-collapse v-else class="mt-4" :model-value="plan.groups.length ? [panelKey(plan.groups[0])] : []">
        <KnowledgePointPanel v-for="g in plan.groups" :key="panelKey(g)" :group="g" />
      </el-collapse>

      <div class="mt-6">
        <el-button text @click="emit('reset')">换一个目标</el-button>
      </div>
    </template>
  </div>
</template>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm test LearningPlanView`
Expected: 全部 PASS。

- [ ] **Step 5: 改造首页**

改写 `frontend/src/pages/home/index.vue`：

```vue
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getRecommendations } from '@/api/recommendation'
import { recordImpressions } from '@/api/impression'
import { useAuthStore } from '@/stores/auth'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import ResourceCardSkeleton from '@/components/resource/ResourceCardSkeleton.vue'
import InterestPicker from '@/components/profile/InterestPicker.vue'
import GoalInput from './GoalInput.vue'
import LearningPlanView from './LearningPlanView.vue'
import type { Resource } from '@/types'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const resources = ref<Resource[]>([])
const loading = ref(false)
const error = ref('')

// 目标存在即进入「学习计划」态，否则保持原有推荐流。
// 目标是唯一的状态来源，刷新与前进/后退都能恢复。
const goal = computed(() => (typeof route.query.goal === 'string' ? route.query.goal : ''))

// 无兴趣且无历史的新用户，展示冷启动引导
const showInterestPicker = computed(
  () => !!auth.user && (auth.user.interests?.length ?? 0) === 0,
)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const data = await getRecommendations(12)
    resources.value = data.list
    // 列表渲染即曝光：记录本屏展示的资源（CTR 的分母）。失败静默，埋点不影响页面。
    recordImpressions('home', data.list.map((r) => r.id)).catch(() => {})
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function submitGoal(next: string) {
  router.push({ name: 'home', query: { goal: next } })
}

function resetGoal() {
  router.push({ name: 'home' })
}

// 未提交目标时才有推荐流——提交后整页切换为知识点面板（决策 #7 / #8）
onMounted(() => {
  if (!goal.value) load()
})
</script>

<template>
  <div class="mx-auto max-w-7xl px-6 py-10">
    <div class="mx-auto max-w-3xl text-center">
      <h1 class="text-2xl font-bold">想学点什么？</h1>
      <p class="mt-2 text-sm text-ink-secondary">
        说出你的学习目标，我们把它拆成知识点，每个知识点配一组资源
      </p>
      <div class="mt-6">
        <GoalInput :initial-goal="goal" @submit="submitGoal" />
      </div>
    </div>

    <LearningPlanView v-if="goal" :goal="goal" @reset="resetGoal" />

    <template v-else>
      <div class="mt-12 flex items-end justify-between">
        <div>
          <h2 class="text-xl font-bold">发现优质教育资源</h2>
          <p class="mt-2 text-sm text-ink-secondary">为你推荐的精选课程、文章与视频</p>
        </div>
        <RouterLink :to="{ name: 'search' }">
          <el-button round>查看全部</el-button>
        </RouterLink>
      </div>

      <InterestPicker v-if="showInterestPicker" @saved="load" />

      <div v-if="loading" class="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <ResourceCardSkeleton v-for="i in 6" :key="i" />
      </div>
      <div v-else-if="error" class="py-24 text-center text-sm text-red-500">{{ error }}</div>
      <el-empty v-else-if="resources.length === 0" description="暂无推荐内容" class="mt-8" />
      <div v-else class="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
        <ResourceCard
          v-for="(r, i) in resources"
          :key="r.id"
          :resource="r"
          :style="{ animationDelay: `${Math.min(i, 8) * 30}ms` }"
        />
      </div>
    </template>
  </div>
</template>
```

- [ ] **Step 6: 扩展首页测试**

改写 `frontend/src/pages/home/__tests__/HomePage.test.ts`。保留既有 5 个用例，并把 `vue-router` 的 mock 换成可读写的 `route`/`router`（双状态与 URL 同步需要它们）：

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { RecommendationResult, Resource } from '@/types'
import { getRecommendations } from '@/api/recommendation'
import { recordImpressions } from '@/api/impression'
import HomePage from '../index.vue'

// route 可变，用来驱动「推荐流 / 学习计划」两态
const routeMock = vi.hoisted(() => ({ query: {} as Record<string, string> }))
const pushMock = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({
  RouterLink: { template: '<a><slot /></a>' },
  useRoute: () => routeMock,
  useRouter: () => ({ push: pushMock }),
}))
vi.mock('@/api/recommendation', () => ({ getRecommendations: vi.fn() }))
vi.mock('@/api/impression', () => ({ recordImpressions: vi.fn(() => Promise.resolve(null)) }))
vi.mock('@/api/category', () => ({ listCategories: vi.fn(() => Promise.resolve([])) }))
vi.mock('@/api/learningPlan', () => ({ getLearningPlan: vi.fn(() => Promise.resolve({ goal: '', matched: false, groups: [] })) }))
vi.mock('@/api/behavior', () => ({ recordBehavior: vi.fn(() => Promise.resolve(null)) }))

const authMock = vi.hoisted(() => ({ user: null as unknown }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authMock }))

const mockedGetRecommendations = vi.mocked(getRecommendations)
const mockedRecordImpressions = vi.mocked(recordImpressions)

const resource: Resource = {
  id: 1, title: '机器学习入门', description: '面向零基础学习者',
  cover_url: null, type: 'course',
  category: { id: 1, name: '人工智能' },
  tags: ['AI'], metadata: {},
  author: '吴恩达', source_url: null,
  avg_rating: 4.5, view_count: 1024,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-15T10:00:00Z',
}

const recResult: RecommendationResult = { list: [resource], updated_at: '2026-08-01T00:00:00Z' }

async function mountPage() {
  const wrapper = mount(HomePage, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

describe('HomePage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    routeMock.query = {}
    authMock.user = { interests: [1] } // 默认已有兴趣 → 不展示引导
    mockedGetRecommendations.mockResolvedValue(recResult)
  })

  it('调用推荐接口并渲染推荐列表', async () => {
    const wrapper = await mountPage()
    expect(mockedGetRecommendations).toHaveBeenCalledWith(12)
    expect(wrapper.text()).toContain('机器学习入门')
  })

  it('推荐接口失败时展示错误态', async () => {
    mockedGetRecommendations.mockRejectedValue(new Error('加载失败'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('加载失败')
  })

  it('推荐加载成功后上报本屏曝光', async () => {
    await mountPage()
    expect(mockedRecordImpressions).toHaveBeenCalledWith('home', [1])
  })

  it('新用户（无兴趣）展示冷启动兴趣引导', async () => {
    authMock.user = { interests: [] }
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('选择你感兴趣的方向')
  })

  it('已有兴趣的用户不展示兴趣引导', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).not.toContain('选择你感兴趣的方向')
  })

  it('始终展示提问框', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('想学点什么？')
  })

  it('提交目标后跳转到带 ?goal= 的首页', async () => {
    const wrapper = await mountPage()
    await wrapper.find('input').setValue('机器学习')
    await wrapper.find('form').trigger('submit')

    expect(pushMock).toHaveBeenCalledWith({ name: 'home', query: { goal: '机器学习' } })
  })

  it('URL 带 goal 时进入学习计划态，不拉推荐流', async () => {
    routeMock.query = { goal: '机器学习' }
    const wrapper = await mountPage()

    expect(mockedGetRecommendations).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('为你规划')
  })
})
```

- [ ] **Step 7: 跑全量前端验证**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm test && pnpm type-check && pnpm lint`
Expected: 全部通过。

- [ ] **Step 8: 手动验证**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm dev`（后端需在跑）

检查：
- 打开 `/` → 提问框在顶部，下方是原有推荐流
- 输入「我想入门机器学习」→ 页面切换为知识点面板，首个展开，其余折叠
- 刷新页面 → 面板仍在（`?goal=` 生效）
- 点浏览器后退 → 回到推荐流
- 输入一个库内没有的目标 → 显示「没有找到对应知识点…」+ 单一分组，**页面不空白**

- [ ] **Step 9: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add frontend/src/pages/home/LearningPlanView.vue \
        frontend/src/pages/home/index.vue \
        frontend/src/pages/home/__tests__/LearningPlanView.test.ts \
        frontend/src/pages/home/__tests__/HomePage.test.ts
git commit -m "feat(frontend): 首页提问框与知识点面板"
```

**提交后停下，等待用户确认。**

---

### Task 11: 知识点面板曝光打点

**Files:**
- Modify: `frontend/src/api/impression.ts`
- Modify: `frontend/src/pages/home/LearningPlanView.vue`
- Test: `frontend/src/pages/home/__tests__/LearningPlanView.test.ts`

**Interfaces:**
- Consumes: `api.recordImpressions`
- Produces: `ImpressionScene` 联合类型含 `'learning-plan'`；计划视图在计划加载成功后上报本屏曝光

- [ ] **Step 1: 写失败的测试**

在 `frontend/src/pages/home/__tests__/LearningPlanView.test.ts` 中追加 mock 与用例。

先在同文件顶部的 `vi.mock` 区加上：

```ts
import { recordImpressions } from '@/api/impression'
vi.mock('@/api/impression', () => ({ recordImpressions: vi.fn(() => Promise.resolve(null)) }))
const mockedRecordImpressions = vi.mocked(recordImpressions)
```

再追加用例（放在既有 `describe` 内）：

```ts
  it('计划加载成功后按 learning-plan 场景上报本屏曝光', async () => {
    mockedGetLearningPlan.mockResolvedValue({
      ...plan,
      groups: [
        {
          knowledge_point: { id: 1, name: '监督学习', description: '', category: null },
          total: 2,
          resources: [
            { ...resourceFixture(1) },
            { ...resourceFixture(2) },
          ],
        },
      ],
    })

    await mountView()

    expect(mockedRecordImpressions).toHaveBeenCalledWith('learning-plan', [1, 2])
  })

  it('没有资源时不发曝光请求', async () => {
    await mountView()
    expect(mockedRecordImpressions).not.toHaveBeenCalled()
  })
```

并在同文件内加一个最小资源工厂（若无）：

```ts
function resourceFixture(id: number): Resource {
  return {
    id, title: `资源 ${id}`, description: '简介',
    cover_url: null, type: 'video',
    category: null, tags: [], metadata: {},
    author: null, source_url: null,
    avg_rating: 4, view_count: 10,
    created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-01T08:00:00Z',
  }
}
```

同时把顶部的类型 import 改成 `import type { LearningPlan, Resource } from '@/types'`。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm test LearningPlanView`
Expected: 新用例失败 — `recordImpressions` 未被调用。

- [ ] **Step 3: 扩展场景联合类型**

修改 `frontend/src/api/impression.ts`：

```ts
// 曝光场景，与后端 ResourceImpression.Scene 对应
export type ImpressionScene = 'home' | 'search' | 'detail' | 'learning-plan'
```

- [ ] **Step 4: 在计划视图上报曝光**

修改 `frontend/src/pages/home/LearningPlanView.vue`，在 import 区加：

```ts
import { recordImpressions } from '@/api/impression'
```

并在 `load` 函数里、`plan.value = data` 之后加：

```ts
    plan.value = data
    // 面板展开即曝光：记录本屏知识点下的资源。用独立场景，避免扰动首页 CTR 口径。
    // 失败静默，埋点不影响页面。
    const shown = data.groups.flatMap((g) => g.resources.map((r) => r.id))
    recordImpressions('learning-plan', shown).catch(() => {})
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm test LearningPlanView`
Expected: 全部 PASS。

- [ ] **Step 6: 跑全量前端验证**

Run: `cd /home/shionyori/project/edurec-platform/frontend && pnpm test && pnpm type-check && pnpm lint`
Expected: 全部通过。

- [ ] **Step 7: 手动验证打点不干扰既有 CTR 口径**

```bash
# 本机没装 mysql 客户端，走容器内的客户端（已核实 edurec-mysql 容器可用）
docker exec edurec-mysql mysql -uroot -proot edurec -e \
  "SELECT scene, COUNT(*) FROM resource_impressions GROUP BY scene;"
```

Expected: 出现 `home` 与新出现的 `learning-plan` 两类行，`home` 的统计不受本改动影响。

- [ ] **Step 8: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add frontend/src/api/impression.ts \
        frontend/src/pages/home/LearningPlanView.vue \
        frontend/src/pages/home/__tests__/LearningPlanView.test.ts
git commit -m "feat(frontend): 知识点面板曝光打点"
```

**提交后停下，等待用户确认。**

---

### Task 12: 文档回写

**Files:**
- Modify: `docs/recsys-design.md`
- Modify: `docs/api-design.md`
- Modify: `docs/design.md`
- Modify: `README.md`

**Interfaces:**
- Consumes: 已完成的实现与 spec §9
- Produces: 四份文档与代码一致

- [ ] **Step 1: 建分支**

```bash
cd /home/shionyori/project/edurec-platform
git checkout -b docs/recsys-boundary-revision
```

- [ ] **Step 2: 回写 recsys-design.md 的边界修订**

在 `docs/recsys-design.md` 的 §1.3 表格中，把「学习路径/知识图谱」一行的关系列改为：

```
| 学习路径/知识图谱 | "按什么顺序学" | **部分纳入**：只做「目标 → 知识点 → 分组资源」的**分块**，不做知识点之间的先后/依赖排序，不碰掌握度与测评。详见 `docs/superpowers/specs/2026-10-13-learning-goal-knowledge-points-design.md` |
```

并在 §7「明确不做（现阶段）」列表末尾追加一条：

```
- 知识点之间的顺序 / 依赖 / 掌握度建模（知识点层只做分块，路径排序仍属另一选题）
```

同时在文件顶部「实施状态」表格后追加一段：

```
### 2026-10 增补：学习目标 · 知识点推荐层

在推荐链条最前面加了一层入口：用户表达学习目标 → 系统给出知识点 → 每个知识点挂一组资源。
**知识点不是新算法，而是对现有推荐信号的一次分桶**：知识点由 `resources.tags` 聚合而来，
组内资源用知识点标签过滤资源池，再按「engine 推荐序 > avg_rating > view_count」排序。
平台侧依然不做推理，§7 的其余排除项（ANN / 实时推理 / 深度排序 / 掌握度）保持不变。
```

- [ ] **Step 3: 补 API 文档**

在 `docs/api-design.md` 的「### 8. 管理模块」之后、「## 附录：接口总览」之前，插入新章节（并从 §6.1 的推荐接口后交叉引用）：

```markdown
### 9. 学习计划模块

#### 9.1 生成学习计划

```
GET /api/v1/learning-plan
```

**需要认证：是**

**Query Parameters**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| goal | string | 是 | 学习目标，2–100 字符 |
| knowledge_points | int | 否 | 返回的知识点数，默认 8，最大 20 |

**Response (200)**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "goal": "我想入门机器学习",
    "matched": true,
    "groups": [
      {
        "knowledge_point": { "id": 3, "name": "监督学习", "description": "", "category": { "id": 1, "name": "人工智能" } },
        "total": 12,
        "resources": []
      }
    ]
  }
}
```

| 字段 | 说明 |
|------|------|
| matched | false 表示未匹配到知识点，groups 为兜底单一分组 |
| groups[].total | 该知识点关联的资源总数（不受组内展示上限影响） |
| groups[].resources | 组内资源，最多 6 条，已按「engine 推荐序 > 平均分 > 浏览量」排序 |

**错误**

| 状态码 | 说明 |
|--------|------|
| 400 | goal 为空或超过 100 字符 |
| 401 | 未认证 |

> 知识点由 `resources.tags` 聚合而来（离线导入，见 `cmd/import_knowledge_points`），
> 目标匹配为确定性子串匹配；匹配不到时兜底为按目标的关键词搜索，不返回空白。
> 知识点面板的资源曝光使用独立场景 `learning-plan`，与首页的 `home` 场景分开统计。
```

并把附录的接口总览表补上 `GET /api/v1/learning-plan` 一行。

- [ ] **Step 4: 修正 design.md 漂移并补页面说明**

在 `docs/design.md`：

1. §3.2 技术选型表里「迁移工具」一行改为：

```
| 迁移工具 | **GORM AutoMigrate** | 启动时自动建表/改表，见 `internal/database/mysql.go` |
```

2. §6 页面设计表里「首页」一行说明改为：

```
| 首页 | `/` | 提问框（学习目标）+ 推荐流；提交目标后切换为知识点面板（`?goal=` 同步） | 是 |
```

3. 决策记录表追加一行：

```
| 33 | 学习目标·知识点层 | 分块而非排序（落地）：用户表达学习目标 → 知识点 → 每组资源。知识点由 `resources.tags` 聚合而来（`cmd/import_knowledge_points`），组内资源用知识点标签过滤资源池、按既有推荐信号排序。不做知识点之间的先后/依赖排序，不做掌握度；平台侧仍不做推理 |
```

- [ ] **Step 5: 更新 README**

在 `README.md` 的「### 已实现」表格中追加一行：

```
| 学习计划 | 学习目标 → 知识点 → 每组资源（知识点由资源标签聚合，组内复用推荐信号排序） |
```

并在「## 脚本」表格后补一句说明命令：

```markdown
知识点层需要先聚合一次（也可通过 `bash scripts/start.sh` 起服务后单独执行）：

```bash
cd backend && go run ./cmd/import_knowledge_points --from-tags --preview  # 先人工检视聚合质量
cd backend && go run ./cmd/import_knowledge_points --from-tags            # 确认后落库
```
```

- [ ] **Step 6: 确认没有遗留矛盾**

Run:
```bash
cd /home/shionyori/project/edurec-platform
grep -n "golang-migrate" docs/*.md README.md ; echo "--- 上面应为空 ---"
grep -n "另一选题，本方案不做" docs/recsys-design.md ; echo "--- 应只剩知识点顺序相关的表述 ---"
```
Expected: 第一条无输出；第二条若仍命中，确认上下文已改为「只做分块、不做排序」。

- [ ] **Step 7: 把本轮的范围收窄回写进 spec，避免两份文档打架**

修改 `docs/superpowers/specs/2026-10-13-learning-goal-knowledge-points-design.md`：

1. §8.1 的命令清单里，把 `--file` 一行标注为延期：

```
go run ./cmd/import_knowledge_points --from-tags            # 从 resources.tags 聚合（本轮实现）
go run ./cmd/import_knowledge_points --from-tags --preview  # 不连库，打印将要导入的知识点（本轮实现）
# --file data/knowledge_points.json                          # 手工/LLM 产出（本轮不实现，见 §10 演进）
```

2. §8.2 第 6 步删去「写入 `data/knowledge_points.json`」的中间产物（本轮不落地中间文件，聚合结果直接落库）：

```
6. 直接落库（本轮不产生中间 JSON 文件；契约见 §8.3，实现时再落地）
```

3. §8.3 末尾补一句：

```
> 本契约在**有第二个产出方时**才实现读取端。本轮只有 `--from-tags` 一个产出方，
> 写一个无法端到端验证的读取器只会留下没被测过的代码（YAGNI）。缝保留在
> 「聚合是独立纯函数、落库是独立 service 方法」这两个接缝上。
```

- [ ] **Step 8: 提交**

```bash
cd /home/shionyori/project/edurec-platform
git add docs/recsys-design.md docs/api-design.md docs/design.md README.md \
        docs/superpowers/specs/2026-10-13-learning-goal-knowledge-points-design.md
git commit -m "docs: 回写知识点层对推荐系统边界的修订"
```

**提交后停下，等待用户确认。**

---

## 完成标准

全部 13 个 Task（Task 4b 是人工闸门后新增的）完成后：

```bash
cd /home/shionyori/project/edurec-platform/backend  && go test ./... && go vet ./...
cd ../frontend && pnpm test && pnpm type-check && pnpm lint
```

两者全绿，且下列手动场景通过：

- [ ] `cmd/import_knowledge_points --from-tags --preview` 产出的知识点经人工判断可接受
- [ ] 输入「我想入门机器学习」返回多个知识点面板，每组有资源
- [ ] 输入库内无对应的目标 → 兜底文案 + 单一分组，页面不空白
- [ ] 空白/超长目标 → 前端即时提示，后端 400
- [ ] 刷新（带 `?goal=`）恢复面板；后退回到推荐流
- [ ] 首页未提交时推荐流与曝光打点行为与改造前一致
- [ ] `resource_impressions` 出现 `learning-plan` 场景，`home` 口径未受影响
