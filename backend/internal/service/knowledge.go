package service

import (
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// KnowledgeService 知识点导航：列表、图谱与知识点下的资源。
type KnowledgeService struct {
	points repository.KnowledgeRepository
}

func NewKnowledgeService(points repository.KnowledgeRepository) *KnowledgeService {
	return &KnowledgeService{points: points}
}

// KnowledgeNode 图谱中的一个节点
type KnowledgeNode struct {
	ID             uint   `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Level          int    `json:"level"`
	DisciplineCode string `json:"discipline_code"`
	ResourceCount  int64  `json:"resource_count"`
	ConceptCount   int64  `json:"concept_count"`
}

// KnowledgeEdge 图谱中的一条先修边：From 学完才能学 To
type KnowledgeEdge struct {
	From uint `json:"from"`
	To   uint `json:"to"`
}

// KnowledgeConceptEdge 概念层的先修边（概念 ID）
type KnowledgeConceptEdge struct {
	From uint `json:"from"`
	To   uint `json:"to"`
}

// KnowledgeDetail 知识点详情：当前节点 + 整张图 + 该节点的资源 + 该学科的概念子图
type KnowledgeDetail struct {
	Point         KnowledgeNode            `json:"point"`
	Nodes         []KnowledgeNode          `json:"nodes"`
	Edges         []KnowledgeEdge          `json:"edges"`
	Resources     []model.Resource         `json:"resources"`
	ResourceTotal int64                    `json:"resource_total"`
	Concepts      []model.KnowledgeConcept `json:"concepts"`
	ConceptEdges  []KnowledgeConceptEdge   `json:"concept_edges"`
}

// List 返回全部知识点（含资源数），按层级与排序号排列
func (s *KnowledgeService) List() ([]KnowledgeNode, error) {
	points, err := s.points.ListPoints()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	counts, err := s.points.CountResourcesByPoint()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	conceptCounts, err := s.points.CountConceptsByPoint()
	if err != nil {
		return nil, apperror.Internal(err)
	}

	nodes := make([]KnowledgeNode, 0, len(points))
	for i := range points {
		nodes = append(nodes, toKnowledgeNode(&points[i], counts[points[i].ID], conceptCounts[points[i].Code]))
	}
	return nodes, nil
}

// Detail 返回知识点详情。code 不存在时返回 404 业务错误。
func (s *KnowledgeService) Detail(code string, resourceLimit int) (*KnowledgeDetail, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, apperror.BadRequest("知识点标识不能为空")
	}
	point, err := s.points.FindPointByCode(code)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if point == nil {
		return nil, apperror.NotFound("知识点不存在")
	}

	points, err := s.points.ListPoints()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	edges, err := s.points.ListEdges()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	counts, err := s.points.CountResourcesByPoint()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	conceptCounts, err := s.points.CountConceptsByPoint()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	resources, total, err := s.points.ListResourcesByPoint(point.ID, resourceLimit, 0)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	// 列表字段始终返回数组：nil 会被编码成 JSON null，前端无法按数组渲染。
	if resources == nil {
		resources = make([]model.Resource, 0)
	}

	// 概念层：该学科下的概念（EMBEDD-ER / MOOCCube），以及这些概念之间的先修边。
	// 上限 120 个，避免一次把上千概念塞给前端；层级升序便于按学习顺序展示。
	concepts, err := s.points.ListConceptsByPoint(point.Code, 120)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if concepts == nil {
		concepts = make([]model.KnowledgeConcept, 0)
	}
	conceptIDs := make([]uint, 0, len(concepts))
	for i := range concepts {
		conceptIDs = append(conceptIDs, concepts[i].ID)
	}
	conceptEdges, err := s.points.ListConceptEdgesAmong(conceptIDs)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	graphConceptEdges := make([]KnowledgeConceptEdge, 0, len(conceptEdges))
	for i := range conceptEdges {
		graphConceptEdges = append(graphConceptEdges, KnowledgeConceptEdge{
			From: conceptEdges[i].FromID,
			To:   conceptEdges[i].ToID,
		})
	}

	nodes := make([]KnowledgeNode, 0, len(points))
	for i := range points {
		nodes = append(nodes, toKnowledgeNode(&points[i], counts[points[i].ID], conceptCounts[points[i].Code]))
	}
	graphEdges := make([]KnowledgeEdge, 0, len(edges))
	for i := range edges {
		graphEdges = append(graphEdges, KnowledgeEdge{From: edges[i].FromID, To: edges[i].ToID})
	}

	return &KnowledgeDetail{
		Point:         toKnowledgeNode(point, counts[point.ID], conceptCounts[point.Code]),
		Nodes:         nodes,
		Edges:         graphEdges,
		Resources:     resources,
		ResourceTotal: total,
		Concepts:      concepts,
		ConceptEdges:  graphConceptEdges,
	}, nil
}

// Resources 分页返回某知识点下的资源，同时返回知识点本身便于前端展示标题
func (s *KnowledgeService) Resources(code string, page, pageSize int) ([]model.Resource, int64, *KnowledgeNode, error) {
	point, err := s.points.FindPointByCode(strings.TrimSpace(code))
	if err != nil {
		return nil, 0, nil, apperror.Internal(err)
	}
	if point == nil {
		return nil, 0, nil, apperror.NotFound("知识点不存在")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 12
	}
	if pageSize > 100 {
		pageSize = 100
	}

	items, total, err := s.points.ListResourcesByPoint(point.ID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, nil, apperror.Internal(err)
	}

	counts, err := s.points.CountResourcesByPoint()
	if err != nil {
		return nil, 0, nil, apperror.Internal(err)
	}
	conceptCounts, err := s.points.CountConceptsByPoint()
	if err != nil {
		return nil, 0, nil, apperror.Internal(err)
	}
	node := toKnowledgeNode(point, counts[point.ID], conceptCounts[point.Code])
	return items, total, &node, nil
}

func toKnowledgeNode(point *model.KnowledgePoint, count, conceptCount int64) KnowledgeNode {
	return KnowledgeNode{
		ID:             point.ID,
		Code:           point.Code,
		Name:           point.Name,
		Description:    point.Description,
		Level:          point.Level,
		DisciplineCode: point.DisciplineCode,
		ResourceCount:  count,
		ConceptCount:   conceptCount,
	}
}
