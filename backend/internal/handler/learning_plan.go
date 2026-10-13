package handler

import (
	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/middleware"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type LearningPlanHandler struct {
	plan *service.KnowledgePointService
}

func NewLearningPlanHandler(plan *service.KnowledgePointService) *LearningPlanHandler {
	return &LearningPlanHandler{plan: plan}
}

type knowledgePointSummary struct {
	ID          uint             `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Category    *categorySummary `json:"category"`
}

type learningPlanGroup struct {
	KnowledgePoint knowledgePointSummary `json:"knowledge_point"`
	Total          int64                 `json:"total"`
	Resources      []resourceListItem    `json:"resources"`
}

// Get 学习目标 → 知识点规划（GET /api/v1/learning-plan）
//
// 参数校验（空目标、超长目标）在 service 里，这里只做绑定与错误映射。
func (h *LearningPlanHandler) Get(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证或 token 无效")
		return
	}

	limit, ok := queryPositiveInt(c, "knowledge_points", service.DefaultKnowledgePointLimit)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	plan, err := h.plan.PlanForGoal(userID, c.Query("goal"), limit)
	if err != nil {
		handleError(c, err)
		return
	}

	groups := make([]learningPlanGroup, 0, len(plan.Groups))
	for i := range plan.Groups {
		g := &plan.Groups[i]

		items := make([]resourceListItem, 0, len(g.Resources))
		for j := range g.Resources {
			items = append(items, toResourceListItem(&g.Resources[j]))
		}

		var category *categorySummary
		if g.KnowledgePoint.Category.ID != 0 {
			category = &categorySummary{
				ID:   g.KnowledgePoint.Category.ID,
				Name: g.KnowledgePoint.Category.Name,
			}
		}

		groups = append(groups, learningPlanGroup{
			KnowledgePoint: knowledgePointSummary{
				ID:          g.KnowledgePoint.ID,
				Name:        g.KnowledgePoint.Name,
				Description: g.KnowledgePoint.Description,
				Category:    category,
			},
			Total:     g.Total,
			Resources: items,
		})
	}

	response.OK(c, gin.H{
		"goal":    plan.Goal,
		"matched": plan.Matched,
		"groups":  groups,
	})
}
