package service_test

import (
	"context"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

type fakeCategoryRepository struct {
	categories []model.Category
	nextID     uint
}

func (f *fakeCategoryRepository) List() ([]model.Category, error) {
	return append([]model.Category(nil), f.categories...), nil
}

func (f *fakeCategoryRepository) Create(category *model.Category) error {
	f.nextID++
	category.ID = f.nextID
	f.categories = append(f.categories, *category)
	return nil
}

func (f *fakeCategoryRepository) FindByName(name string) (*model.Category, error) {
	for i := range f.categories {
		if f.categories[i].Name == name {
			return &f.categories[i], nil
		}
	}
	return nil, repository.ErrNotFound
}

func TestCategoryListReturnsAllCategories(t *testing.T) {
	categories := &fakeCategoryRepository{
		categories: []model.Category{
			{Name: "人工智能"},
			{Name: "前端开发"},
		},
	}
	svc := service.NewCategoryService(categories)

	items, err := svc.List(context.Background())

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() len = %d, want 2", len(items))
	}
	if items[0].Name != "人工智能" {
		t.Fatalf("List() first name = %q, want 人工智能", items[0].Name)
	}
}

func TestCategoryCreateStoresTrimmedName(t *testing.T) {
	categories := &fakeCategoryRepository{}
	svc := service.NewCategoryService(categories)

	category, err := svc.Create(context.Background(), "  数据科学  ", " 数据分析  ")

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if category.ID != 1 {
		t.Fatalf("Create() ID = %d, want 1", category.ID)
	}
	if category.Name != "数据科学" {
		t.Fatalf("Create() name = %q, want 数据科学", category.Name)
	}
	if category.Description != "数据分析" {
		t.Fatalf("Create() description = %q, want 数据分析", category.Description)
	}
}

func TestCategoryCreateRejectsEmptyName(t *testing.T) {
	svc := service.NewCategoryService(&fakeCategoryRepository{})

	_, err := svc.Create(context.Background(), "  ", "")
	assertErrorCode(t, err, apperror.CodeBadRequest)
}
