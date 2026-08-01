package handler

import (
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/middleware"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) Me(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	isAdmin, err := h.users.IsAdmin(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.OK(c, toUserResponse(user, isAdmin))
}

type updateProfileRequest struct {
	DisplayName *string `json:"display_name" binding:"omitempty,max=128"`
	AvatarURL   *string `json:"avatar_url" binding:"omitempty,max=512"`
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	if req.DisplayName != nil {
		value := strings.TrimSpace(*req.DisplayName)
		req.DisplayName = &value
	}
	if req.AvatarURL != nil {
		value := strings.TrimSpace(*req.AvatarURL)
		req.AvatarURL = &value
	}

	user, err := h.users.UpdateProfile(c.Request.Context(), userID, req.DisplayName, req.AvatarURL)
	if err != nil {
		handleError(c, err)
		return
	}

	isAdmin, err := h.users.IsAdmin(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.OK(c, toUserResponse(user, isAdmin))
}
