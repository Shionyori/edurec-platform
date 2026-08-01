package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrRefreshTokenNotFound = errors.New("refresh token 不存在")

// RefreshTokenStore 管理 Redis 中的 Refresh Token
type RefreshTokenStore interface {
	Set(ctx context.Context, key string, userID uint, ttl time.Duration) error
	Get(ctx context.Context, key string) (uint, error)
	Delete(ctx context.Context, key string) error
}

type RedisRefreshTokenStore struct {
	rdb *redis.Client
}

func NewRedisRefreshTokenStore(rdb *redis.Client) *RedisRefreshTokenStore {
	return &RedisRefreshTokenStore{rdb: rdb}
}

func (s *RedisRefreshTokenStore) Set(ctx context.Context, key string, userID uint, ttl time.Duration) error {
	if err := s.rdb.Set(ctx, key, userID, ttl).Err(); err != nil {
		return fmt.Errorf("写入 Refresh Token 失败: %w", err)
	}
	return nil
}

func (s *RedisRefreshTokenStore) Get(ctx context.Context, key string) (uint, error) {
	value, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrRefreshTokenNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("读取 Refresh Token 失败: %w", err)
	}

	userID, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析 Refresh Token 用户 ID 失败: %w", err)
	}
	return uint(userID), nil
}

func (s *RedisRefreshTokenStore) Delete(ctx context.Context, key string) error {
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("删除 Refresh Token 失败: %w", err)
	}
	return nil
}
