package service_test

import (
	"errors"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"gorm.io/gorm"
)

type fakeRecommendationRepository struct {
	found         *model.Recommendation
	findErr       error
	replaceUserID uint
	replaceIDs    string
	replaceErr    error
	saved         *model.Recommendation
}

func (f *fakeRecommendationRepository) FindByUserID(userID uint) (*model.Recommendation, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	return f.found, nil
}

func (f *fakeRecommendationRepository) Replace(userID uint, resourceIDs string, now int64) (*model.Recommendation, error) {
	if f.replaceErr != nil {
		return nil, f.replaceErr
	}
	f.replaceUserID = userID
	f.replaceIDs = resourceIDs
	f.saved = &model.Recommendation{
		ID:          1,
		UserID:      userID,
		ResourceIDs: resourceIDs,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return f.saved, nil
}

func TestRecommendGetCacheHitReturnsCachedOrder(t *testing.T) {
	recRepo := &fakeRecommendationRepository{
		found: &model.Recommendation{
			UserID:      7,
			ResourceIDs: `[3,1,2]`,
			UpdatedAt:   1000,
		},
	}
	// FindByIDs 返回乱序，服务应按缓存顺序 [3,1,2] 重排
	resRepo := &fakeResourceRepository{
		findByIDsResult: []model.Resource{
			{Model: gorm.Model{ID: 1}, Title: "一"},
			{Model: gorm.Model{ID: 2}, Title: "二"},
			{Model: gorm.Model{ID: 3}, Title: "三"},
		},
	}
	svc := service.NewRecommendationService(recRepo, resRepo)

	result, err := svc.Get(7, 20)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(result.List) != 3 {
		t.Fatalf("Get() list len = %d, want 3", len(result.List))
	}
	if result.List[0].ID != 3 || result.List[1].ID != 1 || result.List[2].ID != 2 {
		t.Fatalf("Get() order = %v, want [3 1 2]", []uint{result.List[0].ID, result.List[1].ID, result.List[2].ID})
	}
	if result.UpdatedAt != 1000 {
		t.Fatalf("Get() updated_at = %d, want 1000", result.UpdatedAt)
	}
	if recRepo.replaceUserID != 0 {
		t.Fatal("Get() cache hit should not rewrite cache")
	}
}

func TestRecommendGetCacheHitTruncatesToLimit(t *testing.T) {
	recRepo := &fakeRecommendationRepository{
		found: &model.Recommendation{ResourceIDs: `[1,2,3,4]`, UpdatedAt: 1000},
	}
	resRepo := &fakeResourceRepository{
		findByIDsResult: []model.Resource{{Model: gorm.Model{ID: 1}}, {Model: gorm.Model{ID: 2}}, {Model: gorm.Model{ID: 3}}, {Model: gorm.Model{ID: 4}}},
	}
	svc := service.NewRecommendationService(recRepo, resRepo)

	result, err := svc.Get(7, 2)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(result.List) != 2 {
		t.Fatalf("Get() list len = %d, want 2", len(result.List))
	}
}

func TestRecommendGetCacheHitSkipsDeletedResources(t *testing.T) {
	recRepo := &fakeRecommendationRepository{
		found: &model.Recommendation{ResourceIDs: `[1,99,2]`, UpdatedAt: 1000},
	}
	resRepo := &fakeResourceRepository{
		findByIDsResult: []model.Resource{{Model: gorm.Model{ID: 1}}, {Model: gorm.Model{ID: 2}}}, // 99 已被删除
	}
	svc := service.NewRecommendationService(recRepo, resRepo)

	result, err := svc.Get(7, 20)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(result.List) != 2 {
		t.Fatalf("Get() list len = %d, want 2", len(result.List))
	}
	if result.List[0].ID != 1 || result.List[1].ID != 2 {
		t.Fatalf("Get() order = %v, want [1 2]", []uint{result.List[0].ID, result.List[1].ID})
	}
}

func TestRecommendGetMissGeneratesFallbackAndWritesCache(t *testing.T) {
	recRepo := &fakeRecommendationRepository{} // 未命中
	resRepo := &fakeResourceRepository{
		result: &repository.ResourceListResult{
			Items: []model.Resource{{Model: gorm.Model{ID: 5}, Title: "高分课"}, {Model: gorm.Model{ID: 8}, Title: "次高分课"}},
			Total: 2,
		},
	}
	svc := service.NewRecommendationService(recRepo, resRepo)

	result, err := svc.Get(7, 20)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if resRepo.lastQuery.Sort != "rating" {
		t.Fatalf("Get() fallback sort = %q, want rating", resRepo.lastQuery.Sort)
	}
	if len(result.List) != 2 {
		t.Fatalf("Get() list len = %d, want 2", len(result.List))
	}
	if recRepo.replaceUserID != 7 {
		t.Fatalf("Get() cache write user = %d, want 7", recRepo.replaceUserID)
	}
	if recRepo.replaceIDs != `[5,8]` {
		t.Fatalf("Get() cache ids = %s, want [5,8]", recRepo.replaceIDs)
	}
	if result.UpdatedAt != recRepo.saved.UpdatedAt {
		t.Fatalf("Get() updated_at = %d, want saved %d", result.UpdatedAt, recRepo.saved.UpdatedAt)
	}
}

func TestRecommendGetClampsLimit(t *testing.T) {
	recRepo := &fakeRecommendationRepository{}
	resRepo := &fakeResourceRepository{result: &repository.ResourceListResult{Items: []model.Resource{}}}
	svc := service.NewRecommendationService(recRepo, resRepo)

	// limit = 0 → 默认 20
	_, err := svc.Get(7, 0)
	if err != nil {
		t.Fatalf("Get(0) error = %v", err)
	}
	if resRepo.lastQuery.PageSize != 20 {
		t.Fatalf("Get(0) page_size = %d, want 20", resRepo.lastQuery.PageSize)
	}

	// limit = 100 → 上限 50
	_, err = svc.Get(7, 100)
	if err != nil {
		t.Fatalf("Get(100) error = %v", err)
	}
	if resRepo.lastQuery.PageSize != 50 {
		t.Fatalf("Get(100) page_size = %d, want 50", resRepo.lastQuery.PageSize)
	}
}

func TestRecommendGetCorruptCacheRegenerates(t *testing.T) {
	recRepo := &fakeRecommendationRepository{
		found: &model.Recommendation{ResourceIDs: `not-json`, UpdatedAt: 1000},
	}
	resRepo := &fakeResourceRepository{
		result: &repository.ResourceListResult{Items: []model.Resource{{Model: gorm.Model{ID: 5}}}},
	}
	svc := service.NewRecommendationService(recRepo, resRepo)

	result, err := svc.Get(7, 20)

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(result.List) != 1 {
		t.Fatalf("Get() list len = %d, want 1 (regenerated)", len(result.List))
	}
	if recRepo.replaceUserID != 7 {
		t.Fatal("Get() corrupt cache should be regenerated")
	}
}

func TestRecommendGetMapsFindError(t *testing.T) {
	recRepo := &fakeRecommendationRepository{findErr: errors.New("db error")}
	svc := service.NewRecommendationService(recRepo, &fakeResourceRepository{})

	_, err := svc.Get(7, 20)
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestRecommendGetMapsFallbackError(t *testing.T) {
	recRepo := &fakeRecommendationRepository{}
	resRepo := &fakeResourceRepository{err: errors.New("db error")}
	svc := service.NewRecommendationService(recRepo, resRepo)

	_, err := svc.Get(7, 20)
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestRecommendGetMapsReplaceError(t *testing.T) {
	recRepo := &fakeRecommendationRepository{replaceErr: errors.New("db error")}
	resRepo := &fakeResourceRepository{result: &repository.ResourceListResult{Items: []model.Resource{}}}
	svc := service.NewRecommendationService(recRepo, resRepo)

	_, err := svc.Get(7, 20)
	assertErrorCode(t, err, apperror.CodeInternal)
}
