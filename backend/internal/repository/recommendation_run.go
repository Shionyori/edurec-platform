package repository

import (
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// RecommendationRunRepository 推荐运行记录数据访问接口
type RecommendationRunRepository interface {
	Create(run *model.RecommendationRun) error
	// List 按导入时间倒序返回最近的运行记录
	List(limit int) ([]model.RecommendationRun, error)
}

type RecommendationRunRepo struct {
	db *gorm.DB
}

func NewRecommendationRunRepository(db *gorm.DB) *RecommendationRunRepo {
	return &RecommendationRunRepo{db: db}
}

func (r *RecommendationRunRepo) Create(run *model.RecommendationRun) error {
	if err := r.db.Create(run).Error; err != nil {
		return fmt.Errorf("创建推荐运行记录失败: %w", err)
	}
	return nil
}

func (r *RecommendationRunRepo) List(limit int) ([]model.RecommendationRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	runs := make([]model.RecommendationRun, 0, limit)
	if err := r.db.Order("id DESC").Limit(limit).Find(&runs).Error; err != nil {
		return nil, fmt.Errorf("查询推荐运行记录失败: %w", err)
	}
	return runs, nil
}
