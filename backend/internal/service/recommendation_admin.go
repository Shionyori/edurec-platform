package service

import (
	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// RecommendationAdminService 推荐运营/评估的读模型。
//
// 这些能力（运行记录、曝光/点击统计）此前直接挂在 Handler 上，Handler 因此同时
// 依赖 repository 包并与推荐读取服务并列，破坏了 `handler → service → repository`
// 的单向分层。收进 service 后：
//   - Handler 只做参数校验与序列化；
//   - CTR 计算等业务口径只有一处；
//   - 未来给运营接口加缓存/权限/聚合时，不需要再动 HTTP 层。
type RecommendationAdminService struct {
	runs  repository.RecommendationRunRepository
	stats repository.StatsRepository
}

func NewRecommendationAdminService(
	runs repository.RecommendationRunRepository,
	stats repository.StatsRepository,
) *RecommendationAdminService {
	return &RecommendationAdminService{runs: runs, stats: stats}
}

// RecommendationStatsView 看板视图：在原始计数上补一个归一化指标。
type RecommendationStatsView struct {
	Impressions         int64
	Clicks              int64
	Favorites           int64
	Views               int64
	RecommendationUsers int64
	ClickThroughRate    float64
}

// ListRuns 返回最近的推理运行记录。
func (s *RecommendationAdminService) ListRuns(limit int) ([]model.RecommendationRun, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	runs, err := s.runs.List(limit)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return runs, nil
}

// Stats 返回推荐效果统计；CTR = clicks / impressions，曝光为 0 时记 0。
func (s *RecommendationAdminService) Stats() (*RecommendationStatsView, error) {
	stats, err := s.stats.RecommendationStats()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	view := &RecommendationStatsView{
		Impressions:         stats.Impressions,
		Clicks:              stats.Clicks,
		Favorites:           stats.Favorites,
		Views:               stats.Views,
		RecommendationUsers: stats.RecommendationUsers,
	}
	if stats.Impressions > 0 {
		view.ClickThroughRate = float64(stats.Clicks) / float64(stats.Impressions)
	}
	return view, nil
}
