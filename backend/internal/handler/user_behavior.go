package handler

import (
	"strconv"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/middleware"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type UserBehaviorHandler struct {
	behaviors *service.UserBehaviorService
}

func NewUserBehaviorHandler(behaviors *service.UserBehaviorService) *UserBehaviorHandler {
	return &UserBehaviorHandler{behaviors: behaviors}
}

type reportBehaviorRequest struct {
	Action string `json:"action" binding:"required,oneof=view click favorite"`
}

func (h *UserBehaviorHandler) Record(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
		return
	}

	resourceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || resourceID == 0 {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	var req reportBehaviorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	if err := h.behaviors.Record(c.Request.Context(), userID, uint(resourceID), req.Action); err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, nil)
}
