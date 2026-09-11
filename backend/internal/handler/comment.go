package handler

import (
	"strconv"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	resources *service.ResourceService
	comments  *service.CommentService
}

func NewCommentHandler(resources *service.ResourceService, comments *service.CommentService) *CommentHandler {
	return &CommentHandler{resources: resources, comments: comments}
}

type commentItem struct {
	ID          uint   `json:"id"`
	AuthorName  string `json:"author_name"`
	Content     string `json:"content"`
	LikeCount   uint   `json:"like_count"`
	Floor       int    `json:"floor"`
	PublishedAt int64  `json:"published_at"`
}

func toCommentItem(comment *model.ResourceComment) commentItem {
	return commentItem{
		ID:          comment.ID,
		AuthorName:  comment.AuthorName,
		Content:     comment.Content,
		LikeCount:   comment.LikeCount,
		Floor:       comment.Floor,
		PublishedAt: comment.PublishedAt,
	}
}

// List 返回资源下的 B 站评论列表（首次访问可能触发实时爬取）
func (h *CommentHandler) List(c *gin.Context) {
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

	comments, err := h.comments.ListOrFetch(c.Request.Context(), resource)
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]commentItem, 0, len(comments))
	for i := range comments {
		items = append(items, toCommentItem(&comments[i]))
	}

	response.OK(c, gin.H{"list": items})
}
