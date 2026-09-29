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
	// Exists 判断某用户对某资源是否已有指定行为
	Exists(userID, resourceID uint, action string) (bool, error)
	// DeleteByUserResourceAction 删除某用户对某资源的指定行为（用于取消收藏）
	DeleteByUserResourceAction(userID, resourceID uint, action string) error
	// ListFavoriteResourceIDs 按「最近收藏」倒序返回收藏的资源 ID（去重、分页）
	ListFavoriteResourceIDs(userID uint, page, pageSize int) ([]uint, int64, error)
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

func (r *UserBehaviorRepo) Exists(userID, resourceID uint, action string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.UserBehavior{}).
		Where("user_id = ? AND resource_id = ? AND action = ?", userID, resourceID, action).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("查询用户行为失败: %w", err)
	}
	return count > 0, nil
}

func (r *UserBehaviorRepo) DeleteByUserResourceAction(userID, resourceID uint, action string) error {
	if err := r.db.
		Where("user_id = ? AND resource_id = ? AND action = ?", userID, resourceID, action).
		Delete(&model.UserBehavior{}).Error; err != nil {
		return fmt.Errorf("删除用户行为失败: %w", err)
	}
	return nil
}

// ListFavoriteResourceIDs 收藏的资源 ID：按资源去重、以最近一次收藏时间倒序。
// 返回 (ids, 去重后的总数)，供上层拿完整资源并还原顺序。
func (r *UserBehaviorRepo) ListFavoriteResourceIDs(userID uint, page, pageSize int) ([]uint, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int64
	if err := r.db.Model(&model.UserBehavior{}).
		Where("user_id = ? AND action = ?", userID, "favorite").
		Distinct("resource_id").
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计收藏失败: %w", err)
	}

	ids := make([]uint, 0, pageSize)
	if err := r.db.Table("user_behaviors").
		Where("user_id = ? AND action = ?", userID, "favorite").
		Group("resource_id").
		Order("MAX(created_at) DESC").
		Offset((page-1)*pageSize).
		Limit(pageSize).
		Pluck("resource_id", &ids).Error; err != nil {
		return nil, 0, fmt.Errorf("查询收藏失败: %w", err)
	}
	return ids, total, nil
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
