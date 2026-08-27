package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	jwtutil "github.com/Shionyori/edurec-platform/backend/internal/util/jwt"
	"github.com/Shionyori/edurec-platform/backend/internal/util/refresh"
)

type fakeUserRepository struct {
	users  []*model.User
	nextID uint
}

func (f *fakeUserRepository) Create(user *model.User) error {
	for _, existing := range f.users {
		if existing.Username == user.Username || existing.Email == user.Email {
			return repository.ErrDuplicate
		}
	}
	f.nextID++
	user.ID = f.nextID
	f.users = append(f.users, user)
	return nil
}

func (f *fakeUserRepository) FindByUsername(username string) (*model.User, error) {
	for _, user := range f.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepository) FindByEmail(email string) (*model.User, error) {
	for _, user := range f.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepository) FindByIDs(ids []uint) ([]model.User, error) {
	byID := make(map[uint]bool, len(ids))
	for _, id := range ids {
		byID[id] = true
	}
	result := make([]model.User, 0)
	for _, u := range f.users {
		if byID[u.ID] {
			result = append(result, *u)
		}
	}
	return result, nil
}

func (f *fakeUserRepository) FindByID(id uint) (*model.User, error) {
	for _, user := range f.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepository) Update(user *model.User) error {
	for i, existing := range f.users {
		if existing.ID == user.ID {
			f.users[i] = user
			return nil
		}
	}
	return repository.ErrNotFound
}

func (f *fakeUserRepository) IsAdmin(userID uint) (bool, error) {
	return userID == 1, nil
}

type memoryRefreshTokenStore struct {
	tokens map[string]uint
}

func newMemoryRefreshTokenStore() *memoryRefreshTokenStore {
	return &memoryRefreshTokenStore{tokens: make(map[string]uint)}
}

func (s *memoryRefreshTokenStore) Set(_ context.Context, key string, userID uint, _ time.Duration) error {
	s.tokens[key] = userID
	return nil
}

func (s *memoryRefreshTokenStore) Get(_ context.Context, key string) (uint, error) {
	userID, ok := s.tokens[key]
	if !ok {
		return 0, repository.ErrRefreshTokenNotFound
	}
	return userID, nil
}

func (s *memoryRefreshTokenStore) Delete(_ context.Context, key string) error {
	delete(s.tokens, key)
	return nil
}

func newTestAuthService(users *fakeUserRepository, tokens *memoryRefreshTokenStore) *service.AuthService {
	return service.NewAuthService(
		users,
		tokens,
		jwtutil.NewManager("test-secret"),
		15*time.Minute,
		7*24*time.Hour,
	)
}

func TestRegisterCreatesUser(t *testing.T) {
	users := &fakeUserRepository{}
	auth := newTestAuthService(users, newMemoryRefreshTokenStore())

	user, err := auth.Register(context.Background(), service.RegisterInput{
		Username:    "zhangsan",
		Email:       "zhangsan@example.com",
		Password:    "password123",
		DisplayName: "张三",
	})

	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.ID != 1 {
		t.Fatalf("Register() user.ID = %d, want 1", user.ID)
	}
	if user.PasswordHash == "password123" {
		t.Fatal("Register() stored plaintext password")
	}
	if user.Username != "zhangsan" {
		t.Fatalf("Register() username = %q, want zhangsan", user.Username)
	}
}

func TestRegisterRejectsDuplicateUsername(t *testing.T) {
	users := &fakeUserRepository{}
	auth := newTestAuthService(users, newMemoryRefreshTokenStore())
	input := service.RegisterInput{
		Username: "zhangsan",
		Email:    "zhangsan@example.com",
		Password: "password123",
	}

	_, err := auth.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	_, err = auth.Register(context.Background(), input)
	assertErrorCode(t, err, apperror.CodeConflict)
}

func TestLoginReturnsTokenPair(t *testing.T) {
	users := &fakeUserRepository{}
	tokens := newMemoryRefreshTokenStore()
	auth := newTestAuthService(users, tokens)

	_, err := auth.Register(context.Background(), service.RegisterInput{
		Username: "zhangsan",
		Email:    "zhangsan@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	user, pair, err := auth.Login(context.Background(), service.LoginInput{
		Account:  "zhangsan",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("Login() returned empty token")
	}
	if pair.AccessExpiresIn != 900 {
		t.Fatalf("Login() expires_in = %d, want 900", pair.AccessExpiresIn)
	}
	if user.ID != 1 {
		t.Fatalf("Login() user.ID = %d, want 1", user.ID)
	}
	if _, ok := tokens.tokens[refresh.Key(pair.RefreshToken)]; !ok {
		t.Fatal("Login() did not store refresh token")
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	users := &fakeUserRepository{}
	auth := newTestAuthService(users, newMemoryRefreshTokenStore())

	_, err := auth.Register(context.Background(), service.RegisterInput{
		Username: "zhangsan",
		Email:    "zhangsan@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, _, err = auth.Login(context.Background(), service.LoginInput{
		Account:  "zhangsan",
		Password: "wrong-password",
	})

	assertErrorCode(t, err, apperror.CodeUnauthorized)
}

func TestRefreshRotatesToken(t *testing.T) {
	users := &fakeUserRepository{}
	tokens := newMemoryRefreshTokenStore()
	auth := newTestAuthService(users, tokens)

	_, err := auth.Register(context.Background(), service.RegisterInput{
		Username: "zhangsan",
		Email:    "zhangsan@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	_, pair, err := auth.Login(context.Background(), service.LoginInput{
		Account:  "zhangsan",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	oldKey := refresh.Key(pair.RefreshToken)
	newPair, err := auth.Refresh(context.Background(), pair.RefreshToken)

	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if pair.RefreshToken == newPair.RefreshToken {
		t.Fatal("Refresh() did not rotate refresh token")
	}
	if _, ok := tokens.tokens[oldKey]; ok {
		t.Fatal("Refresh() left old token in store")
	}
	if _, ok := tokens.tokens[refresh.Key(newPair.RefreshToken)]; !ok {
		t.Fatal("Refresh() did not store new token")
	}
}

func TestRefreshRejectsUnknownToken(t *testing.T) {
	users := &fakeUserRepository{}
	auth := newTestAuthService(users, newMemoryRefreshTokenStore())

	_, err := auth.Refresh(context.Background(), "unknown-token")
	assertErrorCode(t, err, apperror.CodeUnauthorized)
}

func assertErrorCode(t *testing.T, err error, wantCode int) {
	t.Helper()
	var appErr *apperror.Error
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.As(err, &appErr) {
		t.Fatalf("error type = %T, want *apperror.Error", err)
	}
	if appErr.Code != wantCode {
		t.Fatalf("error code = %d, want %d, message: %s", appErr.Code, wantCode, appErr.Message)
	}
}
