package repository

import (
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// KnowledgeRepository 知识点与资源关联的数据访问接口
type KnowledgeRepository interface {
	ListPoints() ([]model.KnowledgePoint, error)
	FindPointByCode(code string) (*model.KnowledgePoint, error)
	FindPointByID(id uint) (*model.KnowledgePoint, error)
	ListEdges() ([]model.KnowledgePointEdge, error)
	CountResourcesByPoint() (map[uint]int64, error)
	ListResourcesByPoint(pointID uint, limit, offset int) ([]model.Resource, int64, error)
	UpsertPoint(point *model.KnowledgePoint) error
	UpdatePointLevel(id uint, level int) error

	// 学科层（主页面入口）
	UpsertDiscipline(discipline *model.Discipline) error
	ListDisciplines() ([]model.Discipline, error)
	FindDisciplineByCode(code string) (*model.Discipline, error)
	ListPointsByDiscipline(code string) ([]model.KnowledgePoint, error)
	CountPointsByDiscipline() (map[string]int64, error)
	CountResourcesByDiscipline() (map[string]int64, error)
	ListResourcesByDiscipline(code string, limit, offset int) ([]model.Resource, int64, error)
	CreateEdgeIfAbsent(edge *model.KnowledgePointEdge) error
	UpsertResourceLink(link *model.ResourceKnowledgePoint) error
	ListLinkedResourceIDs(pointID uint) ([]uint, error)

	// 概念层（EMBEDD-ER / MOOCCube + generate.py 生成）
	UpsertConcept(concept *model.KnowledgeConcept) error
	FindConceptByCode(code string) (*model.KnowledgeConcept, error)
	FindConceptByNamePoint(name, pointCode string) (*model.KnowledgeConcept, error)
	UpdateConceptLevel(id uint, level int) error
	UpsertConceptEdge(edge *model.KnowledgeConceptEdge) error
	ListConceptsByPoint(pointCode string, limit int) ([]model.KnowledgeConcept, error)
	ListConceptEdgesAmong(ids []uint) ([]model.KnowledgeConceptEdge, error)
	CountConceptsByPoint() (map[string]int64, error)
	// ResetKnowledgeGraph 清空知识图谱全部派生数据（学科、概念、先修边、资源关联）。
	// 资源与用户数据不受影响；用于图谱规则调整后整体重建。
	ResetKnowledgeGraph() error
}

type KnowledgeRepo struct {
	db *gorm.DB
}

func NewKnowledgeRepository(db *gorm.DB) *KnowledgeRepo {
	return &KnowledgeRepo{db: db}
}

func (r *KnowledgeRepo) ListPoints() ([]model.KnowledgePoint, error) {
	var points []model.KnowledgePoint
	if err := r.db.Order("level ASC, sort_order ASC, id ASC").Find(&points).Error; err != nil {
		return nil, fmt.Errorf("查询知识点失败: %w", err)
	}
	return points, nil
}

func (r *KnowledgeRepo) FindPointByCode(code string) (*model.KnowledgePoint, error) {
	var point model.KnowledgePoint
	err := r.db.Where("code = ?", code).First(&point).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询知识点失败: %w", err)
	}
	return &point, nil
}

func (r *KnowledgeRepo) FindPointByID(id uint) (*model.KnowledgePoint, error) {
	var point model.KnowledgePoint
	err := r.db.First(&point, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询知识点失败: %w", err)
	}
	return &point, nil
}

func (r *KnowledgeRepo) ListEdges() ([]model.KnowledgePointEdge, error) {
	var edges []model.KnowledgePointEdge
	if err := r.db.Order("from_id ASC, to_id ASC").Find(&edges).Error; err != nil {
		return nil, fmt.Errorf("查询知识点关系失败: %w", err)
	}
	return edges, nil
}

// CountResourcesByPoint 统计每个知识点下的资源数（一次聚合查询，避免 N+1）
func (r *KnowledgeRepo) CountResourcesByPoint() (map[uint]int64, error) {
	type row struct {
		PointID uint
		Total   int64
	}
	var rows []row
	err := r.db.Model(&model.ResourceKnowledgePoint{}).
		Select("point_id, COUNT(*) AS total").
		Group("point_id").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("统计知识点资源数失败: %w", err)
	}
	out := make(map[uint]int64, len(rows))
	for _, item := range rows {
		out[item.PointID] = item.Total
	}
	return out, nil
}

// ListResourcesByPoint 按关联强度与评分列出某知识点下的资源，返回当页与总数
func (r *KnowledgeRepo) ListResourcesByPoint(pointID uint, limit, offset int) ([]model.Resource, int64, error) {
	if limit <= 0 {
		limit = 12
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	base := r.db.Model(&model.Resource{}).
		Joins("JOIN resource_knowledge_points rkp ON rkp.resource_id = resources.id AND rkp.deleted_at IS NULL").
		Where("rkp.point_id = ?", pointID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计知识点资源失败: %w", err)
	}

	var items []model.Resource
	err := base.
		Select("resources.*, rkp.weight AS rkp_weight").
		Preload("Category").
		Order("rkp.weight DESC, resources.avg_rating DESC, resources.view_count DESC, resources.id DESC").
		Limit(limit).Offset(offset).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询知识点资源失败: %w", err)
	}
	return items, total, nil
}

// UpsertPoint 按 code 幂等写入知识点
func (r *KnowledgeRepo) UpsertPoint(point *model.KnowledgePoint) error {
	var existing model.KnowledgePoint
	err := r.db.Where("code = ?", point.Code).First(&existing).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		if err := r.db.Create(point).Error; err != nil {
			return fmt.Errorf("创建知识点失败: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("查询知识点失败: %w", err)
	}

	point.ID = existing.ID
	if err := r.db.Model(&existing).Updates(map[string]any{
		"name":            point.Name,
		"description":     point.Description,
		"sort_order":      point.SortOrder,
		"discipline_code": point.DisciplineCode,
	}).Error; err != nil {
		return fmt.Errorf("更新知识点失败: %w", err)
	}
	return nil
}

// UpdatePointLevel 写入由先修关系推导出的层级
func (r *KnowledgeRepo) UpdatePointLevel(id uint, level int) error {
	if err := r.db.Model(&model.KnowledgePoint{}).Where("id = ?", id).
		Update("level", level).Error; err != nil {
		return fmt.Errorf("更新知识点层级失败: %w", err)
	}
	return nil
}

// CreateEdgeIfAbsent 先修关系按 (from,to) 幂等创建
func (r *KnowledgeRepo) CreateEdgeIfAbsent(edge *model.KnowledgePointEdge) error {
	var count int64
	if err := r.db.Model(&model.KnowledgePointEdge{}).
		Where("from_id = ? AND to_id = ?", edge.FromID, edge.ToID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("查询先修关系失败: %w", err)
	}
	if count > 0 {
		return nil
	}
	if err := r.db.Create(edge).Error; err != nil {
		return fmt.Errorf("创建先修关系失败: %w", err)
	}
	return nil
}

// UpsertResourceLink 资源与知识点关联按 (resource,point) 幂等写入，重复执行只更新权重
func (r *KnowledgeRepo) UpsertResourceLink(link *model.ResourceKnowledgePoint) error {
	var existing model.ResourceKnowledgePoint
	err := r.db.Where("resource_id = ? AND point_id = ?", link.ResourceID, link.PointID).
		First(&existing).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		if err := r.db.Create(link).Error; err != nil {
			return fmt.Errorf("创建资源知识点关联失败: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("查询资源知识点关联失败: %w", err)
	}
	if err := r.db.Model(&existing).Update("weight", link.Weight).Error; err != nil {
		return fmt.Errorf("更新资源知识点关联失败: %w", err)
	}
	return nil
}

func (r *KnowledgeRepo) ListLinkedResourceIDs(pointID uint) ([]uint, error) {
	var ids []uint
	if err := r.db.Model(&model.ResourceKnowledgePoint{}).
		Where("point_id = ?", pointID).
		Pluck("resource_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("查询知识点关联资源失败: %w", err)
	}
	return ids, nil
}

// ---------- 概念层（EMBEDD-ER / MOOCCube） ----------

// UpsertConcept 按 code 幂等写入概念；已存在时刷新名称、领域、学科归属与释义
func (r *KnowledgeRepo) UpsertConcept(concept *model.KnowledgeConcept) error {
	var existing model.KnowledgeConcept
	err := r.db.Where("code = ?", concept.Code).First(&existing).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		if err := r.db.Create(concept).Error; err != nil {
			return fmt.Errorf("创建概念失败: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("查询概念失败: %w", err)
	}

	concept.ID = existing.ID
	if err := r.db.Model(&existing).Updates(map[string]any{
		"name":        concept.Name,
		"field":       concept.Field,
		"point_code":  concept.PointCode,
		"explanation": concept.Explanation,
		"source":      concept.Source,
		"page_rank":   concept.PageRank,
	}).Error; err != nil {
		return fmt.Errorf("更新概念失败: %w", err)
	}
	return nil
}

// FindConceptByNamePoint 按「学科 + 概念名」查重，生成概念时避免与 MOOCCube 已有概念重复
func (r *KnowledgeRepo) FindConceptByNamePoint(name, pointCode string) (*model.KnowledgeConcept, error) {
	var concept model.KnowledgeConcept
	err := r.db.Where("name = ? AND point_code = ?", name, pointCode).First(&concept).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询概念失败: %w", err)
	}
	return &concept, nil
}

func (r *KnowledgeRepo) FindConceptByCode(code string) (*model.KnowledgeConcept, error) {
	var concept model.KnowledgeConcept
	err := r.db.Where("code = ?", code).First(&concept).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询概念失败: %w", err)
	}
	return &concept, nil
}

func (r *KnowledgeRepo) UpdateConceptLevel(id uint, level int) error {
	if err := r.db.Model(&model.KnowledgeConcept{}).Where("id = ?", id).
		Update("level", level).Error; err != nil {
		return fmt.Errorf("更新概念层级失败: %w", err)
	}
	return nil
}

// UpsertConceptEdge 概念先修边按 (from,to) 幂等创建
func (r *KnowledgeRepo) UpsertConceptEdge(edge *model.KnowledgeConceptEdge) error {
	var count int64
	if err := r.db.Model(&model.KnowledgeConceptEdge{}).
		Where("from_id = ? AND to_id = ?", edge.FromID, edge.ToID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("查询概念先修关系失败: %w", err)
	}
	if count > 0 {
		return nil
	}
	if err := r.db.Create(edge).Error; err != nil {
		return fmt.Errorf("创建概念先修关系失败: %w", err)
	}
	return nil
}

// ListConceptsByPoint 按学科取概念（层级升序，名称稳定排序），limit<=0 表示不限制
func (r *KnowledgeRepo) ListConceptsByPoint(pointCode string, limit int) ([]model.KnowledgeConcept, error) {
	var concepts []model.KnowledgeConcept
	query := r.db.Where("point_code = ?", pointCode).Order("level ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&concepts).Error; err != nil {
		return nil, fmt.Errorf("查询学科概念失败: %w", err)
	}
	return concepts, nil
}

// ListConceptEdgesAmong 取给定概念集合内部的先修边
func (r *KnowledgeRepo) ListConceptEdgesAmong(ids []uint) ([]model.KnowledgeConceptEdge, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var edges []model.KnowledgeConceptEdge
	if err := r.db.Where("from_id IN ? AND to_id IN ?", ids, ids).
		Find(&edges).Error; err != nil {
		return nil, fmt.Errorf("查询概念先修边失败: %w", err)
	}
	return edges, nil
}

func (r *KnowledgeRepo) CountConceptsByPoint() (map[string]int64, error) {
	type row struct {
		PointCode string
		Total     int64
	}
	var rows []row
	if err := r.db.Model(&model.KnowledgeConcept{}).
		Select("point_code, COUNT(*) AS total").
		Where("point_code <> ''").
		Group("point_code").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("统计学科概念数失败: %w", err)
	}
	out := make(map[string]int64, len(rows))
	for _, item := range rows {
		out[item.PointCode] = item.Total
	}
	return out, nil
}

// ResetKnowledgeGraph 硬删除知识图谱派生数据，供重复构建使用。
//
// 只删「由导入推导出来」的表：资源关联、两层先修边、学科、概念。
// 资源表、用户表、行为/评分等业务数据一概不动——重建图谱不会影响推荐主链路。
// 删除顺序从子表到父表，避免外键/残留引用。
func (r *KnowledgeRepo) ResetKnowledgeGraph() error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		targets := []any{
			&model.ResourceKnowledgePoint{},
			&model.KnowledgeConceptEdge{},
			&model.KnowledgePointEdge{},
			&model.KnowledgeConcept{},
			&model.KnowledgePoint{},
		}
		for _, target := range targets {
			if err := tx.Unscoped().Where("1 = 1").Delete(target).Error; err != nil {
				return fmt.Errorf("清空知识图谱数据失败: %w", err)
			}
		}
		return nil
	})
}

// ---------- 学科层 ----------

// UpsertDiscipline 按 code 幂等写入学科
func (r *KnowledgeRepo) UpsertDiscipline(discipline *model.Discipline) error {
	var existing model.Discipline
	err := r.db.Where("code = ?", discipline.Code).First(&existing).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		if err := r.db.Create(discipline).Error; err != nil {
			return fmt.Errorf("创建学科失败: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("查询学科失败: %w", err)
	}

	if err := r.db.Model(&existing).Updates(map[string]any{
		"name":        discipline.Name,
		"description": discipline.Description,
		"keywords":    discipline.Keywords,
		"sort_order":  discipline.SortOrder,
	}).Error; err != nil {
		return fmt.Errorf("更新学科失败: %w", err)
	}
	return nil
}

func (r *KnowledgeRepo) ListDisciplines() ([]model.Discipline, error) {
	var items []model.Discipline
	if err := r.db.Order("sort_order ASC, id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询学科失败: %w", err)
	}
	return items, nil
}

func (r *KnowledgeRepo) FindDisciplineByCode(code string) (*model.Discipline, error) {
	var discipline model.Discipline
	err := r.db.Where("code = ?", code).First(&discipline).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询学科失败: %w", err)
	}
	return &discipline, nil
}

func (r *KnowledgeRepo) ListPointsByDiscipline(code string) ([]model.KnowledgePoint, error) {
	var points []model.KnowledgePoint
	if err := r.db.Where("discipline_code = ?", code).
		Order("level ASC, sort_order ASC, id ASC").Find(&points).Error; err != nil {
		return nil, fmt.Errorf("查询学科知识点失败: %w", err)
	}
	return points, nil
}

func (r *KnowledgeRepo) CountPointsByDiscipline() (map[string]int64, error) {
	type row struct {
		DisciplineCode string
		Total          int64
	}
	var rows []row
	if err := r.db.Model(&model.KnowledgePoint{}).
		Select("discipline_code, COUNT(*) AS total").
		Where("discipline_code <> ''").
		Group("discipline_code").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("统计学科知识点数失败: %w", err)
	}
	out := make(map[string]int64, len(rows))
	for _, item := range rows {
		out[item.DisciplineCode] = item.Total
	}
	return out, nil
}

func (r *KnowledgeRepo) CountResourcesByDiscipline() (map[string]int64, error) {
	type row struct {
		DisciplineCode string
		Total          int64
	}
	var rows []row
	err := r.db.Raw(`
		SELECT p.discipline_code AS discipline_code, COUNT(DISTINCT rkp.resource_id) AS total
		FROM resource_knowledge_points rkp
		JOIN knowledge_points p ON p.id = rkp.point_id AND p.deleted_at IS NULL
		JOIN resources r ON r.id = rkp.resource_id AND r.deleted_at IS NULL
		WHERE rkp.deleted_at IS NULL AND p.discipline_code <> ''
		GROUP BY p.discipline_code
	`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("统计学科资源数失败: %w", err)
	}
	out := make(map[string]int64, len(rows))
	for _, item := range rows {
		out[item.DisciplineCode] = item.Total
	}
	return out, nil
}

func (r *KnowledgeRepo) ListResourcesByDiscipline(code string, limit, offset int) ([]model.Resource, int64, error) {
	var total int64
	if err := r.db.Raw(`
		SELECT COUNT(DISTINCT r.id)
		FROM resources r
		JOIN resource_knowledge_points rkp ON rkp.resource_id = r.id AND rkp.deleted_at IS NULL
		JOIN knowledge_points p ON p.id = rkp.point_id AND p.deleted_at IS NULL
		WHERE r.deleted_at IS NULL AND p.discipline_code = ?
	`, code).Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计学科资源失败: %w", err)
	}

	// 初始化为空切片：Raw().Scan 在无结果时会保留 nil，JSON 序列化成 null，
	// 前端按数组访问会直接崩溃。
	items := make([]model.Resource, 0)
	if err := r.db.Raw(`
		SELECT DISTINCT r.*
		FROM resources r
		JOIN resource_knowledge_points rkp ON rkp.resource_id = r.id AND rkp.deleted_at IS NULL
		JOIN knowledge_points p ON p.id = rkp.point_id AND p.deleted_at IS NULL
		WHERE r.deleted_at IS NULL AND p.discipline_code = ?
		ORDER BY r.avg_rating DESC, r.id DESC
		LIMIT ? OFFSET ?
	`, code, limit, offset).Scan(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("查询学科资源失败: %w", err)
	}
	return items, total, nil
}
