package service

import (
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// DisciplineNode 学科入口（主页面卡片）
type DisciplineNode struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	PointCount    int64  `json:"point_count"`
	ResourceCount int64  `json:"resource_count"`
}

// DisciplineDetail 学科详情：学科下的知识点 + 资源
type DisciplineDetail struct {
	Discipline    DisciplineNode   `json:"discipline"`
	Points        []KnowledgeNode  `json:"points"`
	Resources     []model.Resource `json:"resources"`
	ResourceTotal int64            `json:"resource_total"`
}

// DisciplineService 学科导航：主页面列表与学科页。
type DisciplineService struct {
	repo repository.KnowledgeRepository
}

func NewDisciplineService(repo repository.KnowledgeRepository) *DisciplineService {
	return &DisciplineService{repo: repo}
}

func (s *DisciplineService) List() ([]DisciplineNode, error) {
	disciplines, err := s.repo.ListDisciplines()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	pointCounts, err := s.repo.CountPointsByDiscipline()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	resourceCounts, err := s.repo.CountResourcesByDiscipline()
	if err != nil {
		return nil, apperror.Internal(err)
	}

	nodes := make([]DisciplineNode, 0, len(disciplines))
	for i := range disciplines {
		nodes = append(nodes, DisciplineNode{
			Code:          disciplines[i].Code,
			Name:          disciplines[i].Name,
			Description:   disciplines[i].Description,
			PointCount:    pointCounts[disciplines[i].Code],
			ResourceCount: resourceCounts[disciplines[i].Code],
		})
	}
	return nodes, nil
}

func (s *DisciplineService) Detail(code string, resourceLimit int) (*DisciplineDetail, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, apperror.BadRequest("学科标识不能为空")
	}
	discipline, err := s.repo.FindDisciplineByCode(code)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if discipline == nil {
		return nil, apperror.NotFound("学科不存在")
	}

	points, err := s.repo.ListPointsByDiscipline(code)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	resourceCounts, err := s.repo.CountResourcesByPoint()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	conceptCounts, err := s.repo.CountConceptsByPoint()
	if err != nil {
		return nil, apperror.Internal(err)
	}

	nodes := make([]KnowledgeNode, 0, len(points))
	for i := range points {
		nodes = append(nodes, toKnowledgeNode(&points[i], resourceCounts[points[i].ID], conceptCounts[points[i].Code]))
	}

	limit := resourceLimit
	if limit <= 0 {
		limit = 12
	}
	if limit > 100 {
		limit = 100
	}
	resources, total, err := s.repo.ListResourcesByDiscipline(code, limit, 0)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	// 列表字段始终返回数组：nil 会被编码成 JSON null，前端无法按数组渲染。
	if resources == nil {
		resources = make([]model.Resource, 0)
	}

	return &DisciplineDetail{
		Discipline: DisciplineNode{
			Code:          discipline.Code,
			Name:          discipline.Name,
			Description:   discipline.Description,
			PointCount:    int64(len(points)),
			ResourceCount: total,
		},
		Points:        nodes,
		Resources:     resources,
		ResourceTotal: total,
	}, nil
}
