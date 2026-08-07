package service

import (
	"context"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

type AdminService struct {
	admins repository.AdminRepository
}

func NewAdminService(admins repository.AdminRepository) *AdminService {
	return &AdminService{admins: admins}
}

func (s *AdminService) ListUsers(ctx context.Context, query repository.AdminUserListQuery) (*repository.AdminUserListResult, error) {
	query.Keyword = strings.TrimSpace(query.Keyword)
	result, err := s.admins.ListUsers(query)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return result, nil
}

func (s *AdminService) ListResources(ctx context.Context, query repository.AdminResourceListQuery) (*repository.AdminResourceListResult, error) {
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Type = strings.TrimSpace(query.Type)
	result, err := s.admins.ListResources(query)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return result, nil
}
