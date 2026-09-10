package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// onlineSearcher 在线搜索 B 站并落库的依赖；*BilibiliOnlineService 实现了它。
// 抽象成接口以便在单测里注入 fake，避免真实 os/exec 调用。
type onlineSearcher interface {
	SearchAndImport(keyword string, page int) ([]model.Resource, bool, error)
}

type ResourceService struct {
	resources repository.ResourceRepository
	online    onlineSearcher
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

// SetBilibiliOnline 注入在线 B 站采集服务（nil 表示禁用在线搜索爬取，测试用）
func (s *ResourceService) SetBilibiliOnline(online onlineSearcher) {
	s.online = online
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

	// 在线翻页：本地结果耗尽后，前端用 online_page 逐页拉 B 站，返回本次新导入资源
	pureKeywordSearch := query.CategoryID == 0 && query.Type == "" && len(query.Tags) == 0
	if s.online != nil && query.OnlinePage > 0 && query.Keyword != "" && pureKeywordSearch {
		resources, hasMore, err := s.online.SearchAndImport(query.Keyword, query.OnlinePage)
		if err != nil {
			slog.Warn("B站搜索爬取失败", "keyword", query.Keyword, "online_page", query.OnlinePage, "error", err)
			return &repository.ResourceListResult{Items: []model.Resource{}, HasMore: false}, nil
		}
		return &repository.ResourceListResult{Items: resources, HasMore: hasMore}, nil
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

func (s *ResourceService) Delete(ctx context.Context, id uint) error {
	err := s.resources.Delete(id)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.NotFound("资源不存在")
	}
	if err != nil {
		return apperror.Internal(err)
	}
	return nil
}
