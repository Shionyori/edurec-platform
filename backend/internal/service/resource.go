package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

type ResourceService struct {
	resources repository.ResourceRepository
}

type CreateResourceInput struct {
	Title       string
	Description string
	CoverURL    string
	Type        string
	CategoryID  uint
	Tags        []string
	Metadata    map[string]any
	Author      string
	SourceURL   string
}

type UpdateResourceInput struct {
	ID          uint
	Title       *string
	Description *string
	CoverURL    *string
	Type        *string
	CategoryID  *uint
	Tags        *[]string
	Metadata    *map[string]any
	Author      *string
	SourceURL   *string
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

func (s *ResourceService) Create(ctx context.Context, input CreateResourceInput) (*model.Resource, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.CoverURL = strings.TrimSpace(input.CoverURL)
	input.Type = strings.TrimSpace(input.Type)
	input.Author = strings.TrimSpace(input.Author)
	input.SourceURL = strings.TrimSpace(input.SourceURL)

	if input.Title == "" || input.Description == "" || input.Type == "" || input.CategoryID == 0 {
		return nil, apperror.BadRequest("请求参数错误")
	}
	switch input.Type {
	case "course", "article", "video":
	default:
		return nil, apperror.BadRequest("请求参数错误")
	}

	tags := []byte("[]")
	if input.Tags != nil {
		var err error
		tags, err = json.Marshal(input.Tags)
		if err != nil {
			return nil, apperror.BadRequest("请求参数错误")
		}
	}
	metadata := []byte("{}")
	if input.Metadata != nil {
		var err error
		metadata, err = json.Marshal(input.Metadata)
		if err != nil {
			return nil, apperror.BadRequest("请求参数错误")
		}
	}

	resource := &model.Resource{
		Title:       input.Title,
		Description: input.Description,
		CoverURL:    input.CoverURL,
		Type:        input.Type,
		CategoryID:  input.CategoryID,
		Tags:        string(tags),
		Metadata:    string(metadata),
		Author:      input.Author,
		SourceURL:   input.SourceURL,
	}
	if err := s.resources.Create(resource); err != nil {
		return nil, apperror.Internal(err)
	}
	return resource, nil
}

func (s *ResourceService) Update(ctx context.Context, input UpdateResourceInput) (*model.Resource, error) {
	resource, err := s.resources.FindByID(input.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.NotFound("资源不存在")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	if input.Title != nil {
		value := strings.TrimSpace(*input.Title)
		if value == "" {
			return nil, apperror.BadRequest("请求参数错误")
		}
		resource.Title = value
	}
	if input.Description != nil {
		value := strings.TrimSpace(*input.Description)
		if value == "" {
			return nil, apperror.BadRequest("请求参数错误")
		}
		resource.Description = value
	}
	if input.CoverURL != nil {
		resource.CoverURL = strings.TrimSpace(*input.CoverURL)
	}
	if input.Type != nil {
		value := strings.TrimSpace(*input.Type)
		switch value {
		case "course", "article", "video":
			resource.Type = value
		default:
			return nil, apperror.BadRequest("请求参数错误")
		}
	}
	if input.CategoryID != nil {
		if *input.CategoryID == 0 {
			return nil, apperror.BadRequest("请求参数错误")
		}
		resource.CategoryID = *input.CategoryID
	}
	if input.Tags != nil {
		tags, err := json.Marshal(input.Tags)
		if err != nil {
			return nil, apperror.BadRequest("请求参数错误")
		}
		resource.Tags = string(tags)
	}
	if input.Metadata != nil {
		metadata, err := json.Marshal(input.Metadata)
		if err != nil {
			return nil, apperror.BadRequest("请求参数错误")
		}
		resource.Metadata = string(metadata)
	}
	if input.Author != nil {
		resource.Author = strings.TrimSpace(*input.Author)
	}
	if input.SourceURL != nil {
		resource.SourceURL = strings.TrimSpace(*input.SourceURL)
	}

	if err := s.resources.Update(resource); err != nil {
		return nil, apperror.Internal(err)
	}
	return resource, nil
}
