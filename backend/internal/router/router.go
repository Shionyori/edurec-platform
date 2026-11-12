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
	behaviorRepo := repository.NewUserBehaviorRepository(db)
	impressionRepo := repository.NewResourceImpressionRepository(db)
	ratingRepo := repository.NewRatingRepository(db)
	recommendationRepo := repository.NewRecommendationRepository(db)
	recommendationRunRepo := repository.NewRecommendationRunRepository(db)
	statsRepo := repository.NewStatsRepository(db)
	adminRepo := repository.NewAdminRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	knowledgeRepo := repository.NewKnowledgeRepository(db)
	learningRepo := repository.NewLearningRepository(db)
	refreshTokenStore := repository.NewRedisRefreshTokenStore(rdb)
	jwtManager := jwtutil.NewManager(cfg.JWT.AccessSecret)

	accessTTL := time.Duration(cfg.JWT.AccessExpire) * time.Minute
	refreshTTL := time.Duration(cfg.JWT.RefreshExpire) * time.Minute

	authService := service.NewAuthService(userRepo, refreshTokenStore, jwtManager, accessTTL, refreshTTL)
	userService := service.NewUserService(userRepo)
	categoryService := service.NewCategoryService(categoryRepo)
	resourceService := service.NewResourceService(resourceRepo)
	behaviorService := service.NewUserBehaviorService(behaviorRepo, resourceRepo)
	impressionService := service.NewResourceImpressionService(impressionRepo, resourceRepo)
	ratingService := service.NewRatingService(ratingRepo, resourceRepo)

	// 在线 B 站采集：搜索落库 + 评论抓取（复用离线导入服务，filePath 仅离线导入用）
	// 白名单与内容归类规则一并传入：在线抓取与离线导入共用同一套过滤与类型判定
	bilibiliImportService := service.NewCrawlImportService(
		resourceRepo, categoryRepo, "",
		cfg.Bilibili.AllowedTypenamesOrDefault(),
		cfg.ContentRules,
	)
	bilibiliOnlineService := service.NewBilibiliOnlineService(bilibiliImportService, cfg.Bilibili)
	resourceService.SetBilibiliOnline(bilibiliOnlineService)
	disciplineCrawler := service.NewDisciplineCrawlService(knowledgeRepo, bilibiliOnlineService, 30*time.Minute)
	commentService := service.NewCommentService(commentRepo, bilibiliOnlineService, cfg.Bilibili.CommentLimit)
	recommendationService := service.NewRecommendationService(recommendationRepo, resourceRepo, userRepo)
	knowledgeService := service.NewKnowledgeService(knowledgeRepo)
	disciplineService := service.NewDisciplineService(knowledgeRepo)
	learningService := service.NewLearningService(knowledgeRepo, learningRepo)
	// 概念生成：只通过 HTTP 调 engine 的 kg_server.py，平台不执行 Python
	conceptRunner := service.NewHTTPConceptRunner(service.HTTPConceptRunnerConfig{
		BaseURL:          cfg.Knowledge.GeneratorURL,
		Timeout:          time.Duration(cfg.Knowledge.GeneratorTimeoutSeconds) * time.Second,
		Token:            cfg.Knowledge.GeneratorToken,
		MaxConcurrency:   cfg.Knowledge.GeneratorMaxConcurrency,
		FailureThreshold: cfg.Knowledge.GeneratorFailureThreshold,
		CircuitCooldown:  time.Duration(cfg.Knowledge.GeneratorCircuitCooldownSeconds) * time.Second,
	})
	conceptGenerator := service.NewConceptGenerator(knowledgeRepo, conceptRunner, 10*time.Minute)
	recommendationImportService := service.NewRecommendationImportService(
		recommendationRepo, recommendationRunRepo, userRepo, resourceRepo, cfg.Engine.RecommendationsFile,
	)
	recommendationAdminService := service.NewRecommendationAdminService(
		recommendationRunRepo, statsRepo,
	)
	adminService := service.NewAdminService(adminRepo)
	authHandler := handler.NewAuthHandler(authService, userService)
	userHandler := handler.NewUserHandler(userService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	resourceHandler := handler.NewResourceHandler(resourceService, behaviorService)
	behaviorHandler := handler.NewUserBehaviorHandler(behaviorService)
	impressionHandler := handler.NewResourceImpressionHandler(impressionService)
	ratingHandler := handler.NewRatingHandler(ratingService)
	commentHandler := handler.NewCommentHandler(resourceService, commentService)
	recommendationHandler := handler.NewRecommendationHandler(
		recommendationService, recommendationImportService, recommendationAdminService,
	)
	knowledgeHandler := handler.NewKnowledgeHandler(knowledgeService)
	learningHandler := handler.NewLearningHandler(learningService)
	generationHandler := handler.NewGenerationHandler(conceptGenerator, knowledgeService)
	disciplineHandler := handler.NewDisciplineHandler(disciplineService, disciplineCrawler)
	adminHandler := handler.NewAdminHandler(adminService)

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
	protected.PUT("/users/me/interests", userHandler.UpdateInterests)
	protected.GET("/categories", categoryHandler.List)
	protected.POST("/categories", middleware.AdminRequired(userRepo), categoryHandler.Create)
	protected.GET("/resources", resourceHandler.List)
	protected.GET("/resources/:id", resourceHandler.Detail)
	protected.POST("/resources", middleware.AdminRequired(userRepo), resourceHandler.Create)
	protected.PUT("/resources/:id", middleware.AdminRequired(userRepo), resourceHandler.Update)
	protected.DELETE("/resources/:id", middleware.AdminRequired(userRepo), resourceHandler.Delete)
	protected.POST("/resources/:id/behaviors", behaviorHandler.Record)
	protected.POST("/resources/:id/favorite", behaviorHandler.Favorite)
	protected.DELETE("/resources/:id/favorite", behaviorHandler.Unfavorite)
	protected.POST("/impressions", impressionHandler.Record)
	protected.GET("/users/me/behaviors", behaviorHandler.List)
	protected.GET("/users/me/favorites", behaviorHandler.Favorites)
	protected.GET("/resources/:id/ratings", ratingHandler.List)
	protected.POST("/resources/:id/ratings", ratingHandler.Upsert)
	protected.GET("/resources/:id/comments", commentHandler.List)
	protected.GET("/recommendations", recommendationHandler.Get)
	// 知识点导航：列表 / 图谱详情 / 知识点下的资源
	protected.GET("/knowledge-points", knowledgeHandler.List)
	protected.GET("/knowledge-points/:code", knowledgeHandler.Detail)
	protected.GET("/knowledge-points/:code/resources", knowledgeHandler.Resources)
	// 学科导航：主页面学科 -> 学科页 -> 知识点/资源；无资源时自动爬取
	protected.GET("/disciplines", disciplineHandler.List)
	protected.GET("/disciplines/:code", disciplineHandler.Detail)
	protected.POST("/disciplines/:code/crawl", disciplineHandler.Crawl)
	protected.GET("/disciplines/:code/crawl", disciplineHandler.CrawlStatus)
	// 概念生成（engine kg_server.py）：搜索栏实时触发 + 状态轮询
	protected.POST("/knowledge-generation", generationHandler.StartByKeyword)
	protected.POST("/knowledge-points/:code/generate", generationHandler.Start)
	protected.GET("/knowledge-points/:code/generate", generationHandler.Status)
	// 自适应学习：问卷 -> 知识画像 -> 先修路径 -> 逐节解锁
	protected.GET("/learning-profile", learningHandler.Profile)
	protected.POST("/learning-profile/questionnaire", learningHandler.SubmitQuestionnaire)
	protected.POST("/learning-paths", learningHandler.CreatePath)
	protected.GET("/learning-paths/:id", learningHandler.GetPath)
	protected.POST("/learning-paths/:id/complete", learningHandler.CompleteStep)
	protected.GET("/learning-paths/:id/resources", learningHandler.Resources)
	protected.GET("/admin/users", middleware.AdminRequired(userRepo), adminHandler.ListUsers)
	protected.GET("/admin/resources", middleware.AdminRequired(userRepo), adminHandler.ListResources)
	protected.POST("/admin/recommendations/import", middleware.AdminRequired(userRepo), recommendationHandler.Import)
	protected.GET("/admin/recommendation-runs", middleware.AdminRequired(userRepo), recommendationHandler.ListRuns)
	protected.GET("/admin/recommendation-stats", middleware.AdminRequired(userRepo), recommendationHandler.Stats)

	return r
}
