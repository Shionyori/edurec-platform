package service

import (
	"context"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
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
