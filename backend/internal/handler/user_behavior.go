package handler

import (
	"strconv"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/middleware"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
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

// Favorite 收藏资源（POST /resources/:id/favorite）
func (h *UserBehaviorHandler) Favorite(c *gin.Context) {
	h.setFavorite(c, true)
}

// Unfavorite 取消收藏（DELETE /resources/:id/favorite）
func (h *UserBehaviorHandler) Unfavorite(c *gin.Context) {
	h.setFavorite(c, false)
}

func (h *UserBehaviorHandler) setFavorite(c *gin.Context, favorite bool) {
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

	if err := h.behaviors.SetFavorite(c.Request.Context(), userID, uint(resourceID), favorite); err != nil {
		handleError(c, err)
		return
	}

	response.OK(c, gin.H{"favorited": favorite})
}

type behaviorResourceSummary struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"`
	CoverURL string `json:"cover_url"`
	Type     string `json:"type"`
}

type behaviorListItem struct {
	ID        uint                     `json:"id"`
	Resource  *behaviorResourceSummary `json:"resource"`
	Action    string                   `json:"action"`
	CreatedAt time.Time                `json:"created_at"`
}

func toBehaviorListItem(behavior *model.UserBehavior) behaviorListItem {
	var resource *behaviorResourceSummary
	if behavior.Resource.ID != 0 {
		resource = &behaviorResourceSummary{
			ID:       behavior.Resource.ID,
			Title:    behavior.Resource.Title,
			CoverURL: behavior.Resource.CoverURL,
			Type:     behavior.Resource.Type,
		}
	}

	return behaviorListItem{
		ID:        behavior.ID,
		Resource:  resource,
		Action:    behavior.Action,
		CreatedAt: behavior.CreatedAt,
	}
}

func (h *UserBehaviorHandler) List(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
		return
	}

	page, ok := queryPositiveInt(c, "page", 1)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	pageSize, ok := queryPositiveInt(c, "page_size", 20)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	result, err := h.behaviors.List(c.Request.Context(), repository.UserBehaviorListQuery{
		UserID:   userID,
		Action:   c.Query("action"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]behaviorListItem, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, toBehaviorListItem(&result.Items[i]))
	}

	response.OK(c, response.Page{
		List:     items,
		Total:    result.Total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Favorites 我的收藏（GET /users/me/favorites）：分页返回收藏的资源
func (h *UserBehaviorHandler) Favorites(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
		return
	}

	page, ok := queryPositiveInt(c, "page", 1)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	pageSize, ok := queryPositiveInt(c, "page_size", 12)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 || pageSize > 100 {
		pageSize = 12
	}

	resources, total, err := h.behaviors.ListFavorites(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]resourceListItem, 0, len(resources))
	for i := range resources {
		items = append(items, toResourceListItem(&resources[i]))
	}

	response.OK(c, response.Page{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
