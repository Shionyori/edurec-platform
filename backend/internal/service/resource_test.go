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
	result           *repository.ResourceListResult
	err              error
	lastQuery        repository.ResourceListQuery
	findResult       *model.Resource
	findErr          error
	lastFindID       uint
	findByIDsResult  []model.Resource
	findByIDsErr     error
	findByURLsResult []model.Resource
	findByURLsErr    error
	lastCreated      *model.Resource
	createErr        error
	lastUpdated      *model.Resource
	updateErr        error
	lastDeleteID     uint
	deleteErr        error
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

func (f *fakeResourceRepository) FindByIDs(ids []uint) ([]model.Resource, error) {
	if f.findByIDsErr != nil {
		return nil, f.findByIDsErr
	}
	return f.findByIDsResult, nil
}

func (f *fakeResourceRepository) FindBySourceURLs([]string) ([]model.Resource, error) {
	if f.findByURLsErr != nil {
		return nil, f.findByURLsErr
	}
	return f.findByURLsResult, nil
}

func (f *fakeResourceRepository) Update(resource *model.Resource) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.lastUpdated = resource
	return nil
}

func (f *fakeResourceRepository) Delete(id uint) error {
	f.lastDeleteID = id
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return nil
}

type fakeOnlineSearcher struct {
	resources   []model.Resource
	hasMore     bool
	err         error
	lastKeyword string
	lastPage    int
}

func (f *fakeOnlineSearcher) SearchAndImport(keyword string, page int) ([]model.Resource, bool, error) {
	f.lastKeyword = keyword
	f.lastPage = page
	return f.resources, f.hasMore, f.err
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

func TestResourceListOnlinePageCallsSearcher(t *testing.T) {
	repo := &fakeResourceRepository{result: &repository.ResourceListResult{}}
	online := &fakeOnlineSearcher{
		resources: []model.Resource{{Title: "B站视频"}},
		hasMore:   true,
	}
	svc := service.NewResourceService(repo)
	svc.SetBilibiliOnline(online)

	result, err := svc.List(context.Background(), repository.ResourceListQuery{
		Keyword:    "机器学习",
		OnlinePage: 2,
	})

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if online.lastKeyword != "机器学习" || online.lastPage != 2 {
		t.Fatalf("List() online args = %q/%d, want 机器学习/2", online.lastKeyword, online.lastPage)
	}
	if len(result.Items) != 1 || result.Items[0].Title != "B站视频" {
		t.Fatalf("List() items = %v, want [B站视频]", result.Items)
	}
	if !result.HasMore {
		t.Fatalf("List() has_more = false, want true")
	}
	if repo.lastQuery.Keyword != "" {
		t.Fatalf("List() should not hit local repository in online branch")
	}
}

func TestResourceListOnlinePageWithFiltersFallsThroughToLocal(t *testing.T) {
	repo := &fakeResourceRepository{result: &repository.ResourceListResult{Items: []model.Resource{{Title: "本地课程"}}}}
	online := &fakeOnlineSearcher{}
	svc := service.NewResourceService(repo)
	svc.SetBilibiliOnline(online)

	result, err := svc.List(context.Background(), repository.ResourceListQuery{
		Keyword:    "机器学习",
		CategoryID: 3,
		OnlinePage: 1,
	})

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if online.lastKeyword != "" {
		t.Fatalf("List() should not crawl B站 when filters are present")
	}
	if repo.lastQuery.CategoryID != 3 {
		t.Fatalf("List() should fall through to local repository")
	}
	if result.Total != 0 || len(result.Items) != 1 || result.Items[0].Title != "本地课程" {
		t.Fatalf("List() result = %+v, want local items", result)
	}
}

func TestResourceListOnlinePageSearcherErrorDegrades(t *testing.T) {
	repo := &fakeResourceRepository{result: &repository.ResourceListResult{}}
	online := &fakeOnlineSearcher{err: errors.New("exec timeout")}
	svc := service.NewResourceService(repo)
	svc.SetBilibiliOnline(online)

	result, err := svc.List(context.Background(), repository.ResourceListQuery{
		Keyword:    "机器学习",
		OnlinePage: 1,
	})

	if err != nil {
		t.Fatalf("List() error = %v, want nil (degrade to empty)", err)
	}
	if result == nil || len(result.Items) != 0 {
		t.Fatalf("List() items = %v, want empty", result)
	}
	if result.HasMore {
		t.Fatalf("List() has_more = true, want false on failure")
	}
}

func TestResourceListOnlinePageWithoutSearcherFallsThroughToLocal(t *testing.T) {
	repo := &fakeResourceRepository{result: &repository.ResourceListResult{Items: []model.Resource{{Title: "本地资源"}}}}
	svc := service.NewResourceService(repo) // 未注入 online

	result, err := svc.List(context.Background(), repository.ResourceListQuery{
		Keyword:    "机器学习",
		OnlinePage: 1,
	})

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repo.lastQuery.Keyword != "机器学习" {
		t.Fatalf("List() should fall through to local repository when online is nil")
	}
	if len(result.Items) != 1 || result.Items[0].Title != "本地资源" {
		t.Fatalf("List() result = %+v, want local items", result)
	}
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

func TestResourceDeleteCallsRepository(t *testing.T) {
	repo := &fakeResourceRepository{}
	svc := service.NewResourceService(repo)

	err := svc.Delete(context.Background(), 7)

	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if repo.lastDeleteID != 7 {
		t.Fatalf("Delete() id = %d, want 7", repo.lastDeleteID)
	}
}

func TestResourceDeleteMapsNotFound(t *testing.T) {
	repo := &fakeResourceRepository{deleteErr: repository.ErrNotFound}
	svc := service.NewResourceService(repo)

	err := svc.Delete(context.Background(), 7)
	assertErrorCode(t, err, apperror.CodeNotFound)
}

func TestResourceDeleteMapsRepositoryError(t *testing.T) {
	repo := &fakeResourceRepository{deleteErr: errors.New("db error")}
	svc := service.NewResourceService(repo)

	err := svc.Delete(context.Background(), 7)
	assertErrorCode(t, err, apperror.CodeInternal)
}
