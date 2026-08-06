package handler

import (
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	categories *service.CategoryService
}

func NewCategoryHandler(categories *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categories: categories}
}

type categoryResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func toCategoryResponse(category *model.Category) categoryResponse {
	return categoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Description: category.Description,
		CreatedAt:   category.CreatedAt,
	}
}

func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.categories.List(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]categoryResponse, 0, len(categories))
	for i := range categories {
		items = append(items, toCategoryResponse(&categories[i]))
	}
	response.OK(c, items)
}

type createCategoryRequest struct {
	Name        string `json:"name" binding:"required,max=64"`
	Description string `json:"description" binding:"omitempty,max=256"`
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	category, err := h.categories.Create(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, toCategoryResponse(category))
}
