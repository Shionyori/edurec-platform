package handler

import (
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/middleware"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type RecommendationHandler struct {
	recommendations *service.RecommendationService
}

func NewRecommendationHandler(recommendations *service.RecommendationService) *RecommendationHandler {
	return &RecommendationHandler{recommendations: recommendations}
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

	items := make([]resourceListItem, 0, len(result.List))
	for i := range result.List {
		items = append(items, toResourceListItem(&result.List[i]))
	}

	response.OK(c, gin.H{
		"list":       items,
		"updated_at": time.Unix(result.UpdatedAt, 0).UTC().Format(time.RFC3339),
	})
}
