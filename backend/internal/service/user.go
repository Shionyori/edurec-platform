package service

import (
	"context"
	"errors"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

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
