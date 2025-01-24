package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	v1 "lauth/api/v1"
	"lauth/internal/model"
	"lauth/internal/repository"
	"lauth/internal/service"
	"lauth/pkg/config"
	"lauth/pkg/database"
	"lauth/pkg/middleware"
	"lauth/pkg/redis"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置gin模式
	gin.SetMode(cfg.Server.Mode)

	// 连接数据库
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Printf("Successfully connected to database")

	// 自动迁移数据库表
	if err := db.AutoMigrate(&model.App{}, &model.User{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 获取底层sqlDB以便在程序结束时关闭
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying *sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// 初始化Redis客户端
	redisClient, err := redis.NewClient(&redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Printf("Successfully connected to Redis")

	// 初始化仓储层
	appRepo := repository.NewAppRepository(db)
	userRepo := repository.NewUserRepository(db)

	// 初始化Token服务
	tokenService := service.NewTokenService(
		redisClient,
		cfg.JWT.Secret,
		time.Duration(cfg.JWT.AccessTokenExpire)*time.Hour,
		time.Duration(cfg.JWT.RefreshTokenExpire)*time.Hour,
	)

	// 初始化服务层
	appService := service.NewAppService(appRepo)
	userService := service.NewUserService(userRepo, appRepo)
	authService := service.NewAuthService(userRepo, tokenService)

	// 创建默认的gin引擎
	r := gin.Default()

	// 基础健康检查路由
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "lauth",
		})
	})

	// 初始化认证中间件
	authMiddleware := middleware.NewAuthMiddleware(tokenService, cfg.Server.AuthEnabled)

	// 注册API路由
	api := r.Group("/api/v1")
	{
		appHandler := v1.NewAppHandler(appService)
		appHandler.Register(api, authMiddleware)

		userHandler := v1.NewUserHandler(userService)
		userHandler.Register(api, authMiddleware)

		authHandler := v1.NewAuthHandler(authService)
		authHandler.Register(api)

	}

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
