package service

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

// AggregateOptions 标签聚合的配置。用结构体而非散参：字段还会增加，
// 散参每次都要改所有调用点。
type AggregateOptions struct {
	// Stoplist 丢弃这些标签（平台徽章、人名、元信息词等无区分度的噪音）
	Stoplist []string
	// MinCount 标签至少要出现多少次才算知识点
	MinCount int
	// Max 知识点上限，<=0 表示不限
	Max int
	// Synonyms 把别名归到规范名，键是规范名（如 {"Go": ["Golang", "go"]}）。
	// 拉丁文即便不在表里，也会按小写再归并一轮，避免同一个词因大小写拆成多条。
	Synonyms map[string][]string
}

// TagAggregate 一个标签的聚合结果，即一个候选知识点
type TagAggregate struct {
	Name       string
	Count      int
	CategoryID uint
	// Aliases 折叠进这个知识点的其它写法（大小写变体 + synonyms 里配的别名），
	// 去重、不含 Name 本身、已排序。落库进 knowledge_points.aliases，
	// 并参与资源检索（见 ResourceListQuery.AnyTags）。
	Aliases []string
}

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

	// 别名（已小写）→ 规范名
	aliasToCanonical := make(map[string]string)
	for canonical, aliases := range opts.Synonyms {
		for _, alias := range aliases {
			aliasToCanonical[strings.ToLower(strings.TrimSpace(alias))] = canonical
		}
	}

	type spelling struct {
		original string
		count    int
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

	candidates := AggregateKnowledgePointsFromTags(resources, AggregateOptions{
		Stoplist: effective.TagStoplist,
		MinCount: effective.MinResourceCount,
		Max:      effective.MaxKnowledgePoints,
		Synonyms: effective.Synonyms,
	})

	result := &KnowledgePointImportResult{Candidates: len(candidates)}
	if !write || len(candidates) == 0 {
		return result, candidates, nil
	}

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
