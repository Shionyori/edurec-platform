package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

type ResourceService struct {
	resources repository.ResourceRepository
}

func NewResourceService(resources repository.ResourceRepository) *ResourceService {
	return &ResourceService{resources: resources}
}

func (s *ResourceService) List(ctx context.Context, query repository.ResourceListQuery) (*repository.ResourceListResult, error) {
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Type = strings.TrimSpace(query.Type)
	for i := range query.Tags {
		query.Tags[i] = strings.TrimSpace(query.Tags[i])
	}
	if query.Sort == "" {
		query.Sort = "latest"
	}

	result, err := s.resources.List(query)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return result, nil
}

func (s *ResourceService) GetByID(ctx context.Context, id uint) (*model.Resource, error) {
	resource, err := s.resources.FindByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.NotFound("资源不存在")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return resource, nil
}
