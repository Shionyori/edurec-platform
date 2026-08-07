package repository

import (
	"fmt"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

type AdminUserListQuery struct {
	Page     int
	PageSize int
	Keyword  string
}

type AdminUserListResult struct {
	Items           []model.User
	IsAdminByUserID map[uint]bool
	Total           int64
}

type AdminResourceListQuery struct {
	Page       int
	PageSize   int
	Keyword    string
	Type       string
	CategoryID uint
}

type AdminResourceListResult struct {
	Items []model.Resource
	Total int64
}

// AdminRepository 管理后台数据访问接口
type AdminRepository interface {
	ListUsers(query AdminUserListQuery) (*AdminUserListResult, error)
	ListResources(query AdminResourceListQuery) (*AdminResourceListResult, error)
}

type AdminRepo struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) *AdminRepo {
	return &AdminRepo{db: db}
}

func (r *AdminRepo) ListUsers(query AdminUserListQuery) (*AdminUserListResult, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	db := r.db.Model(&model.User{})
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		db = db.Where("username LIKE ? OR email LIKE ?", pattern, pattern)
	}

	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计用户列表失败: %w", err)
	}

	items := make([]model.User, 0)
	offset := (query.Page - 1) * query.PageSize
	if err := db.Session(&gorm.Session{}).
		Order("created_at DESC, id DESC").
		Offset(offset).
		Limit(query.PageSize).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	adminUserIDs := make([]uint, 0)
	if err := r.db.Model(&model.Admin{}).Pluck("user_id", &adminUserIDs).Error; err != nil {
		return nil, fmt.Errorf("查询管理员列表失败: %w", err)
	}
	isAdminByUserID := make(map[uint]bool, len(adminUserIDs))
	for _, userID := range adminUserIDs {
		isAdminByUserID[userID] = true
	}

	return &AdminUserListResult{
		Items:           items,
		IsAdminByUserID: isAdminByUserID,
		Total:           total,
	}, nil
}

func (r *AdminRepo) ListResources(query AdminResourceListQuery) (*AdminResourceListResult, error) {
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
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		db = db.Where("title LIKE ?", pattern)
	}
	if query.Type != "" {
		db = db.Where("type = ?", query.Type)
	}
	if query.CategoryID != 0 {
		db = db.Where("category_id = ?", query.CategoryID)
	}

	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计资源列表失败: %w", err)
	}

	items := make([]model.Resource, 0)
	offset := (query.Page - 1) * query.PageSize
	if err := db.Session(&gorm.Session{}).
		Preload("Category").
		Order("created_at DESC, id DESC").
		Offset(offset).
		Limit(query.PageSize).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询资源列表失败: %w", err)
	}

	return &AdminResourceListResult{Items: items, Total: total}, nil
}
