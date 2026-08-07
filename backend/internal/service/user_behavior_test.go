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

type fakeUserBehaviorRepository struct {
	createErr   error
	lastCreated *model.UserBehavior
	listResult  *repository.UserBehaviorListResult
	listErr     error
	lastQuery   repository.UserBehaviorListQuery
}

func (f *fakeUserBehaviorRepository) Create(behavior *model.UserBehavior) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.lastCreated = behavior
	return nil
}

func (f *fakeUserBehaviorRepository) List(query repository.UserBehaviorListQuery) (*repository.UserBehaviorListResult, error) {
	f.lastQuery = query
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResult, nil
}

func TestUserBehaviorRecordCreatesBehavior(t *testing.T) {
	behaviors := &fakeUserBehaviorRepository{}
	resources := &fakeResourceRepository{findResult: &model.Resource{Title: "机器学习入门"}}
	svc := service.NewUserBehaviorService(behaviors, resources)

	err := svc.Record(context.Background(), 7, 9, "view")

	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if behaviors.lastCreated == nil {
		t.Fatal("Record() did not call repository")
	}
	if behaviors.lastCreated.UserID != 7 || behaviors.lastCreated.ResourceID != 9 || behaviors.lastCreated.Action != "view" {
		t.Fatalf("Record() behavior fields are incorrect")
	}
	if resources.lastFindID != 9 {
		t.Fatalf("Record() resource id = %d, want 9", resources.lastFindID)
	}
}

func TestUserBehaviorRecordRejectsInvalidAction(t *testing.T) {
	svc := service.NewUserBehaviorService(&fakeUserBehaviorRepository{}, &fakeResourceRepository{})

	err := svc.Record(context.Background(), 7, 9, "download")
	assertErrorCode(t, err, apperror.CodeBadRequest)
}

func TestUserBehaviorRecordMapsResourceNotFound(t *testing.T) {
	resources := &fakeResourceRepository{findErr: repository.ErrNotFound}
	svc := service.NewUserBehaviorService(&fakeUserBehaviorRepository{}, resources)

	err := svc.Record(context.Background(), 7, 9, "view")
	assertErrorCode(t, err, apperror.CodeNotFound)
}

func TestUserBehaviorRecordMapsRepositoryError(t *testing.T) {
	behaviors := &fakeUserBehaviorRepository{createErr: errors.New("db error")}
	resources := &fakeResourceRepository{findResult: &model.Resource{}}
	svc := service.NewUserBehaviorService(behaviors, resources)

	err := svc.Record(context.Background(), 7, 9, "view")
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestUserBehaviorListTrimsActionAndPassesQuery(t *testing.T) {
	behaviors := &fakeUserBehaviorRepository{
		listResult: &repository.UserBehaviorListResult{
			Items: []model.UserBehavior{{Action: "view"}},
			Total: 1,
		},
	}
	svc := service.NewUserBehaviorService(behaviors, &fakeResourceRepository{})

	result, err := svc.List(context.Background(), repository.UserBehaviorListQuery{
		UserID:   7,
		Action:   "  view  ",
		Page:     2,
		PageSize: 10,
	})

	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("List() total = %d, want 1", result.Total)
	}
	if behaviors.lastQuery.UserID != 7 || behaviors.lastQuery.Action != "view" {
		t.Fatalf("List() query was not passed through")
	}
	if behaviors.lastQuery.Page != 2 || behaviors.lastQuery.PageSize != 10 {
		t.Fatalf("List() page/page_size = %d/%d, want 2/10", behaviors.lastQuery.Page, behaviors.lastQuery.PageSize)
	}
}

func TestUserBehaviorListRejectsInvalidAction(t *testing.T) {
	svc := service.NewUserBehaviorService(&fakeUserBehaviorRepository{}, &fakeResourceRepository{})

	_, err := svc.List(context.Background(), repository.UserBehaviorListQuery{Action: "download"})
	assertErrorCode(t, err, apperror.CodeBadRequest)
}

func TestUserBehaviorListMapsRepositoryError(t *testing.T) {
	behaviors := &fakeUserBehaviorRepository{listErr: errors.New("db error")}
	svc := service.NewUserBehaviorService(behaviors, &fakeResourceRepository{})

	_, err := svc.List(context.Background(), repository.UserBehaviorListQuery{})
	assertErrorCode(t, err, apperror.CodeInternal)
}
