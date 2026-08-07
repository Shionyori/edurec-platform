package handler

import (
	"strconv"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/middleware"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type RatingHandler struct {
	ratings *service.RatingService
}

func NewRatingHandler(ratings *service.RatingService) *RatingHandler {
	return &RatingHandler{ratings: ratings}
}

type ratingUserSummary struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

type ratingListItem struct {
	ID        uint               `json:"id"`
	User      *ratingUserSummary `json:"user"`
	Score     uint8              `json:"score"`
	Comment   string             `json:"comment"`
	CreatedAt time.Time          `json:"created_at"`
}

func toRatingListItem(rating *model.Rating) ratingListItem {
	var user *ratingUserSummary
	if rating.User.ID != 0 {
		user = &ratingUserSummary{
			ID:          rating.User.ID,
			Username:    rating.User.Username,
			DisplayName: rating.User.DisplayName,
			AvatarURL:   rating.User.AvatarURL,
		}
	}

	return ratingListItem{
		ID:        rating.ID,
		User:      user,
		Score:     rating.Score,
		Comment:   rating.Comment,
		CreatedAt: rating.CreatedAt,
	}
}

func (h *RatingHandler) List(c *gin.Context) {
	resourceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || resourceID == 0 {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
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

	result, err := h.ratings.ListByResource(c.Request.Context(), uint(resourceID), page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]ratingListItem, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, toRatingListItem(&result.Items[i]))
	}

	response.OK(c, response.Page{
		List:     items,
		Total:    result.Total,
		Page:     page,
		PageSize: pageSize,
	})
}

type upsertRatingRequest struct {
	Score   int    `json:"score" binding:"required,min=1,max=5"`
	Comment string `json:"comment" binding:"omitempty"`
}

func (h *RatingHandler) Upsert(c *gin.Context) {
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

	var req upsertRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	rating, err := h.ratings.Upsert(c.Request.Context(), userID, uint(resourceID), uint8(req.Score), req.Comment)
	if err != nil {
		handleError(c, err)
		return
	}

	response.Created(c, gin.H{
		"id":         rating.ID,
		"score":      rating.Score,
		"comment":    rating.Comment,
		"created_at": rating.CreatedAt,
	})
}
