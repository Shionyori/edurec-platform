package service

import (
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// RecommendationImportService 将 edurec-engine 输出的推荐结果导入缓存表（路线 B，engine 零改动）
type RecommendationImportService struct {
	recommendations repository.RecommendationRepository
	users           repository.UserRepository
	resources       repository.ResourceRepository
	filePath        string
}

func NewRecommendationImportService(
	recommendations repository.RecommendationRepository,
	users repository.UserRepository,
	resources repository.ResourceRepository,
	filePath string,
) *RecommendationImportService {
	return &RecommendationImportService{
		recommendations: recommendations,
		users:           users,
		resources:       resources,
		filePath:        filePath,
	}
}

// ImportResult 导入统计
type ImportResult struct {
	ImportedUsers     int `json:"imported_users"`
	SkippedUsers      int `json:"skipped_users"`
	ImportedResources int `json:"imported_resources"`
	SkippedResources  int `json:"skipped_resources"`
}

// Import 读取 engine 输出文件（{ "<user_id>": [<resource_id>, ...] }）并写入推荐缓存表。
// engine 的用户/资源 ID 是它模拟数据集的内部编号，与平台库不一一对应，
// 因此只导入 platform 数据库中真实存在的用户与资源（数值 ID 相同才匹配），其余跳过。
func (s *RecommendationImportService) Import() (*ImportResult, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	raw := map[string][]uint{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, apperror.Internal(err)
	}
	if len(raw) == 0 {
		return &ImportResult{}, nil
	}

	// 收集全部用户与资源 ID，用于一次性查询存在性
	userIDs := make([]uint, 0, len(raw))
	resourceIDs := make([]uint, 0)
	for uidStr, rids := range raw {
		id, err := strconv.ParseUint(uidStr, 10, 64)
		if err != nil {
			continue // 非数字 key，忽略
		}
		userIDs = append(userIDs, uint(id))
		resourceIDs = append(resourceIDs, rids...)
	}

	existingUsers, err := s.users.FindByIDs(userIDs)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	userSet := make(map[uint]struct{}, len(existingUsers))
	for i := range existingUsers {
		userSet[existingUsers[i].ID] = struct{}{}
	}

	existingResources, err := s.resources.FindByIDs(resourceIDs)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	resourceSet := make(map[uint]struct{}, len(existingResources))
	for i := range existingResources {
		resourceSet[existingResources[i].ID] = struct{}{}
	}

	now := time.Now().Unix()
	result := &ImportResult{}

	for uidStr, rids := range raw {
		id, err := strconv.ParseUint(uidStr, 10, 64)
		if err != nil {
			continue
		}
		uid := uint(id)
		if _, ok := userSet[uid]; !ok {
			result.SkippedUsers++
			continue
		}

		valid := make([]uint, 0, len(rids))
		for _, rid := range rids {
			if _, ok := resourceSet[rid]; ok {
				valid = append(valid, rid)
			} else {
				result.SkippedResources++
			}
		}
		if len(valid) == 0 {
			result.SkippedUsers++
			continue
		}

		idsJSON, err := json.Marshal(valid)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		if _, err := s.recommendations.Replace(uid, string(idsJSON), now); err != nil {
			return nil, apperror.Internal(err)
		}
		result.ImportedUsers++
		result.ImportedResources += len(valid)
	}

	return result, nil
}
