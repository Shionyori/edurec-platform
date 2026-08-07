package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

type UserBehaviorService struct {
	behaviors repository.UserBehaviorRepository
	resources repository.ResourceRepository
}

func NewUserBehaviorService(behaviors repository.UserBehaviorRepository, resources repository.ResourceRepository) *UserBehaviorService {
	return &UserBehaviorService{behaviors: behaviors, resources: resources}
}

func (s *UserBehaviorService) Record(ctx context.Context, userID uint, resourceID uint, action string) error {
	action = strings.TrimSpace(action)
	if userID == 0 || resourceID == 0 || action == "" {
		return apperror.BadRequest("请求参数错误")
	}
	switch action {
	case "view", "click", "favorite":
	default:
		return apperror.BadRequest("请求参数错误")
	}

	_, err := s.resources.FindByID(resourceID)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.NotFound("资源不存在")
	}
	if err != nil {
		return apperror.Internal(err)
	}

	behavior := &model.UserBehavior{
		UserID:     userID,
		ResourceID: resourceID,
		Action:     action,
	}
	if err := s.behaviors.Create(behavior); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func (s *UserBehaviorService) List(ctx context.Context, query repository.UserBehaviorListQuery) (*repository.UserBehaviorListResult, error) {
	query.Action = strings.TrimSpace(query.Action)
	if query.Action != "" {
		switch query.Action {
		case "view", "click", "favorite":
		default:
			return nil, apperror.BadRequest("请求参数错误")
		}
	}

	result, err := s.behaviors.List(query)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return result, nil
}
