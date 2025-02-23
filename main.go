package main

import (
	"context"
	"fmt"
	"log"

	"lauth/internal/boot"
	"lauth/pkg/redis"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := boot.InitConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置gin模式
	gin.SetMode(cfg.Server.Mode)

	// 初始化数据库连接
	db, err := boot.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying *sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// 初始化MongoDB连接
	mongodb, err := boot.InitMongo(&cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongodb.Close(context.Background())

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

	// 初始化仓储层
	repos := boot.InitRepositories(db, mongodb)

	// 初始化服务层
	services, err := boot.InitServices(cfg, repos, redisClient)
	if err != nil {
		log.Fatalf("Failed to init services: %v", err)
	}

	// 初始化审计组件
	auditComponents, err := boot.InitAudit(cfg, services.RoleService)
	if err != nil {
		log.Fatalf("Failed to init audit components: %v", err)
	}

	// 启动WebSocket服务器
	go auditComponents.WebSocketServer.Start()

	// 初始化HTTP处理器
	handlers := boot.InitHandlers(services, repos, auditComponents, cfg)

	// 初始化gin引擎和路由
	r := gin.Default()
	_ = boot.InitRouter(r, handlers, services.TokenService, services.IPLocationService, auditComponents, cfg)

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
