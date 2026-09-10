package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// ResourceRepository 资源数据访问接口
type ResourceRepository interface {
	Create(resource *model.Resource) error
	List(query ResourceListQuery) (*ResourceListResult, error)
	FindByID(id uint) (*model.Resource, error)
	FindByIDs(ids []uint) ([]model.Resource, error)
	FindBySourceURLs(urls []string) ([]model.Resource, error)
	Update(resource *model.Resource) error
	Delete(id uint) error
}

// ResourceListQuery 资源列表查询条件
type ResourceListQuery struct {
	Page       int
	PageSize   int
	Keyword    string
	CategoryID uint
	Type       string
	Tags       []string
	Sort       string
}

// ResourceListResult 资源列表分页结果
type ResourceListResult struct {
	Items []model.Resource
	Total int64
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

func (r *ResourceRepo) List(query ResourceListQuery) (*ResourceListResult, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	db := r.db.Model(&model.Resource{})
	if query.Keyword != "" {
		keyword := "%" + query.Keyword + "%"
		db = db.Where("title LIKE ? OR description LIKE ?", keyword, keyword)
	}
	if query.CategoryID != 0 {
		db = db.Where("category_id = ?", query.CategoryID)
	}
	if query.Type != "" {
		db = db.Where("type = ?", query.Type)
	}
	for _, tag := range query.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		tagJSON, _ := json.Marshal(tag)
		db = db.Where("JSON_CONTAINS(tags, ?)", string(tagJSON))
	}

	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计资源列表失败: %w", err)
	}

	items := make([]model.Resource, 0)
	order := "created_at DESC, id DESC"
	switch query.Sort {
	case "popular":
		order = "view_count DESC, id DESC"
	case "rating":
		order = "avg_rating DESC, id DESC"
	}

	offset := (query.Page - 1) * query.PageSize
	if err := db.Session(&gorm.Session{}).
		Preload("Category").
		Order(order).
		Offset(offset).
		Limit(query.PageSize).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询资源列表失败: %w", err)
	}

	return &ResourceListResult{Items: items, Total: total}, nil
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

func (r *ResourceRepo) FindByIDs(ids []uint) ([]model.Resource, error) {
	if len(ids) == 0 {
		return []model.Resource{}, nil
	}
	items := make([]model.Resource, 0)
	if err := r.db.Preload("Category").Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("按 ID 批量查询资源失败: %w", err)
	}
	return items, nil
}

func (r *ResourceRepo) FindBySourceURLs(urls []string) ([]model.Resource, error) {
	if len(urls) == 0 {
		return []model.Resource{}, nil
	}
	items := make([]model.Resource, 0)
	if err := r.db.Preload("Category").Where("source_url IN ?", urls).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("按来源链接批量查询资源失败: %w", err)
	}
	return items, nil
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
