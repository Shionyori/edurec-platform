package service

import (
	"encoding/json"
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
}

func NewRecommendationService(recommendations repository.RecommendationRepository, resources repository.ResourceRepository) *RecommendationService {
	return &RecommendationService{recommendations: recommendations, resources: resources}
}

// RecommendationResult 推荐结果
type RecommendationResult struct {
	List      []model.Resource
	UpdatedAt int64 // Unix 时间戳，本次推荐生成时间
}

// Get 获取用户个性化推荐：优先命中缓存，未命中时兜底生成并写入缓存。
// TODO(engine): engine 接入（路线 B/A）后，兜底生成改为消费 edurec-engine 的结果。
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
			return &RecommendationResult{List: ordered, UpdatedAt: rec.UpdatedAt}, nil
		}
	}

	// 3. 缓存未命中：兜底生成（按评分降序取热门资源）并写入缓存
	fallback, err := s.resources.List(repository.ResourceListQuery{
		Page:     1,
		PageSize: limit,
		Sort:     "rating",
	})
	if err != nil {
		return nil, apperror.Internal(err)
	}

	resourceIDs := make([]uint, 0, len(fallback.Items))
	for i := range fallback.Items {
		resourceIDs = append(resourceIDs, fallback.Items[i].ID)
	}
	idsJSON, err := json.Marshal(resourceIDs)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	saved, err := s.recommendations.Replace(userID, string(idsJSON), now)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return &RecommendationResult{List: fallback.Items, UpdatedAt: saved.UpdatedAt}, nil
}

// parseResourceIDs 解析缓存的资源 ID JSON 数组
func parseResourceIDs(raw string) ([]uint, error) {
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, err
	}
	return ids, nil
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
