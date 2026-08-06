package service

import (
	"context"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

type CategoryService struct {
	categories repository.CategoryRepository
}

func NewCategoryService(categories repository.CategoryRepository) *CategoryService {
	return &CategoryService{categories: categories}
}

func (s *CategoryService) List(ctx context.Context) ([]model.Category, error) {
	categories, err := s.categories.List()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return categories, nil
}

func (s *CategoryService) Create(ctx context.Context, name string, description string) (*model.Category, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return nil, apperror.BadRequest("请求参数错误")
	}

	category := &model.Category{
		Name:        name,
		Description: description,
	}
	if err := s.categories.Create(category); err != nil {
		return nil, apperror.Internal(err)
	}
	return category, nil
}
