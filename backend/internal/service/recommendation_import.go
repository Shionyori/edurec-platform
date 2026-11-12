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

const (
	// IDLookupChunk 限制单条 SQL 的 IN 参数个数。
	// 全量推荐文件的资源 ID 可能达到「用户数 × top_n」量级（数万～数十万），
	// 一次性 `IN (...)` 会撞上 max_allowed_packet / 占位符上限。
	IDLookupChunk = 1000
	// ImportWriteChunk 单事务写入的用户数。整份文件一个事务会长时间持锁；
	// 按批提交既控制锁范围，又保证每一批内部原子。
	ImportWriteChunk = 500
)

// uniqueUint 去重但保留首次出现顺序；用于「存在性查询」的输入归一。
func uniqueUint(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func chunkUint(ids []uint, size int) [][]uint {
	if size <= 0 {
		size = len(ids)
	}
	chunks := make([][]uint, 0, (len(ids)+size-1)/size)
	for start := 0; start < len(ids); start += size {
		end := start + size
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[start:end])
	}
	return chunks
}

func chunkUpserts(inputs []repository.RecommendationUpsert,
	size int) [][]repository.RecommendationUpsert {
	if size <= 0 {
		size = len(inputs)
	}
	chunks := make([][]repository.RecommendationUpsert, 0, (len(inputs)+size-1)/size)
	for start := 0; start < len(inputs); start += size {
		end := start + size
		if end > len(inputs) {
			end = len(inputs)
		}
		chunks = append(chunks, inputs[start:end])
	}
	return chunks
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

	// 收集全部用户与资源 ID，用于分批查询存在性
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

	userSet, err := s.existingUserIDs(userIDs)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	resourceSet, err := s.existingResourceIDs(resourceIDs)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	now := time.Now().Unix()
	result := &ImportResult{}
	pending := make([]repository.RecommendationUpsert, 0, len(raw))

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
		pending = append(pending, repository.RecommendationUpsert{
			UserID:      uid,
			ResourceIDs: string(idsJSON),
			RunID:       envelope.RunID,
			Reasons:     string(reasonsJSON),
			Now:         now,
		})
		result.ImportedUsers++
		result.ImportedResources += len(valid)
	}

	// 分批事务写入：每一批内部原子，避免中途失败留下半批新、半批旧的状态。
	for _, batch := range chunkUpserts(pending, ImportWriteChunk) {
		if err := s.recommendations.ReplaceMany(batch); err != nil {
			return nil, apperror.Internal(err)
		}
	}

	s.recordRun(envelope, result, now)
	return result, nil
}

// existingUserIDs 分批查询哪些用户真实存在，返回集合。
func (s *RecommendationImportService) existingUserIDs(ids []uint) (map[uint]struct{}, error) {
	set := make(map[uint]struct{}, len(ids))
	for _, chunk := range chunkUint(uniqueUint(ids), IDLookupChunk) {
		users, err := s.users.FindByIDs(chunk)
		if err != nil {
			return nil, err
		}
		for i := range users {
			set[users[i].ID] = struct{}{}
		}
	}
	return set, nil
}

// existingResourceIDs 分批查询哪些资源真实存在，返回集合。
func (s *RecommendationImportService) existingResourceIDs(ids []uint) (map[uint]struct{}, error) {
	set := make(map[uint]struct{}, len(ids))
	for _, chunk := range chunkUint(uniqueUint(ids), IDLookupChunk) {
		resources, err := s.resources.FindByIDs(chunk)
		if err != nil {
			return nil, err
		}
		for i := range resources {
			set[resources[i].ID] = struct{}{}
		}
	}
	return set, nil
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
