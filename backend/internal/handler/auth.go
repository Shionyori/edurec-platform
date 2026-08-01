package handler

import (
	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth  *service.AuthService
	users *service.UserService
}

func NewAuthHandler(auth *service.AuthService, users *service.UserService) *AuthHandler {
	return &AuthHandler{auth: auth, users: users}
}

type registerRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=64"`
	Email       string `json:"email" binding:"required,email,max=128"`
	Password    string `json:"password" binding:"required,min=6,max=128"`
	DisplayName string `json:"display_name" binding:"omitempty,max=128"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	user, err := h.auth.Register(c.Request.Context(), service.RegisterInput{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, gin.H{"user": toUserResponse(user, false)})
}

type loginRequest struct {
	Account  string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	user, pair, err := h.auth.Login(c.Request.Context(), service.LoginInput{
		Account:  req.Account,
		Password: req.Password,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	isAdmin, err := h.users.IsAdmin(c.Request.Context(), user.ID)
	if err != nil {
		handleError(c, err)
		return
	}

	response.OK(c, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.AccessExpiresIn,
		"user":          toUserResponse(user, isAdmin),
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	pair, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		handleError(c, err)
		return
	}

	response.OK(c, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.AccessExpiresIn,
	})
}
