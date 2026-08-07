package repository

import (
	"errors"
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// RatingRepository 评分评论数据访问接口
type RatingRepository interface {
	FindByUserAndResource(userID uint, resourceID uint) (*model.Rating, error)
	Create(rating *model.Rating) error
	Update(rating *model.Rating) error
	ListByResource(query RatingListQuery) (*RatingListResult, error)
	AverageScore(resourceID uint) (float64, error)
}

// RatingListQuery 评分列表查询条件
type RatingListQuery struct {
	ResourceID uint
	Page       int
	PageSize   int
}

// RatingListResult 评分列表分页结果
type RatingListResult struct {
	Items []model.Rating
	Total int64
}

type RatingRepo struct {
	db *gorm.DB
}

func NewRatingRepository(db *gorm.DB) *RatingRepo {
	return &RatingRepo{db: db}
}

func (r *RatingRepo) FindByUserAndResource(userID uint, resourceID uint) (*model.Rating, error) {
	rating := &model.Rating{}
	err := r.db.Where("user_id = ? AND resource_id = ?", userID, resourceID).First(rating).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户评分失败: %w", err)
	}
	return rating, nil
}

func (r *RatingRepo) Create(rating *model.Rating) error {
	if err := r.db.Create(rating).Error; err != nil {
		if isDuplicateKey(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("创建评分失败: %w", err)
	}
	return nil
}

func (r *RatingRepo) Update(rating *model.Rating) error {
	if err := r.db.Save(rating).Error; err != nil {
		return fmt.Errorf("更新评分失败: %w", err)
	}
	return nil
}

func (r *RatingRepo) ListByResource(query RatingListQuery) (*RatingListResult, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	db := r.db.Model(&model.Rating{}).Where("resource_id = ?", query.ResourceID)
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计评分失败: %w", err)
	}

	items := make([]model.Rating, 0)
	offset := (query.Page - 1) * query.PageSize
	if err := db.Session(&gorm.Session{}).
		Preload("User").
		Order("created_at DESC, id DESC").
		Offset(offset).
		Limit(query.PageSize).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询评分列表失败: %w", err)
	}

	return &RatingListResult{Items: items, Total: total}, nil
}

func (r *RatingRepo) AverageScore(resourceID uint) (float64, error) {
	var average float64
	if err := r.db.Model(&model.Rating{}).
		Where("resource_id = ?", resourceID).
		Select("COALESCE(AVG(score), 0)").
		Scan(&average).Error; err != nil {
		return 0, fmt.Errorf("计算资源平均分失败: %w", err)
	}
	return average, nil
}
