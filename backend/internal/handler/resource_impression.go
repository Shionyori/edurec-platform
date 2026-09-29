package handler

import (
	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/middleware"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type ResourceImpressionHandler struct {
	impressions *service.ResourceImpressionService
}

func NewResourceImpressionHandler(impressions *service.ResourceImpressionService) *ResourceImpressionHandler {
	return &ResourceImpressionHandler{impressions: impressions}
}

type recordImpressionsRequest struct {
	Scene       string `json:"scene" binding:"required"`
	ResourceIDs []uint `json:"resource_ids" binding:"required"`
}

// Record 上报一次推荐展示（批量）。列表渲染后由前端触发。
func (h *ResourceImpressionHandler) Record(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
		return
	}

	var req recordImpressionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	if err := h.impressions.Record(c.Request.Context(), userID, req.Scene, req.ResourceIDs); err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, nil)
}
