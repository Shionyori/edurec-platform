package repository

import (
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// UserBehaviorRepository 用户行为数据访问接口
type UserBehaviorRepository interface {
	Create(behavior *model.UserBehavior) error
	List(query UserBehaviorListQuery) (*UserBehaviorListResult, error)
}

// UserBehaviorListQuery 用户行为历史查询条件
type UserBehaviorListQuery struct {
	UserID   uint
	Action   string
	Page     int
	PageSize int
}

// UserBehaviorListResult 用户行为历史分页结果
type UserBehaviorListResult struct {
	Items []model.UserBehavior
	Total int64
}

type UserBehaviorRepo struct {
	db *gorm.DB
}

func NewUserBehaviorRepository(db *gorm.DB) *UserBehaviorRepo {
	return &UserBehaviorRepo{db: db}
}

func (r *UserBehaviorRepo) Create(behavior *model.UserBehavior) error {
	if err := r.db.Create(behavior).Error; err != nil {
		return fmt.Errorf("创建用户行为失败: %w", err)
	}
	return nil
}

func (r *UserBehaviorRepo) List(query UserBehaviorListQuery) (*UserBehaviorListResult, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	db := r.db.Model(&model.UserBehavior{})
	if query.UserID != 0 {
		db = db.Where("user_id = ?", query.UserID)
	}
	if query.Action != "" {
		db = db.Where("action = ?", query.Action)
	}

	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计用户行为失败: %w", err)
	}

	items := make([]model.UserBehavior, 0)
	offset := (query.Page - 1) * query.PageSize
	if err := db.Session(&gorm.Session{}).
		Preload("Resource").
		Order("created_at DESC, id DESC").
		Offset(offset).
		Limit(query.PageSize).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询用户行为失败: %w", err)
	}

	return &UserBehaviorListResult{Items: items, Total: total}, nil
}
