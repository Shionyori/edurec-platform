package handler

import (
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// GenerationHandler 知识点概念生成（调用 engine 的 kg_server.py）
type GenerationHandler struct {
	generator *service.ConceptGenerator
	knowledge *service.KnowledgeService
}

func NewGenerationHandler(generator *service.ConceptGenerator, knowledge *service.KnowledgeService) *GenerationHandler {
	return &GenerationHandler{generator: generator, knowledge: knowledge}
}

// Start POST /knowledge-points/:code/generate
func (h *GenerationHandler) Start(c *gin.Context) {
	ctx := c.Request.Context()
	if rid := strings.TrimSpace(c.GetHeader("X-Request-ID")); rid != "" {
		ctx = service.WithRequestID(ctx, rid)
	}
	status, err := h.generator.Start(ctx, c.Param("code"))
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, status)
}

// Status GET /knowledge-points/:code/generate
func (h *GenerationHandler) Status(c *gin.Context) {
	status, err := h.generator.Status(c.Param("code"))
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, status)
}

type keywordGenerateRequest struct {
	Keyword string `json:"keyword" binding:"required"`
}

// StartByKeyword POST /knowledge-points/generate
//
// 搜索栏实时触发入口：把关键词匹配到最近的知识点（名称互相包含，取最长匹配），
// 再异步生成概念。匹配不到时返回 404，由前端静默忽略。
func (h *GenerationHandler) StartByKeyword(c *gin.Context) {
	var req keywordGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	if keyword == "" {
		response.Error(c, 400, apperror.CodeBadRequest, "关键词不能为空")
		return
	}

	points, err := h.knowledge.List()
	if err != nil {
		handleError(c, err)
		return
	}

	bestCode := ""
	bestName := ""
	for _, point := range points {
		name := strings.ToLower(point.Name)
		if name == "" {
			continue
		}
		if point.Code != keyword && !strings.Contains(keyword, name) && !strings.Contains(name, keyword) {
			continue
		}
		if len(point.Name) > len(bestName) {
			bestCode = point.Code
			bestName = point.Name
		}
	}
	if bestCode == "" {
		response.Error(c, 404, apperror.CodeNotFound, "没有匹配的知识点")
		return
	}

	ctx := c.Request.Context()
	if rid := strings.TrimSpace(c.GetHeader("X-Request-ID")); rid != "" {
		ctx = service.WithRequestID(ctx, rid)
	}
	status, err := h.generator.Start(ctx, bestCode)
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, gin.H{
		"point_code": bestCode,
		"point_name": bestName,
		"status":     status,
	})
}
