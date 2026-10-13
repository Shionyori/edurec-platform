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
