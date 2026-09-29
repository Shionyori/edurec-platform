package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

const (
	defaultRecommendLimit = 20
	maxRecommendLimit     = 50
)

type RecommendationService struct {
	recommendations repository.RecommendationRepository
	resources       repository.ResourceRepository
	users           repository.UserRepository
}

func NewRecommendationService(recommendations repository.RecommendationRepository, resources repository.ResourceRepository, users repository.UserRepository) *RecommendationService {
	return &RecommendationService{recommendations: recommendations, resources: resources, users: users}
}

// RecommendationResult 推荐结果
type RecommendationResult struct {
	List      []model.Resource
	Reasons   map[uint]string // 资源 ID → 推荐理由（缺省为空）
	RunID     string          // 产出本次结果的运行标识（平台兜底时为空）
	UpdatedAt int64           // Unix 时间戳，本次推荐生成时间
}

// Get 获取用户个性化推荐：优先命中缓存，未命中时兜底生成并写入缓存。
func (s *RecommendationService) Get(userID uint, limit int) (*RecommendationResult, error) {
	if limit <= 0 {
		limit = defaultRecommendLimit
	}
	if limit > maxRecommendLimit {
		limit = maxRecommendLimit
	}

	now := time.Now().Unix()

	// 1. 优先读缓存
	rec, err := s.recommendations.FindByUserID(userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	// 2. 缓存命中：按缓存中的顺序返回资源
	if rec != nil {
		resourceIDs, err := parseResourceIDs(rec.ResourceIDs)
		if err != nil {
			// 缓存数据损坏，视为未命中重新生成
			rec = nil
		} else {
			resources, err := s.resources.FindByIDs(resourceIDs)
			if err != nil {
				return nil, apperror.Internal(err)
			}
			ordered := orderResources(resources, resourceIDs)
			if len(ordered) > limit {
				ordered = ordered[:limit]
			}
			return &RecommendationResult{
				List:      ordered,
				Reasons:   parseReasons(rec.Reasons, resourceIDs),
				RunID:     rec.RunID,
				UpdatedAt: rec.UpdatedAt,
			}, nil
		}
	}

	// 3. 缓存未命中：兜底生成（优先兴趣分类，否则按评分降序取热门）并写入缓存
	fallback, err := s.fallbackResources(userID, limit)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	resourceIDs := make([]uint, 0, len(fallback))
	for i := range fallback {
		resourceIDs = append(resourceIDs, fallback[i].ID)
	}
	idsJSON, err := json.Marshal(resourceIDs)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	saved, err := s.recommendations.Replace(repository.RecommendationUpsert{
		UserID:      userID,
		ResourceIDs: string(idsJSON),
		Now:         now,
	})
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return &RecommendationResult{List: fallback, UpdatedAt: saved.UpdatedAt}, nil
}

// fallbackResources 冷启动兜底：优先按用户兴趣分类取高分资源，否则按评分降序取热门。
// 兴趣读取失败不阻断兜底——宁可退化为热门，也不能返回空列表。
func (s *RecommendationService) fallbackResources(userID uint, limit int) ([]model.Resource, error) {
	if user, err := s.users.FindByID(userID); err == nil && user != nil {
		if interests := parseUserInterests(user.Interests); len(interests) > 0 {
			items, err := s.resources.ListTopByCategories(interests, limit)
			if err != nil {
				return nil, err
			}
			if len(items) > 0 {
				return items, nil
			}
		}
	}

	fallback, err := s.resources.List(repository.ResourceListQuery{
		Page:     1,
		PageSize: limit,
		Sort:     "rating",
	})
	if err != nil {
		return nil, err
	}
	return fallback.Items, nil
}

// parseResourceIDs 解析缓存的资源 ID JSON 数组
func parseResourceIDs(raw string) ([]uint, error) {
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// parseReasons 解析缓存里的推荐理由，返回 资源ID → 理由 的映射。
// 理由数组与 resource_ids 一一对应；长度不一致或损坏时按能对齐的部分返回，不影响推荐主流程。
func parseReasons(raw string, ids []uint) map[uint]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var reasons []string
	if err := json.Unmarshal([]byte(raw), &reasons); err != nil {
		return nil
	}
	out := make(map[uint]string, len(ids))
	for i, id := range ids {
		if i < len(reasons) && reasons[i] != "" {
			out[id] = reasons[i]
		}
	}
	return out
}

// orderResources 按缓存中的 ID 顺序重排资源，已删除的资源自动跳过
func orderResources(resources []model.Resource, ids []uint) []model.Resource {
	byID := make(map[uint]model.Resource, len(resources))
	for _, r := range resources {
		byID[r.ID] = r
	}
	ordered := make([]model.Resource, 0, len(resources))
	for _, id := range ids {
		if r, ok := byID[id]; ok {
			ordered = append(ordered, r)
		}
	}
	return ordered
}
