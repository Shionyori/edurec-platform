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

// maxUserInterests 单用户最多保存的兴趣分类数，避免冷启动推荐被过长兴趣列表稀释。
const maxUserInterests = 20

type UserService struct {
	users repository.UserRepository
}

func NewUserService(users repository.UserRepository) *UserService {
	return &UserService{users: users}
}

func (s *UserService) GetByID(ctx context.Context, userID uint) (*model.User, error) {
	user, err := s.users.FindByID(userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.NotFound("用户不存在")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return user, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID uint, displayName *string, avatarURL *string) (*model.User, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if displayName != nil {
		user.DisplayName = *displayName
	}
	if avatarURL != nil {
		user.AvatarURL = *avatarURL
	}
	if err := s.users.Update(user); err != nil {
		return nil, apperror.Internal(err)
	}
	return user, nil
}

func (s *UserService) IsAdmin(ctx context.Context, userID uint) (bool, error) {
	isAdmin, err := s.users.IsAdmin(userID)
	if err != nil {
		return false, apperror.Internal(err)
	}
	return isAdmin, nil
}

// UpdateInterests 覆盖式保存用户的冷启动兴趣（分类 ID 列表）：去重、丢弃 0、上限截断。
func (s *UserService) UpdateInterests(ctx context.Context, userID uint, categoryIDs []uint) (*model.User, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	seen := make(map[uint]struct{}, len(categoryIDs))
	unique := make([]uint, 0, len(categoryIDs))
	for _, id := range categoryIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
		if len(unique) >= maxUserInterests {
			break
		}
	}

	raw, err := json.Marshal(unique)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	user.Interests = string(raw)
	if err := s.users.Update(user); err != nil {
		return nil, apperror.Internal(err)
	}
	return user, nil
}

// parseUserInterests 解析用户兴趣 JSON 数组；为空或损坏时返回 nil。
func parseUserInterests(raw string) []uint {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	return ids
}
