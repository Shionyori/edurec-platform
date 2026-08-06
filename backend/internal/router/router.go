package router

import (
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/handler"
	"github.com/Shionyori/edurec-platform/backend/internal/middleware"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	jwtutil "github.com/Shionyori/edurec-platform/backend/internal/util/jwt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func New(cfg *config.Config, db *gorm.DB, rdb *redis.Client) *gin.Engine {
	userRepo := repository.NewUserRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	resourceRepo := repository.NewResourceRepository(db)
	refreshTokenStore := repository.NewRedisRefreshTokenStore(rdb)
	jwtManager := jwtutil.NewManager(cfg.JWT.AccessSecret)

	accessTTL := time.Duration(cfg.JWT.AccessExpire) * time.Minute
	refreshTTL := time.Duration(cfg.JWT.RefreshExpire) * time.Minute

	authService := service.NewAuthService(userRepo, refreshTokenStore, jwtManager, accessTTL, refreshTTL)
	userService := service.NewUserService(userRepo)
	categoryService := service.NewCategoryService(categoryRepo)
	resourceService := service.NewResourceService(resourceRepo)
	authHandler := handler.NewAuthHandler(authService, userService)
	userHandler := handler.NewUserHandler(userService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	resourceHandler := handler.NewResourceHandler(resourceService)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	api := r.Group("/api/v1")
	api.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "healthy"})
	})

	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)

	protected := api.Group("")
	protected.Use(middleware.AuthRequired(jwtManager))
	protected.GET("/users/me", userHandler.Me)
	protected.PUT("/users/me", userHandler.UpdateMe)
	protected.GET("/categories", categoryHandler.List)
	protected.POST("/categories", middleware.AdminRequired(userRepo), categoryHandler.Create)
	protected.GET("/resources", resourceHandler.List)
	protected.GET("/resources/:id", resourceHandler.Detail)

	return r
}
