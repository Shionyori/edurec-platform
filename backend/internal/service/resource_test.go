package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

type fakeResourceRepository struct {
	result     *repository.ResourceListResult
	err        error
	lastQuery  repository.ResourceListQuery
	findResult *model.Resource
	findErr    error
	lastFindID uint
}

func (f *fakeResourceRepository) Create(_ *model.Resource) error {
	return nil
}

func (f *fakeResourceRepository) List(query repository.ResourceListQuery) (*repository.ResourceListResult, error) {
	f.lastQuery = query
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeResourceRepository) FindByID(id uint) (*model.Resource, error) {
	f.lastFindID = id
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.findResult, nil
}

func (f *fakeResourceRepository) Update(_ *model.Resource) error {
	return nil
}

func (f *fakeResourceRepository) Delete(_ uint) error {
	return nil
}

func TestResourceListPassesQuery(t *testing.T) {
	repo := &fakeResourceRepository{
		result: &repository.ResourceListResult{
			Items: []model.Resource{{Title: "机器学习入门"}},
			Total: 1,
		},
	}
	svc := service.NewResourceService(repo)
	query := repository.ResourceListQuery{
		Page:       2,
		PageSize:   10,
		Keyword:    "机器学习",
		CategoryID: 3,
		Type:       "course",
		Tags:       []string{"AI", "Python"},
		Sort:       "rating",
	}

	result, err := svc.List(context.Background(), query)

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("List() total = %d, want 1", result.Total)
	}
	if repo.lastQuery.Page != 2 || repo.lastQuery.PageSize != 10 {
		t.Fatalf("List() page/page_size = %d/%d, want 2/10", repo.lastQuery.Page, repo.lastQuery.PageSize)
	}
	if repo.lastQuery.Keyword != "机器学习" || repo.lastQuery.CategoryID != 3 || repo.lastQuery.Type != "course" {
		t.Fatalf("List() filters were not passed through")
	}
	if len(repo.lastQuery.Tags) != 2 {
		t.Fatalf("List() tags len = %d, want 2", len(repo.lastQuery.Tags))
	}
}

func TestResourceListTrimsAndDefaultsSort(t *testing.T) {
	repo := &fakeResourceRepository{result: &repository.ResourceListResult{}}
	svc := service.NewResourceService(repo)

	_, err := svc.List(context.Background(), repository.ResourceListQuery{
		Keyword: "  Go  ",
		Tags:    []string{" 后端 ", " Go "},
	})

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repo.lastQuery.Keyword != "Go" {
		t.Fatalf("List() keyword = %q, want Go", repo.lastQuery.Keyword)
	}
	if repo.lastQuery.Tags[0] != "后端" || repo.lastQuery.Tags[1] != "Go" {
		t.Fatalf("List() tags = %v, want trimmed values", repo.lastQuery.Tags)
	}
	if repo.lastQuery.Sort != "latest" {
		t.Fatalf("List() sort = %q, want latest", repo.lastQuery.Sort)
	}
}

func TestResourceListMapsRepositoryError(t *testing.T) {
	repo := &fakeResourceRepository{err: errors.New("db error")}
	svc := service.NewResourceService(repo)

	_, err := svc.List(context.Background(), repository.ResourceListQuery{})
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestResourceGetByIDReturnsResource(t *testing.T) {
	repo := &fakeResourceRepository{
		findResult: &model.Resource{Title: "机器学习入门"},
	}
	svc := service.NewResourceService(repo)

	resource, err := svc.GetByID(context.Background(), 7)

	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if repo.lastFindID != 7 {
		t.Fatalf("GetByID() id = %d, want 7", repo.lastFindID)
	}
	if resource.Title != "机器学习入门" {
		t.Fatalf("GetByID() title = %q, want 机器学习入门", resource.Title)
	}
}

func TestResourceGetByIDMapsNotFound(t *testing.T) {
	repo := &fakeResourceRepository{findErr: repository.ErrNotFound}
	svc := service.NewResourceService(repo)

	_, err := svc.GetByID(context.Background(), 7)
	assertErrorCode(t, err, apperror.CodeNotFound)
}

func TestResourceGetByIDMapsRepositoryError(t *testing.T) {
	repo := &fakeResourceRepository{findErr: errors.New("db error")}
	svc := service.NewResourceService(repo)

	_, err := svc.GetByID(context.Background(), 7)
	assertErrorCode(t, err, apperror.CodeInternal)
}
