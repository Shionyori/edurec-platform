package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	jwtutil "github.com/Shionyori/edurec-platform/backend/internal/util/jwt"
	"github.com/Shionyori/edurec-platform/backend/internal/util/refresh"
	"golang.org/x/crypto/bcrypt"
)

type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	DisplayName string
}

type LoginInput struct {
	Account  string
	Password string
}

type TokenPair struct {
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
	AccessExpiresIn int    `json:"expires_in"`
}

type AuthService struct {
	users        repository.UserRepository
	refreshToken repository.RefreshTokenStore
	jwt          *jwtutil.Manager
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

func NewAuthService(
	users repository.UserRepository,
	refreshToken repository.RefreshTokenStore,
	jwt *jwtutil.Manager,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		users:        users,
		refreshToken: refreshToken,
		jwt:          jwt,
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*model.User, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.DisplayName = strings.TrimSpace(input.DisplayName)

	if input.Username == "" || input.Email == "" || input.Password == "" {
		return nil, apperror.BadRequest("请求参数错误")
	}

	// 先做一次唯一性检查，便于给出更明确的业务提示
	if err := s.ensureUsernameAvailable(input.Username); err != nil {
		return nil, err
	}
	if err := s.ensureEmailAvailable(input.Email); err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	user := &model.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(passwordHash),
		DisplayName:  input.DisplayName,
	}
	if err := s.users.Create(user); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, apperror.Conflict("用户名或邮箱已存在")
		}
		return nil, apperror.Internal(err)
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*model.User, *TokenPair, error) {
	account := strings.TrimSpace(input.Account)
	if account == "" || input.Password == "" {
		return nil, nil, apperror.BadRequest("请求参数错误")
	}

	user, err := s.findUserByAccount(account)
	if err != nil {
		if apperror.IsBusinessError(err) {
			return nil, nil, err
		}
		return nil, nil, apperror.Internal(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, nil, apperror.Unauthorized("用户名或邮箱或密码错误")
	}

	pair, err := s.issueTokenPair(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, pair, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, apperror.Unauthorized("Refresh Token 无效或已过期")
	}

	key := refresh.Key(refreshToken)
	userID, err := s.refreshToken.Get(ctx, key)
	if errors.Is(err, repository.ErrRefreshTokenNotFound) {
		return nil, apperror.Unauthorized("Refresh Token 无效或已过期")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	pair, err := s.issueTokenPair(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 新 token 写入成功后再删除旧 token，避免刷新过程中短暂失效
	if err := s.refreshToken.Delete(ctx, key); err != nil {
		return nil, apperror.Internal(err)
	}

	return pair, nil
}

func (s *AuthService) ensureUsernameAvailable(username string) error {
	_, err := s.users.FindByUsername(username)
	if err == nil {
		return apperror.Conflict("用户名或邮箱已存在")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Internal(err)
	}
	return nil
}

func (s *AuthService) ensureEmailAvailable(email string) error {
	_, err := s.users.FindByEmail(email)
	if err == nil {
		return apperror.Conflict("用户名或邮箱已存在")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Internal(err)
	}
	return nil
}

func (s *AuthService) findUserByAccount(account string) (*model.User, error) {
	user, err := s.users.FindByUsername(account)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	user, err = s.users.FindByEmail(account)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.Unauthorized("用户名或邮箱或密码错误")
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) issueTokenPair(ctx context.Context, userID uint) (*TokenPair, error) {
	accessToken, err := s.jwt.GenerateAccessToken(userID, s.accessTTL)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	refreshToken, err := refresh.NewToken()
	if err != nil {
		return nil, apperror.Internal(err)
	}

	key := refresh.Key(refreshToken)
	if err := s.refreshToken.Set(ctx, key, userID, s.refreshTTL); err != nil {
		return nil, apperror.Internal(err)
	}

	return &TokenPair{
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		AccessExpiresIn: int(s.accessTTL.Seconds()),
	}, nil
}
