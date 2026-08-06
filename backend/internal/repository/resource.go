package repository

import (
	"errors"
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// ResourceRepository 资源数据访问接口
type ResourceRepository interface {
	Create(resource *model.Resource) error
	FindByID(id uint) (*model.Resource, error)
	Update(resource *model.Resource) error
	Delete(id uint) error
}

type ResourceRepo struct {
	db *gorm.DB
}

func NewResourceRepository(db *gorm.DB) *ResourceRepo {
	return &ResourceRepo{db: db}
}

func (r *ResourceRepo) Create(resource *model.Resource) error {
	if err := r.db.Create(resource).Error; err != nil {
		return fmt.Errorf("创建资源失败: %w", err)
	}
	return nil
}

func (r *ResourceRepo) FindByID(id uint) (*model.Resource, error) {
	resource := &model.Resource{}
	err := r.db.Preload("Category").First(resource, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("按 ID 查询资源失败: %w", err)
	}
	return resource, nil
}

func (r *ResourceRepo) Update(resource *model.Resource) error {
	if err := r.db.Save(resource).Error; err != nil {
		return fmt.Errorf("更新资源失败: %w", err)
	}
	return nil
}

func (r *ResourceRepo) Delete(id uint) error {
	result := r.db.Delete(&model.Resource{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除资源失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
