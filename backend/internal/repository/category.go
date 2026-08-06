package repository

import (
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// CategoryRepository 分类数据访问接口
type CategoryRepository interface {
	List() ([]model.Category, error)
	Create(category *model.Category) error
}

type CategoryRepo struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepo {
	return &CategoryRepo{db: db}
}

func (r *CategoryRepo) List() ([]model.Category, error) {
	var categories []model.Category
	if err := r.db.Order("id ASC").Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("查询分类列表失败: %w", err)
	}
	return categories, nil
}

func (r *CategoryRepo) Create(category *model.Category) error {
	if err := r.db.Create(category).Error; err != nil {
		return fmt.Errorf("创建分类失败: %w", err)
	}
	return nil
}
