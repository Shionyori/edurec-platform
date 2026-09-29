package handler

import (
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/middleware"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type RecommendationHandler struct {
	recommendations *service.RecommendationService
	importer        *service.RecommendationImportService
	runs            repository.RecommendationRunRepository
}

func NewRecommendationHandler(
	recommendations *service.RecommendationService,
	importer *service.RecommendationImportService,
	runs repository.RecommendationRunRepository,
) *RecommendationHandler {
	return &RecommendationHandler{recommendations: recommendations, importer: importer, runs: runs}
}

// recommendationItem 在资源条目上附带推荐理由（可解释性）
type recommendationItem struct {
	resourceListItem
	Reason string `json:"reason,omitempty"`
}

// Get 获取个性化推荐（GET /api/v1/recommendations）
func (h *RecommendationHandler) Get(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
		return
	}

	limit, ok := queryPositiveInt(c, "limit", 20)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	if limit == 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	result, err := h.recommendations.Get(userID, limit)
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]recommendationItem, 0, len(result.List))
	for i := range result.List {
		item := recommendationItem{resourceListItem: toResourceListItem(&result.List[i])}
		if reason, ok := result.Reasons[result.List[i].ID]; ok {
			item.Reason = reason
		}
		items = append(items, item)
	}

	response.OK(c, gin.H{
		"list":       items,
		"updated_at": time.Unix(result.UpdatedAt, 0).UTC().Format(time.RFC3339),
		"run_id":     result.RunID,
	})
}

// Import 从 engine 输出文件导入推荐缓存（POST /api/v1/admin/recommendations/import，管理员）
func (h *RecommendationHandler) Import(c *gin.Context) {
	result, err := h.importer.Import()
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, result)
}

// ListRuns 列出最近的推荐运行记录（GET /api/v1/admin/recommendation-runs，管理员）
func (h *RecommendationHandler) ListRuns(c *gin.Context) {
	limit, ok := queryPositiveInt(c, "limit", 20)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	if limit == 0 {
		limit = 20
	}
	runs, err := h.runs.List(limit)
	if err != nil {
		handleError(c, apperror.Internal(err))
		return
	}
	response.OK(c, gin.H{"list": runs})
}
