package main

import (
	"fmt"
	"log"
	"time"

	v1 "lauth/api/v1"
	"lauth/internal/model"
	"lauth/internal/repository"
	"lauth/internal/service"
	"lauth/pkg/config"
	"lauth/pkg/database"
	"lauth/pkg/engine"
	"lauth/pkg/middleware"
	"lauth/pkg/redis"
	"lauth/pkg/router"

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
	if err := db.AutoMigrate(
		&model.App{},
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
		&model.RolePermission{},
		&model.Rule{},
		&model.RuleCondition{},
	); err != nil {
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
	roleRepo := repository.NewRoleRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	ruleRepo := repository.NewRuleRepository(db)

	// 初始化Token服务
	tokenService := service.NewTokenService(
		redisClient,
		cfg.JWT.Secret,
		time.Duration(cfg.JWT.AccessTokenExpire)*time.Hour,
		time.Duration(cfg.JWT.RefreshTokenExpire)*time.Hour,
	)

	// 初始化规则引擎
	ruleParser := engine.NewParser()
	ruleExecutor := engine.NewExecutor()
	ruleCache := engine.NewCache(redisClient)
	ruleEngine := engine.NewEngine(ruleParser, ruleExecutor, ruleCache, ruleRepo)

	// 初始化服务层
	appService := service.NewAppService(appRepo)
	userService := service.NewUserService(userRepo, appRepo)
	ruleService := service.NewRuleService(ruleRepo, ruleEngine)
	authService := service.NewAuthService(userRepo, tokenService, ruleService)
	roleService := service.NewRoleService(roleRepo, permissionRepo)
	permissionService := service.NewPermissionService(permissionRepo, roleRepo)

	// 创建默认的gin引擎
	r := gin.Default()

	// 初始化认证中间件
	authMiddleware := middleware.NewAuthMiddleware(tokenService, cfg.Server.AuthEnabled)

	// 初始化处理器
	appHandler := v1.NewAppHandler(appService)
	userHandler := v1.NewUserHandler(userService)
	authHandler := v1.NewAuthHandler(authService)
	roleHandler := v1.NewRoleHandler(roleService)
	permissionHandler := v1.NewPermissionHandler(permissionService)
	ruleHandler := v1.NewRuleHandler(ruleService)

	// 初始化路由管理器
	router := router.NewRouter(
		r,
		authMiddleware,
		authHandler,
		appHandler,
		userHandler,
		permissionHandler,
		roleHandler,
		ruleHandler,
	)

	// 注册所有路由
	router.RegisterRoutes()

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
