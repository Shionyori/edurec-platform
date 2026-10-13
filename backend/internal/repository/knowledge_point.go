package repository

import (
	"errors"
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// KnowledgePointRepository 知识点数据访问接口
type KnowledgePointRepository interface {
	ListAll() ([]model.KnowledgePoint, error)
	UpsertByName(points []model.KnowledgePoint) (created int, updated int, err error)
}

type KnowledgePointRepo struct {
	db *gorm.DB
}

func NewKnowledgePointRepository(db *gorm.DB) *KnowledgePointRepo {
	return &KnowledgePointRepo{db: db}
}

// ListAll 载入全部知识点（含分类）。知识点量级很小（默认上限 60 条），
// 而目标匹配需要拿全量候选，所以不分页。
func (r *KnowledgePointRepo) ListAll() ([]model.KnowledgePoint, error) {
	points := make([]model.KnowledgePoint, 0)
	if err := r.db.Preload("Category").Order("sort_order ASC, id ASC").Find(&points).Error; err != nil {
		return nil, fmt.Errorf("查询知识点失败: %w", err)
	}
	return points, nil
}

// UpsertByName 按 name 幂等写入：已存在则更新展示字段（不新建行，避免重复导入
// 把知识点刷成一堆同义行），不存在则新建。整体在一个事务里，中途失败不留半批。
func (r *KnowledgePointRepo) UpsertByName(points []model.KnowledgePoint) (int, int, error) {
	created, updated := 0, 0
	err := r.db.Transaction(func(tx *gorm.DB) error {
		for i := range points {
			var existing model.KnowledgePoint
			err := tx.Where("name = ?", points[i].Name).First(&existing).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				if err := tx.Create(&points[i]).Error; err != nil {
					return fmt.Errorf("创建知识点 %q 失败: %w", points[i].Name, err)
				}
				created++
			case err != nil:
				return fmt.Errorf("查询知识点 %q 失败: %w", points[i].Name, err)
			default:
				existing.Description = points[i].Description
				existing.Aliases = points[i].Aliases
				existing.CategoryID = points[i].CategoryID
				existing.Source = points[i].Source
				existing.SortOrder = points[i].SortOrder
				if err := tx.Save(&existing).Error; err != nil {
					return fmt.Errorf("更新知识点 %q 失败: %w", points[i].Name, err)
				}
				updated++
			}
		}
		return nil
	})
	return created, updated, err
}
