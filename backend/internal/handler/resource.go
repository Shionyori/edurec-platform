package handler

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type ResourceHandler struct {
	resources *service.ResourceService
}

func NewResourceHandler(resources *service.ResourceService) *ResourceHandler {
	return &ResourceHandler{resources: resources}
}

type categorySummary struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type resourceListItem struct {
	ID          uint             `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	CoverURL    string           `json:"cover_url"`
	Type        string           `json:"type"`
	Category    *categorySummary `json:"category"`
	Tags        []string         `json:"tags"`
	Author      string           `json:"author"`
	AvgRating   float32          `json:"avg_rating"`
	ViewCount   uint             `json:"view_count"`
	CreatedAt   time.Time        `json:"created_at"`
}

type resourceDetail struct {
	ID          uint             `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	CoverURL    string           `json:"cover_url"`
	Type        string           `json:"type"`
	Category    *categorySummary `json:"category"`
	Tags        []string         `json:"tags"`
	Metadata    map[string]any   `json:"metadata"`
	Author      string           `json:"author"`
	SourceURL   string           `json:"source_url"`
	AvgRating   float32          `json:"avg_rating"`
	ViewCount   uint             `json:"view_count"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

func toResourceListItem(resource *model.Resource) resourceListItem {
	var category *categorySummary
	if resource.Category.ID != 0 {
		category = &categorySummary{
			ID:   resource.Category.ID,
			Name: resource.Category.Name,
		}
	}

	return resourceListItem{
		ID:          resource.ID,
		Title:       resource.Title,
		Description: resource.Description,
		CoverURL:    resource.CoverURL,
		Type:        resource.Type,
		Category:    category,
		Tags:        parseTags(resource.Tags),
		Author:      resource.Author,
		AvgRating:   resource.AvgRating,
		ViewCount:   resource.ViewCount,
		CreatedAt:   resource.CreatedAt,
	}
}

func toResourceDetail(resource *model.Resource) resourceDetail {
	var category *categorySummary
	if resource.Category.ID != 0 {
		category = &categorySummary{
			ID:   resource.Category.ID,
			Name: resource.Category.Name,
		}
	}

	return resourceDetail{
		ID:          resource.ID,
		Title:       resource.Title,
		Description: resource.Description,
		CoverURL:    resource.CoverURL,
		Type:        resource.Type,
		Category:    category,
		Tags:        parseTags(resource.Tags),
		Metadata:    parseMetadata(resource.Metadata),
		Author:      resource.Author,
		SourceURL:   resource.SourceURL,
		AvgRating:   resource.AvgRating,
		ViewCount:   resource.ViewCount,
		CreatedAt:   resource.CreatedAt,
		UpdatedAt:   resource.UpdatedAt,
	}
}

func (h *ResourceHandler) List(c *gin.Context) {
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

	query := repository.ResourceListQuery{
		Page:       page,
		PageSize:   pageSize,
		Keyword:    c.Query("keyword"),
		CategoryID: categoryID,
		Type:       c.Query("type"),
		Tags:       parseCommaList(c.Query("tags")),
		Sort:       c.Query("sort"),
	}

	result, err := h.resources.List(c.Request.Context(), query)
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]resourceListItem, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, toResourceListItem(&result.Items[i]))
	}

	response.OK(c, response.Page{
		List:     items,
		Total:    result.Total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (h *ResourceHandler) Detail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	resource, err := h.resources.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		handleError(c, err)
		return
	}

	response.OK(c, toResourceDetail(resource))
}

func parseTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}

	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return []string{}
	}
	if tags == nil {
		return []string{}
	}
	return tags
}

func parseMetadata(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}

	var metadata map[string]any
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return map[string]any{}
	}
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func parseCommaList(raw string) []string {
	if raw == "" {
		return nil
	}
	items := strings.Split(raw, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func queryPositiveInt(c *gin.Context, key string, defaultValue int) (int, bool) {
	raw := c.Query(key)
	if raw == "" {
		return defaultValue, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}

func queryUint(c *gin.Context, key string, defaultValue uint) (uint, bool) {
	raw := c.Query(key)
	if raw == "" {
		return defaultValue, true
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(value), true
}
