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
	result      *repository.ResourceListResult
	err         error
	lastQuery   repository.ResourceListQuery
	findResult  *model.Resource
	findErr     error
	lastFindID  uint
	lastCreated *model.Resource
	createErr   error
	lastUpdated *model.Resource
	updateErr   error
}

func (f *fakeResourceRepository) Create(resource *model.Resource) error {
	if f.createErr != nil {
		return f.createErr
	}
	resource.ID = 1
	f.lastCreated = resource
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

func (f *fakeResourceRepository) Update(resource *model.Resource) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.lastUpdated = resource
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

func TestResourceCreateStoresJSONFields(t *testing.T) {
	repo := &fakeResourceRepository{}
	svc := service.NewResourceService(repo)

	resource, err := svc.Create(context.Background(), service.CreateResourceInput{
		Title:       "机器学习入门",
		Description: "面向零基础学习者的课程",
		Type:        "course",
		CategoryID:  2,
		Tags:        []string{"AI", "Python"},
		Metadata:    map[string]any{"duration": "12小时"},
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repo.lastCreated == nil {
		t.Fatal("Create() did not call repository")
	}
	if resource.Tags != `["AI","Python"]` {
		t.Fatalf("Create() tags = %s, want JSON array", resource.Tags)
	}
	if resource.Metadata != `{"duration":"12小时"}` {
		t.Fatalf("Create() metadata = %s, want JSON object", resource.Metadata)
	}
	if resource.CategoryID != 2 {
		t.Fatalf("Create() category_id = %d, want 2", resource.CategoryID)
	}
}

func TestResourceCreateRejectsInvalidType(t *testing.T) {
	svc := service.NewResourceService(&fakeResourceRepository{})

	_, err := svc.Create(context.Background(), service.CreateResourceInput{
		Title:       "测试资源",
		Description: "描述",
		Type:        "book",
		CategoryID:  1,
	})
	assertErrorCode(t, err, apperror.CodeBadRequest)
}

func TestResourceCreateMapsRepositoryError(t *testing.T) {
	repo := &fakeResourceRepository{createErr: errors.New("db error")}
	svc := service.NewResourceService(repo)

	_, err := svc.Create(context.Background(), service.CreateResourceInput{
		Title:       "测试资源",
		Description: "描述",
		Type:        "course",
		CategoryID:  1,
	})
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestResourceUpdateAppliesProvidedFields(t *testing.T) {
	repo := &fakeResourceRepository{
		findResult: &model.Resource{
			Title:       "旧标题",
			Description: "旧描述",
			Type:        "article",
			CategoryID:  1,
			Tags:        `["旧"]`,
			Metadata:    `{"old":true}`,
		},
	}
	svc := service.NewResourceService(repo)
	title := "新标题"
	resourceType := "course"
	categoryID := uint(3)

	resource, err := svc.Update(context.Background(), service.UpdateResourceInput{
		ID:         7,
		Title:      &title,
		Type:       &resourceType,
		CategoryID: &categoryID,
		Tags:       &[]string{"AI", "Python"},
		Metadata:   &map[string]any{"duration": "12小时"},
	})

	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repo.lastUpdated == nil {
		t.Fatal("Update() did not call repository")
	}
	if resource.Title != "新标题" || resource.Type != "course" || resource.CategoryID != 3 {
		t.Fatalf("Update() fields were not applied")
	}
	if resource.Description != "旧描述" {
		t.Fatalf("Update() description = %q, want 旧描述", resource.Description)
	}
	if resource.Tags != `["AI","Python"]` || resource.Metadata != `{"duration":"12小时"}` {
		t.Fatalf("Update() JSON fields were not serialized")
	}
}

func TestResourceUpdateMapsNotFound(t *testing.T) {
	repo := &fakeResourceRepository{findErr: repository.ErrNotFound}
	svc := service.NewResourceService(repo)

	_, err := svc.Update(context.Background(), service.UpdateResourceInput{ID: 7})
	assertErrorCode(t, err, apperror.CodeNotFound)
}

func TestResourceUpdateRejectsInvalidType(t *testing.T) {
	repo := &fakeResourceRepository{findResult: &model.Resource{}}
	svc := service.NewResourceService(repo)
	resourceType := "book"

	_, err := svc.Update(context.Background(), service.UpdateResourceInput{
		ID:   7,
		Type: &resourceType,
	})
	assertErrorCode(t, err, apperror.CodeBadRequest)
}

func TestResourceUpdateMapsRepositoryError(t *testing.T) {
	repo := &fakeResourceRepository{
		findResult: &model.Resource{},
		updateErr:  errors.New("db error"),
	}
	svc := service.NewResourceService(repo)

	_, err := svc.Update(context.Background(), service.UpdateResourceInput{ID: 7})
	assertErrorCode(t, err, apperror.CodeInternal)
}
