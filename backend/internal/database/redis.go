package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// InitRedis 初始化 Redis 客户端并验证连接
func InitRedis(cfg config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}

	slog.Info("Redis 连接成功", "host", cfg.Host, "port", cfg.Port)
	return rdb, nil
}
