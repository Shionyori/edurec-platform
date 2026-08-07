package handler

import (
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	admins *service.AdminService
}

func NewAdminHandler(admins *service.AdminService) *AdminHandler {
	return &AdminHandler{admins: admins}
}

type adminUserListItem struct {
	ID          uint      `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	IsAdmin     bool      `json:"is_admin"`
	CreatedAt   time.Time `json:"created_at"`
}

type adminResourceListItem struct {
	ID        uint             `json:"id"`
	Title     string           `json:"title"`
	Type      string           `json:"type"`
	Category  *categorySummary `json:"category"`
	AvgRating float32          `json:"avg_rating"`
	ViewCount uint             `json:"view_count"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
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

	result, err := h.admins.ListUsers(c.Request.Context(), repository.AdminUserListQuery{
		Page:     page,
		PageSize: pageSize,
		Keyword:  c.Query("keyword"),
	})
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]adminUserListItem, 0, len(result.Items))
	for i := range result.Items {
		user := &result.Items[i]
		items = append(items, adminUserListItem{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			IsAdmin:     result.IsAdminByUserID[user.ID],
			CreatedAt:   user.CreatedAt,
		})
	}

	response.OK(c, response.Page{
		List:     items,
		Total:    result.Total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (h *AdminHandler) ListResources(c *gin.Context) {
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
	categoryID, ok := queryUint(c, "category_id", 0)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	result, err := h.admins.ListResources(c.Request.Context(), repository.AdminResourceListQuery{
		Page:       page,
		PageSize:   pageSize,
		Keyword:    c.Query("keyword"),
		Type:       c.Query("type"),
		CategoryID: categoryID,
	})
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]adminResourceListItem, 0, len(result.Items))
	for i := range result.Items {
		resource := &result.Items[i]
		var category *categorySummary
		if resource.Category.ID != 0 {
			category = &categorySummary{
				ID:   resource.Category.ID,
				Name: resource.Category.Name,
			}
		}
		items = append(items, adminResourceListItem{
			ID:        resource.ID,
			Title:     resource.Title,
			Type:      resource.Type,
			Category:  category,
			AvgRating: resource.AvgRating,
			ViewCount: resource.ViewCount,
			CreatedAt: resource.CreatedAt,
			UpdatedAt: resource.UpdatedAt,
		})
	}

	response.OK(c, response.Page{
		List:     items,
		Total:    result.Total,
		Page:     page,
		PageSize: pageSize,
	})
}
