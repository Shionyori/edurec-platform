package repository

import (
	"errors"
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// RecommendationUpsert 覆盖写某用户推荐缓存的入参。
// 用结构体而非散参，便于后续增加字段（如 scores/曝光）而不反复改签名。
type RecommendationUpsert struct {
	UserID      uint
	ResourceIDs string // JSON 数组字符串
	RunID       string // 产出本次结果的运行标识，空表示平台兜底生成
	Reasons     string // JSON 数组字符串，与 ResourceIDs 一一对应；可为空
	Now         int64  // Unix 秒
}

// RecommendationRepository 推荐结果缓存数据访问接口
type RecommendationRepository interface {
	// FindByUserID 按用户查询推荐缓存，未命中时返回 (nil, nil)
	FindByUserID(userID uint) (*model.Recommendation, error)
	// Replace 整行替换某用户的推荐缓存（每个用户只保留一份）
	Replace(input RecommendationUpsert) (*model.Recommendation, error)
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

func (r *RecommendationRepo) Replace(input RecommendationUpsert) (*model.Recommendation, error) {
	// 先删除旧缓存再写入，保证 CreatedAt / UpdatedAt 均为本次生成时间
	if err := r.db.Where("user_id = ?", input.UserID).Delete(&model.Recommendation{}).Error; err != nil {
		return nil, fmt.Errorf("清理旧推荐缓存失败: %w", err)
	}
	if input.Reasons == "" {
		input.Reasons = "[]"
	}
	rec := &model.Recommendation{
		UserID:      input.UserID,
		ResourceIDs: input.ResourceIDs,
		RunID:       input.RunID,
		Reasons:     input.Reasons,
		CreatedAt:   input.Now,
		UpdatedAt:   input.Now,
	}
	if err := r.db.Create(rec).Error; err != nil {
		return nil, fmt.Errorf("保存推荐缓存失败: %w", err)
	}
	return rec, nil
}
