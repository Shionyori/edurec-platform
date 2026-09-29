package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"gorm.io/gorm"
)

type fakeImpressionRepository struct {
	created []model.ResourceImpression
	err     error
}

func (f *fakeImpressionRepository) CreateBatch(items []model.ResourceImpression) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, items...)
	return nil
}

func resourcesWithIDs(ids ...uint) []model.Resource {
	out := make([]model.Resource, 0, len(ids))
	for _, id := range ids {
		out = append(out, model.Resource{Model: gorm.Model{ID: id}})
	}
	return out
}

func TestResourceImpressionRecordCreatesBatch(t *testing.T) {
	impressions := &fakeImpressionRepository{}
	resources := &fakeResourceRepository{findByIDsResult: resourcesWithIDs(10, 20)}
	svc := service.NewResourceImpressionService(impressions, resources)

	err := svc.Record(context.Background(), 7, "home", []uint{10, 20})

	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if len(impressions.created) != 2 {
		t.Fatalf("Record() created %d impressions, want 2", len(impressions.created))
	}
	if impressions.created[0].UserID != 7 || impressions.created[0].Scene != "home" {
		t.Fatalf("Record() first impression fields incorrect: %+v", impressions.created[0])
	}
	if impressions.created[0].ResourceID != 10 || impressions.created[0].Position != 0 {
		t.Fatalf("Record() first impression position/resource incorrect: %+v", impressions.created[0])
	}
	if impressions.created[1].ResourceID != 20 || impressions.created[1].Position != 1 {
		t.Fatalf("Record() second impression position/resource incorrect: %+v", impressions.created[1])
	}
}

func TestResourceImpressionRecordDedupAndDropZero(t *testing.T) {
	impressions := &fakeImpressionRepository{}
	resources := &fakeResourceRepository{findByIDsResult: resourcesWithIDs(5, 6)}
	svc := service.NewResourceImpressionService(impressions, resources)

	err := svc.Record(context.Background(), 7, "home", []uint{0, 5, 5, 6})

	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if len(impressions.created) != 2 {
		t.Fatalf("Record() created %d impressions, want 2 (dedup + drop 0)", len(impressions.created))
	}
}

func TestResourceImpressionRecordRejectsBadInput(t *testing.T) {
	svc := service.NewResourceImpressionService(&fakeImpressionRepository{}, &fakeResourceRepository{})

	assertErrorCode(t, svc.Record(context.Background(), 0, "home", []uint{1}), apperror.CodeBadRequest)
	assertErrorCode(t, svc.Record(context.Background(), 7, "  ", []uint{1}), apperror.CodeBadRequest)
	assertErrorCode(t, svc.Record(context.Background(), 7, "home", nil), apperror.CodeBadRequest)
	assertErrorCode(t, svc.Record(context.Background(), 7, "home", []uint{0}), apperror.CodeBadRequest)
}

func TestResourceImpressionRecordFiltersUnknownResources(t *testing.T) {
	impressions := &fakeImpressionRepository{}
	resources := &fakeResourceRepository{findByIDsResult: resourcesWithIDs(5)}
	svc := service.NewResourceImpressionService(impressions, resources)

	err := svc.Record(context.Background(), 7, "home", []uint{5, 99})

	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if len(impressions.created) != 1 || impressions.created[0].ResourceID != 5 {
		t.Fatalf("Record() should keep only existing resources, got %+v", impressions.created)
	}
}

func TestResourceImpressionRecordCapsBatchSize(t *testing.T) {
	impressions := &fakeImpressionRepository{}
	ids := make([]uint, 150)
	existing := make([]model.Resource, 150)
	for i := range ids {
		ids[i] = uint(i + 1)
		existing[i] = model.Resource{Model: gorm.Model{ID: uint(i + 1)}}
	}
	resources := &fakeResourceRepository{findByIDsResult: existing}
	svc := service.NewResourceImpressionService(impressions, resources)

	if err := svc.Record(context.Background(), 7, "home", ids); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if len(impressions.created) != 100 {
		t.Fatalf("Record() created %d impressions, want capped at 100", len(impressions.created))
	}
}

func TestResourceImpressionRecordMapsInternalErrors(t *testing.T) {
	resources := &fakeResourceRepository{findByIDsErr: errors.New("db down")}
	svc := service.NewResourceImpressionService(&fakeImpressionRepository{}, resources)
	assertErrorCode(t, svc.Record(context.Background(), 7, "home", []uint{1}), apperror.CodeInternal)

	impressions := &fakeImpressionRepository{err: errors.New("write failed")}
	okResources := &fakeResourceRepository{findByIDsResult: resourcesWithIDs(1)}
	svc2 := service.NewResourceImpressionService(impressions, okResources)
	assertErrorCode(t, svc2.Record(context.Background(), 7, "home", []uint{1}), apperror.CodeInternal)
}
