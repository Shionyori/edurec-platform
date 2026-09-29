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

// SetFavorite 收藏/取消收藏（幂等）。
// 收藏 = 新增一条 favorite 行为（已存在则不重复插入）；取消 = 删除该用户对该资源的 favorite 行为。
// favorite 是强正反馈，会随 user_behaviors 一并导出给 engine 训练。
func (s *UserBehaviorService) SetFavorite(ctx context.Context, userID, resourceID uint, favorite bool) error {
	if userID == 0 || resourceID == 0 {
		return apperror.BadRequest("请求参数错误")
	}
	_, err := s.resources.FindByID(resourceID)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.NotFound("资源不存在")
	}
	if err != nil {
		return apperror.Internal(err)
	}

	if favorite {
		exists, err := s.behaviors.Exists(userID, resourceID, "favorite")
		if err != nil {
			return apperror.Internal(err)
		}
		if exists {
			return nil
		}
		if err := s.behaviors.Create(&model.UserBehavior{
			UserID: userID, ResourceID: resourceID, Action: "favorite",
		}); err != nil {
			return apperror.Internal(err)
		}
		return nil
	}

	if err := s.behaviors.DeleteByUserResourceAction(userID, resourceID, "favorite"); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

// IsFavorite 判断用户是否已收藏某资源
func (s *UserBehaviorService) IsFavorite(ctx context.Context, userID, resourceID uint) (bool, error) {
	if userID == 0 || resourceID == 0 {
		return false, nil
	}
	exists, err := s.behaviors.Exists(userID, resourceID, "favorite")
	if err != nil {
		return false, apperror.Internal(err)
	}
	return exists, nil
}

// ListFavorites 分页返回用户收藏的资源（按最近收藏倒序），供「我的收藏」页使用。
func (s *UserBehaviorService) ListFavorites(ctx context.Context, userID uint, page, pageSize int) ([]model.Resource, int64, error) {
	ids, total, err := s.behaviors.ListFavoriteResourceIDs(userID, page, pageSize)
	if err != nil {
		return nil, 0, apperror.Internal(err)
	}
	if len(ids) == 0 {
		return []model.Resource{}, total, nil
	}

	resources, err := s.resources.FindByIDs(ids)
	if err != nil {
		return nil, 0, apperror.Internal(err)
	}
	// FindByIDs 不保证顺序，按收藏顺序重排
	byID := make(map[uint]model.Resource, len(resources))
	for i := range resources {
		byID[resources[i].ID] = resources[i]
	}
	ordered := make([]model.Resource, 0, len(ids))
	for _, id := range ids {
		if r, ok := byID[id]; ok {
			ordered = append(ordered, r)
		}
	}
	return ordered, total, nil
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
