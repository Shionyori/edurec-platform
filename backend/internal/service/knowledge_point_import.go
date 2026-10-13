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
