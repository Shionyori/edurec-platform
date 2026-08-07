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

type fakeAdminRepository struct {
	userResult        *repository.AdminUserListResult
	userErr           error
	lastUserQuery     repository.AdminUserListQuery
	resourceResult    *repository.AdminResourceListResult
	resourceErr       error
	lastResourceQuery repository.AdminResourceListQuery
}

func (f *fakeAdminRepository) ListUsers(query repository.AdminUserListQuery) (*repository.AdminUserListResult, error) {
	f.lastUserQuery = query
	if f.userErr != nil {
		return nil, f.userErr
	}
	return f.userResult, nil
}

func (f *fakeAdminRepository) ListResources(query repository.AdminResourceListQuery) (*repository.AdminResourceListResult, error) {
	f.lastResourceQuery = query
	if f.resourceErr != nil {
		return nil, f.resourceErr
	}
	return f.resourceResult, nil
}

func TestAdminListUsersPassesQuery(t *testing.T) {
	repo := &fakeAdminRepository{
		userResult: &repository.AdminUserListResult{
			Items:           []model.User{{Username: "zhangsan"}},
			IsAdminByUserID: map[uint]bool{1: true},
			Total:           1,
		},
	}
	svc := service.NewAdminService(repo)

	result, err := svc.ListUsers(context.Background(), repository.AdminUserListQuery{
		Page:     2,
		PageSize: 10,
		Keyword:  "  zhangsan  ",
	})

	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("ListUsers() total = %d, want 1", result.Total)
	}
	if repo.lastUserQuery.Keyword != "zhangsan" || repo.lastUserQuery.Page != 2 || repo.lastUserQuery.PageSize != 10 {
		t.Fatalf("ListUsers() query was not passed through")
	}
}

func TestAdminListUsersMapsRepositoryError(t *testing.T) {
	repo := &fakeAdminRepository{userErr: errors.New("db error")}
	svc := service.NewAdminService(repo)

	_, err := svc.ListUsers(context.Background(), repository.AdminUserListQuery{})
	assertErrorCode(t, err, apperror.CodeInternal)
}

func TestAdminListResourcesPassesQuery(t *testing.T) {
	repo := &fakeAdminRepository{
		resourceResult: &repository.AdminResourceListResult{
			Items: []model.Resource{{Title: "机器学习入门"}},
			Total: 1,
		},
	}
	svc := service.NewAdminService(repo)

	result, err := svc.ListResources(context.Background(), repository.AdminResourceListQuery{
		Page:       2,
		PageSize:   10,
		Keyword:    "  AI  ",
		Type:       " course ",
		CategoryID: 3,
	})

	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("ListResources() total = %d, want 1", result.Total)
	}
	if repo.lastResourceQuery.Keyword != "AI" || repo.lastResourceQuery.Type != "course" {
		t.Fatalf("ListResources() filters were not trimmed")
	}
	if repo.lastResourceQuery.Page != 2 || repo.lastResourceQuery.PageSize != 10 || repo.lastResourceQuery.CategoryID != 3 {
		t.Fatalf("ListResources() query was not passed through")
	}
}

func TestAdminListResourcesMapsRepositoryError(t *testing.T) {
	repo := &fakeAdminRepository{resourceErr: errors.New("db error")}
	svc := service.NewAdminService(repo)

	_, err := svc.ListResources(context.Background(), repository.AdminResourceListQuery{})
	assertErrorCode(t, err, apperror.CodeInternal)
}
