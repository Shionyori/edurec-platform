package service

import (
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// RecommendationImportService 将 edurec-engine 输出的推荐结果导入缓存表。
//
// 主文件 `{ "<user_id>": [<resource_id>, ...] }` 只含排名；可选旁挂信封
// `recommendations.meta.json` 提供 run_id / 生成时间 / 模型版本 / 分数 / 理由。
// 读信封是为了让「这批推荐从哪来」在库层面可追溯。
type RecommendationImportService struct {
	recommendations repository.RecommendationRepository
	runs            repository.RecommendationRunRepository
	users           repository.UserRepository
	resources       repository.ResourceRepository
	filePath        string
}

func NewRecommendationImportService(
	recommendations repository.RecommendationRepository,
	runs repository.RecommendationRunRepository,
	users repository.UserRepository,
	resources repository.ResourceRepository,
	filePath string,
) *RecommendationImportService {
	return &RecommendationImportService{
		recommendations: recommendations,
		runs:            runs,
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

// recommendationEnvelope 旁挂信封（可选）：与主文件同目录、主文件名加 .meta.json。
// 缺失、字段不全都不影响导入——平台只存不解释这些元信息。
type recommendationEnvelope struct {
	ContractVersion int    `json:"contract_version"`
	GeneratedAt     int64  `json:"generated_at"`
	RunID           string `json:"run_id"`
	SnapshotRunID   string `json:"snapshot_run_id"`
	Model           struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Encoder string `json:"encoder"`
	} `json:"model"`
	TopN       int                 `json:"top_n"`
	UsersCount int                 `json:"users_count"`
	Reasons    map[string][]string `json:"reasons"`
}

// envelopePath 由主文件路径推出信封路径：a.json → a.meta.json
func envelopePath(filePath string) string {
	if strings.HasSuffix(filePath, ".json") {
		return strings.TrimSuffix(filePath, ".json") + ".meta.json"
	}
	return filePath + ".meta.json"
}

func (s *RecommendationImportService) readEnvelope() recommendationEnvelope {
	var env recommendationEnvelope
	data, err := os.ReadFile(envelopePath(s.filePath))
	if err != nil {
		return env // 信封可选：缺失即零值
	}
	_ = json.Unmarshal(data, &env) // 损坏也降级为零值，不阻断导入
	return env
}

// Import 读取 engine 输出文件并写入推荐缓存表。
// engine 的用户/资源 ID 与平台库按数值匹配——只导入真实存在的用户与资源，其余跳过。
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

	envelope := s.readEnvelope()

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

		// 理由数组与 rids 一一对应，随资源过滤同步剔除
		rawReasons := envelope.Reasons[uidStr]
		valid := make([]uint, 0, len(rids))
		reasons := make([]string, 0, len(rids))
		for i, rid := range rids {
			if _, ok := resourceSet[rid]; ok {
				valid = append(valid, rid)
				if i < len(rawReasons) {
					reasons = append(reasons, rawReasons[i])
				} else {
					reasons = append(reasons, "")
				}
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
		reasonsJSON, err := json.Marshal(reasons)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		if _, err := s.recommendations.Replace(repository.RecommendationUpsert{
			UserID:      uid,
			ResourceIDs: string(idsJSON),
			RunID:       envelope.RunID,
			Reasons:     string(reasonsJSON),
			Now:         now,
		}); err != nil {
			return nil, apperror.Internal(err)
		}
		result.ImportedUsers++
		result.ImportedResources += len(valid)
	}

	s.recordRun(envelope, result, now)
	return result, nil
}

// recordRun 落一条运行记录。写失败只告警不报错：推荐已导入，不能让审计记录失败回滚结果。
func (s *RecommendationImportService) recordRun(
	env recommendationEnvelope, result *ImportResult, now int64,
) {
	generatedAt := env.GeneratedAt
	if generatedAt == 0 {
		generatedAt = now
	}
	run := &model.RecommendationRun{
		RunID:             env.RunID,
		SnapshotRunID:     env.SnapshotRunID,
		ModelName:         env.Model.Name,
		ModelVersion:      env.Model.Version,
		Encoder:           env.Model.Encoder,
		GeneratedAt:       generatedAt,
		TopN:              env.TopN,
		UsersCount:        env.UsersCount,
		ImportedUsers:     result.ImportedUsers,
		SkippedUsers:      result.SkippedUsers,
		ImportedResources: result.ImportedResources,
		SkippedResources:  result.SkippedResources,
		CreatedAt:         now,
	}
	if err := s.runs.Create(run); err != nil {
		slog.Warn("记录推荐运行失败", "error", err)
	}
}
