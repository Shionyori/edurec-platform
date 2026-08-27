package repository

import (
	"errors"
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// RecommendationRepository 推荐结果缓存数据访问接口
type RecommendationRepository interface {
	// FindByUserID 按用户查询推荐缓存，未命中时返回 (nil, nil)
	FindByUserID(userID uint) (*model.Recommendation, error)
	// Replace 整行替换某用户的推荐缓存（每个用户只保留一份），返回保存后的记录
	Replace(userID uint, resourceIDs string, now int64) (*model.Recommendation, error)
}

type RecommendationRepo struct {
	db *gorm.DB
}

func NewRecommendationRepository(db *gorm.DB) *RecommendationRepo {
	return &RecommendationRepo{db: db}
}

func (r *RecommendationRepo) FindByUserID(userID uint) (*model.Recommendation, error) {
	rec := &model.Recommendation{}
	err := r.db.Where("user_id = ?", userID).First(rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询推荐缓存失败: %w", err)
	}
	return rec, nil
}

func (r *RecommendationRepo) Replace(userID uint, resourceIDs string, now int64) (*model.Recommendation, error) {
	// 先删除旧缓存再写入，保证 CreatedAt / UpdatedAt 均为本次生成时间
	if err := r.db.Where("user_id = ?", userID).Delete(&model.Recommendation{}).Error; err != nil {
		return nil, fmt.Errorf("清理旧推荐缓存失败: %w", err)
	}
	rec := &model.Recommendation{
		UserID:      userID,
		ResourceIDs: resourceIDs,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := r.db.Create(rec).Error; err != nil {
		return nil, fmt.Errorf("保存推荐缓存失败: %w", err)
	}
	return rec, nil
}
