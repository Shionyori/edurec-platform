package repository

import (
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// CommentRepository B 站评论数据访问接口
type CommentRepository interface {
	HasByResourceID(resourceID uint) (bool, error)
	ListByResourceID(resourceID uint, limit int) ([]model.ResourceComment, error)
	BatchCreate(comments []model.ResourceComment) error
}

type CommentRepo struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepo {
	return &CommentRepo{db: db}
}

func (r *CommentRepo) HasByResourceID(resourceID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.ResourceComment{}).
		Where("resource_id = ?", resourceID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("统计评论失败: %w", err)
	}
	return count > 0, nil
}

func (r *CommentRepo) ListByResourceID(resourceID uint, limit int) ([]model.ResourceComment, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	items := make([]model.ResourceComment, 0)
	if err := r.db.Where("resource_id = ?", resourceID).
		Order("floor ASC, id ASC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询评论列表失败: %w", err)
	}
	return items, nil
}

func (r *CommentRepo) BatchCreate(comments []model.ResourceComment) error {
	if len(comments) == 0 {
		return nil
	}
	if err := r.db.Create(&comments).Error; err != nil {
		return fmt.Errorf("批量写入评论失败: %w", err)
	}
	return nil
}
