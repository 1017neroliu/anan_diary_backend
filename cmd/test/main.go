package main

import (
	"anan-diary/ad-backend/internal/handlers"
	"anan-diary/ad-backend/internal/repository"
	"anan-diary/ad-backend/pkg/database"
	"anan-diary/ad-backend/pkg/middleware"
	"anan-diary/ad-backend/pkg/response"
	"log"

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

	// 从环境变量获取数据库连接字符串
	dsn := "host=localhost user=postgres password=postgres dbname=anan_diary_test port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	// 初始化数据库
	if err := database.InitDBWithMigration(dsn); err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// 初始化Gin引擎
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// CORS 配置
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
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

	// 初始化仓储层
	userRepo := repository.NewUserRepository(database.GetDB())
	babyRepo := repository.NewBabyRepository(database.GetDB())
	memberRepo := repository.NewFamilyMemberRepository(database.GetDB())
	recordRepo := repository.NewGrowthRecordRepository(database.GetDB())
	milestoneRepo := repository.NewMilestoneRepository(database.GetDB())

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(userRepo, babyRepo, memberRepo, logger, database.GetDB())
	babyHandler := handlers.NewBabyHandler(babyRepo, memberRepo, logger, database.GetDB())
	recordHandler := handlers.NewGrowthRecordHandler(recordRepo, babyRepo, memberRepo, logger)
	milestoneHandler := handlers.NewMilestoneHandler(milestoneRepo, logger)

	// API路由
	api := router.Group("/api")
	{
		// 认证相关
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.GET("/me", authHandler.GetCurrentUser)
		}

		// 宝宝相关
		baby := api.Group("/baby")
		baby.Use(middleware.AuthMiddleware())
		{
			baby.GET("/current", babyHandler.GetCurrentBaby)
			baby.GET("", babyHandler.GetBabies)
			baby.POST("", babyHandler.CreateBaby)
			baby.GET("/:id", babyHandler.GetBabyByID)
			baby.DELETE("/:id", babyHandler.DeleteBaby)
		}

		// 成长记录相关
		records := api.Group("/growth-record")
		records.Use(middleware.AuthMiddleware())
		{
			records.GET("/timeline", recordHandler.GetTimeline)
			records.POST("", recordHandler.CreateGrowthRecord)
			records.DELETE("/:id", recordHandler.DeleteRecord)
		}

		// 里程碑相关
		milestones := api.Group("/milestone")
		milestones.Use(middleware.AuthMiddleware())
		{
			milestones.GET("", milestoneHandler.GetMilestones)
		}
	}

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status": "ok",
			"time":   gin.H{"datetime": "2006-01-02 15:04:05"},
		})
	})

	// 404处理
	router.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "接口不存在")
	})

	// 启动服务器
	port := ":8081"
	logger.Info("Starting test server", zap.String("port", port))
	if err := router.Run(port); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
