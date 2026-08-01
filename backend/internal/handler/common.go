package handler

import (
	"errors"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/gin-gonic/gin"
)

type userResponse struct {
	ID          uint      `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	IsAdmin     bool      `json:"is_admin"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toUserResponse(user *model.User, isAdmin bool) userResponse {
	return userResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		IsAdmin:     isAdmin,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

func handleError(c *gin.Context, err error) {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		response.Error(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
		return
	}
	response.Error(c, 500, apperror.CodeInternal, "服务器内部错误")
}
