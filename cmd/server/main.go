package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"anan-diary/ad-backend/internal/config"
	"anan-diary/ad-backend/internal/handlers"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/internal/service"
	"anan-diary/ad-backend/pkg/database"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var logger *zap.Logger

func initLogger() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Sync()
}

func main() {
	// 初始化日志
	initLogger()

	// 设置时区为 Asia/Shanghai
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logger.Fatal("Failed to load Asia/Shanghai timezone", zap.Error(err))
	}
	os.Setenv("TZ", "Asia/Shanghai")
	time.Local = loc

	// 加载环境配置
	cfg := config.Load()

	// 注入 JWT 配置
	middleware.SetJWTConfig(cfg.JWTSecret, cfg.JWTExpire)

	// 初始化数据库
	if err := database.InitDBWithMigration(cfg.DSN); err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// 初始化Gin引擎
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// CORS 配置（使用白名单）
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" && cfg.AllowedOrigin(origin) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		// 开发环境允许所有 localhost 源
		if origin != "" && !cfg.IsProduction() {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
		}
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	router.Use(func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.Next()
	})

	// 预创建上传目录（router.Static 要求目录存在，否则 panic）
	for _, sub := range []string{"avatars", "records/images", "records/videos"} {
		if err := os.MkdirAll(filepath.Join(cfg.UploadDir, sub), 0755); err != nil {
			logger.Fatal("Failed to create upload directory", zap.String("dir", sub), zap.Error(err))
		}
	}

	// 配置静态文件服务（头像和记录媒体上传目录）
	router.Static("/uploads/avatars", filepath.Join(cfg.UploadDir, "avatars"))
	router.Static("/uploads/records/images", filepath.Join(cfg.UploadDir, "records/images"))
	router.Static("/uploads/records/videos", filepath.Join(cfg.UploadDir, "records/videos"))

	// 初始化仓储层
	userRepo := repository.NewUserRepository(database.GetDB())
	babyRepo := repository.NewBabyRepository(database.GetDB())
	memberRepo := repository.NewFamilyMemberRepository(database.GetDB())
	recordRepo := repository.NewGrowthRecordRepository(database.GetDB())
	milestoneRepo := repository.NewMilestoneRepository(database.GetDB())
	invitationRepo := repository.NewInvitationRepository(database.GetDB())
	approvalRepo := repository.NewApprovalRepository(database.GetDB())
	permissionRepo := repository.NewPermissionRepository(database.GetDB())
	likeRepo := repository.NewLikeRepository(database.GetDB())
	commentRepo := repository.NewCommentRepository(database.GetDB())
	statsRepo := repository.NewStatsRepository(database.GetDB())
	subscriptionRepo := repository.NewSubscriptionRepository(database.GetDB())

	// 初始化授权服务（依赖接口化的接口）
	// 使用 Go 类型断言确保 repository 结构满足 interfaces 定义的接口
	// （实际注入的是具体 struct 指针；Go 会隐式断言它实现了接口）
	authzService := service.NewAuthzService(permissionRepo, memberRepo, babyRepo)

	// 初始化业务服务
	_ = service.NewBabyService(babyRepo, memberRepo, authzService) // baby handler 后续阶段接入
	approvalSvc := service.NewApprovalService(approvalRepo, memberRepo, permissionRepo, authzService)
	permissionSvc := service.NewPermissionService(permissionRepo, approvalRepo, memberRepo, babyRepo, authzService)

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(userRepo, babyRepo, memberRepo, logger, database.GetDB())
	babyHandler := handlers.NewBabyHandler(babyRepo, memberRepo, authzService, logger, database.GetDB())
	familyMemberHandler := handlers.NewFamilyMemberHandler(memberRepo, authzService, logger)
	recordHandler := handlers.NewGrowthRecordHandler(recordRepo, babyRepo, memberRepo, authzService, logger)
	milestoneHandler := handlers.NewMilestoneHandler(milestoneRepo, logger)
	invitationHandler := handlers.NewInvitationHandler(invitationRepo, approvalRepo, permissionRepo, memberRepo, authzService, logger)
	approvalHandler := handlers.NewApprovalHandler(approvalSvc, logger)
	permissionHandler := handlers.NewPermissionHandler(permissionSvc, logger)
	likeHandler := handlers.NewLikeHandler(likeRepo, commentRepo, permissionRepo, authzService, logger)
	commentHandler := handlers.NewCommentHandler(commentRepo, permissionRepo, logger)
	statsHandler := handlers.NewStatsHandler(statsRepo, permissionRepo, logger)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionRepo, logger)
	exportHandler := handlers.NewExportHandler(recordRepo, babyRepo, memberRepo, logger)

	// API路由
	api := router.Group("/api")
	{
		// 认证相关（不需要认证）
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}
		// 获取当前用户（需要认证）
		api.GET("/auth/me", middleware.AuthMiddleware(), authHandler.GetCurrentUser)

		// 宝宝相关（需要认证）
		baby := api.Group("/baby")
		baby.Use(middleware.AuthMiddleware())
		{
			baby.GET("/current", babyHandler.GetCurrentBaby)
			baby.GET("", babyHandler.GetBabies)
			baby.POST("", babyHandler.CreateBaby)
			baby.GET("/:id", babyHandler.GetBabyByID)
			baby.DELETE("/:id", babyHandler.DeleteBaby)
			baby.POST("/:id/avatar", babyHandler.UploadAvatar)
		}

		// 邀请码相关（需要认证）
		invitations := api.Group("/invitation")
		invitations.Use(middleware.AuthMiddleware())
		{
			invitations.POST("/generate", invitationHandler.CreateInviteCode)
			invitations.GET("/:baby_id/codes", invitationHandler.GetInvitationCodes)
			invitations.GET("/:baby_id/code", invitationHandler.GetInviteCode)
			invitations.POST("/revoke/:baby_id", invitationHandler.RevokeInviteCode)
			invitations.POST("/join", invitationHandler.JoinFamily)
		}

		// 审批相关（需要认证）
		approvals := api.Group("/approval")
		approvals.Use(middleware.AuthMiddleware())
		{
			approvals.GET("/:baby_id", approvalHandler.GetApprovals)
			approvals.GET("/my", approvalHandler.GetMyApprovals)
			approvals.GET("/detail/:approval_id", approvalHandler.GetApprovalDetail)
			approvals.POST("/detail/:approval_id/approve", approvalHandler.ApproveMember)
			approvals.POST("/detail/:approval_id/reject", approvalHandler.RejectMember)
			approvals.POST("/detail/:approval_id/cancel", approvalHandler.CancelMyRequest)
		}

		// 权限相关（需要认证）
		permissions := api.Group("/permission")
		permissions.Use(middleware.AuthMiddleware())
		{
			permissions.GET("/:baby_id/members", permissionHandler.GetMyFamilyMembers)
			permissions.GET("/my/:baby_id", permissionHandler.GetMyPermissions)
			permissions.PUT("/:baby_id", permissionHandler.UpdatePermission)
			permissions.DELETE("/member/:user_id", permissionHandler.RemoveMember)
			permissions.GET("/check/:type", permissionHandler.CheckPermission)
			permissions.GET("/is-member/:baby_id", permissionHandler.IsMember)
		}

		// 点赞相关（需要认证）
		likes := api.Group("/like")
		likes.Use(middleware.AuthMiddleware())
		{
			likes.POST("/record/:id", likeHandler.LikeRecord)
			likes.DELETE("/record/:id", likeHandler.UnlikeRecord)
			likes.GET("/record/:id", likeHandler.GetRecordLikes)
			likes.GET("/record/:id/count", likeHandler.GetLikeCount)
			likes.GET("/has-liked/:id", likeHandler.HasLikedCheck)
			likes.GET("/user", likeHandler.GetUserLikes)
			likes.GET("/top", likeHandler.GetTopLikedRecords)
		}

		// 评论相关（需要认证）
		comments := api.Group("/comment")
		comments.Use(middleware.AuthMiddleware())
		{
			comments.POST("/record/:id", commentHandler.CreateComment)
			comments.GET("/record/:id", commentHandler.GetComments)
			comments.GET("/tree/:id", commentHandler.GetCommentTree)
			comments.GET("/:id", commentHandler.GetCommentDetail)
			comments.PUT("/:id", commentHandler.UpdateComment)
			comments.DELETE("/:id", commentHandler.DeleteComment)
			comments.GET("/user", commentHandler.GetUserComments)
			comments.GET("/record/:id/count", commentHandler.GetCommentCount)
		}

		// 统计相关（需要认证）
		stats := api.Group("/stats")
		stats.Use(middleware.AuthMiddleware())
		{
			stats.GET("/user", statsHandler.GetUserStats)
			stats.GET("/baby/:baby_id/members", statsHandler.GetMemberContributions)
			stats.GET("/baby/:baby_id/ranking", statsHandler.GetContributionRanking)
			stats.GET("/baby/:baby_id/highlight", statsHandler.GetMemberHighlight)
			stats.GET("/daily", statsHandler.GetDailyRecordCount)
			stats.GET("/weekly", statsHandler.GetWeeklyRecordCount)
			stats.GET("/monthly", statsHandler.GetMonthlyRecordCount)
		}

		// 导出相关（需要认证）
		export := api.Group("/export")
		export.Use(middleware.AuthMiddleware())
		{
			export.GET("/pdf", exportHandler.ExportToPDF)
			export.GET("/preview", exportHandler.GetExportPreview)
		}

		// 订阅相关（需要认证）
		subscriptions := api.Group("/subscription")
		subscriptions.Use(middleware.AuthMiddleware())
		{
			subscriptions.GET("/detail", subscriptionHandler.GetSubscriptionDetail)
			subscriptions.GET("/check", subscriptionHandler.CheckSubscriptionStatus)
			subscriptions.GET("/quota", subscriptionHandler.GetSubscriptionQuota)
			subscriptions.GET("/multi-baby", subscriptionHandler.HasMultiBabySupport)
			subscriptions.GET("/pdf-export", subscriptionHandler.HasPDFExport)
			subscriptions.GET("/plans", subscriptionHandler.GetSubscriptionPlans)
			subscriptions.POST("/payment", subscriptionHandler.RecordPayment)
			subscriptions.POST("/renew", subscriptionHandler.RenewSubscription)
			subscriptions.POST("/cancel", subscriptionHandler.CancelSubscription)
			subscriptions.GET("/usage", subscriptionHandler.GetUsageSummary)
		}

		// 里程碑相关（需要认证）
		milestones := api.Group("/milestone")
		milestones.Use(middleware.AuthMiddleware())
		{
			milestones.GET("", milestoneHandler.GetMilestones)
			milestones.GET("/check", milestoneHandler.CheckMilestones)
			// TODO: 修复或移除这些方法
			// milestones.POST("/baby/:baby_id/custom", milestoneHandler.CreateCustomMilestone)
			// milestones.GET("/baby/:baby_id/reached", milestoneHandler.GetBabyMilestones)
			// milestones.GET("/baby/:baby_id/upcoming", milestoneHandler.GetUpcomingMilestones)
			// milestones.GET("/templates", milestoneHandler.GetMilestoneTemplates)
		}

		// 家庭成员相关（需要认证）
		members := api.Group("/family-member")
		members.Use(middleware.AuthMiddleware())
		{
			members.GET("/:id", familyMemberHandler.GetFamilyMembers)
			members.POST("", familyMemberHandler.AddFamilyMember)
		}

		// 成长记录相关（需要认证）
		records := api.Group("/growth-record")
		records.Use(middleware.AuthMiddleware())
		{
			records.GET("/timeline", recordHandler.GetTimeline)
			records.POST("", recordHandler.CreateGrowthRecord)
			records.DELETE("/:id", recordHandler.DeleteRecord)
			records.POST("/media", recordHandler.UploadMedia)
			records.GET("/:id/media", recordHandler.GetRecordMedia)
		}
	}

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status": "ok",
			"time":   gin.H{"datetime": time.Now().Format("2006-01-02 15:04:05")},
		})
	})

	// 404处理
	router.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "接口不存在")
	})

	// 启动服务器
	addr := "0.0.0.0:" + cfg.Port
	logger.Info("Starting server", zap.String("addr", addr), zap.String("env", cfg.Env))
	if err := router.Run(addr); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
