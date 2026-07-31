package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	var logger *slog.Logger
	switch cfg.Server.Mode {
	case "release":
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	slog.SetDefault(logger)

	// 初始化 MySQL
	db, err := database.InitMySQL(cfg.Database)
	if err != nil {
		slog.Error("MySQL 初始化失败", "error", err)
		os.Exit(1)
	}

	// 初始化 Redis
	rdb, err := database.InitRedis(cfg.Redis)
	if err != nil {
		slog.Error("Redis 初始化失败", "error", err)
		os.Exit(1)
	}
	_, _ = db, rdb // 后续传递给 handler/service 使用

	// 设置 Gin 运行模式
	gin.SetMode(cfg.Server.Mode)

	// 创建路由
	r := gin.New()

	// 全局中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// 健康检查
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    0,
			"message": "ok",
			"data": gin.H{
				"status": "healthy",
			},
		})
	})

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("服务器启动", "port", cfg.Server.Port, "mode", cfg.Server.Mode)
	if err := r.Run(addr); err != nil {
		slog.Error("服务器启动失败", "error", err)
		os.Exit(1)
	}
}
