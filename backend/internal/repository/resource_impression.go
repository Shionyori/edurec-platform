package repository

import (
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// ResourceImpressionRepository 推荐曝光数据访问接口
type ResourceImpressionRepository interface {
	CreateBatch(impressions []model.ResourceImpression) error
}

type ResourceImpressionRepo struct {
	db *gorm.DB
}

func NewResourceImpressionRepository(db *gorm.DB) *ResourceImpressionRepo {
	return &ResourceImpressionRepo{db: db}
}

// CreateBatch 一次性写入一批曝光记录（曝光量大，逐条写不合适）。
func (r *ResourceImpressionRepo) CreateBatch(impressions []model.ResourceImpression) error {
	if len(impressions) == 0 {
		return nil
	}
	if err := r.db.Create(&impressions).Error; err != nil {
		return fmt.Errorf("创建曝光记录失败: %w", err)
	}
	return nil
}
