package main

import (
	"context"
	"fmt"
	"time"

	"lauth/internal/boot"
	"lauth/internal/plugin/types"
	"lauth/pkg/copyright"
	"lauth/pkg/logger"
	"lauth/pkg/redis"
	"lauth/pkg/version"

	"github.com/gin-gonic/gin"
)

// checkFatalErr 用于统一处理错误检查并中断流程。
func checkFatalErr(err error, message string) {
	if err != nil {
		logger.Fatal("%s: %v", message, err)
	}
}

func main() {
	// 设置构建时间（Build Time）
	version.BuildTime = time.Now().Format(time.RFC3339)

	// 加载配置文件（Configuration）
	cfg, err := boot.InitConfig("config/config.yaml")
	checkFatalErr(err, "Failed to load config")

	// 根据配置设置 Gin 的运行模式（Gin Mode）
	gin.SetMode(cfg.Server.Mode)

	// 初始化数据库连接（PostgreSQL）
	db, err := boot.InitDB(&cfg.Database)
	checkFatalErr(err, "Failed to connect to database")

	sqlDB, err := db.DB()
	checkFatalErr(err, "Failed to get underlying *sql.DB")
	defer sqlDB.Close()

	// 初始化 MongoDB 连接（MongoDB）
	mongodb, err := boot.InitMongo(&cfg.MongoDB)
	checkFatalErr(err, "Failed to connect to MongoDB")
	defer mongodb.Close(context.Background())

	// 初始化 Redis 客户端（Redis）
	redisClient, err := redis.NewClient(&redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	checkFatalErr(err, "Failed to connect to Redis")

	// 初始化仓储层（Repositories）
	repos := boot.InitRepositories(db, mongodb)

	// 初始化服务层（Services）
	services, err := boot.InitServices(cfg, repos, redisClient, db)
	checkFatalErr(err, "Failed to init services")

	// 初始化超级管理员（如果不存在）
	adminUser, adminPass, isNewAdmin, err := services.SuperAdminService.CheckAndInitSuperAdmin(context.Background())
	checkFatalErr(err, "Failed to init super admin")

	// 初始化审计组件（Audit Components）
	auditComponents, err := boot.InitAudit(cfg, services.RoleService)
	checkFatalErr(err, "Failed to init audit components")

	// 启动 WebSocket 服务器，用于审计（WebSocket Server）
	go auditComponents.WebSocketServer.Start()

	// 初始化 HTTP 处理器（Handlers）
	handlers := boot.InitHandlers(services, repos, auditComponents, cfg)

	// 初始化 Gin 引擎和路由（Router）
	r := gin.Default()
	_ = boot.InitRouter(r, handlers, services.TokenService, services.IPLocationService, auditComponents, cfg, services)

	// 统计相关系统状态信息（System Status）
	userCount, _ := repos.UserRepo.Count(context.Background())

	// 从数据库获取应用列表
	dbApps, _, _ := services.AppService.ListApps(context.Background(), 0, 100)
	appsIDs := make([]string, len(dbApps))
	for i, app := range dbApps {
		appsIDs[i] = app.ID
	}

	// 获取审计日志相关信息，仅用于统计日志数量
	var totalLogs int64
	auditApps, _ := auditComponents.Reader.ListApps()
	for _, appID := range auditApps {
		count, err := auditComponents.Reader.GetLogCount(appID)
		if err == nil {
			totalLogs += count
		}
	}

	plugins := types.GetRegisteredPlugins()
	pluginNames := make([]string, len(plugins))
	pluginPaths := make([]copyright.PluginEndpoint, 0)

	for i, p := range plugins {
		instance := p.Factory()
		pluginNames[i] = instance.Name()

		// 若插件可路由，则获取路由及 API 信息
		if routable, ok := instance.(types.Routable); ok {
			authRoutes := routable.GetRoutesRequireAuth()
			needsAuth := false
			for _, route := range authRoutes {
				if route == "*" {
					needsAuth = true
					break
				}
			}

			apis := routable.GetAPIInfo()
			routes := make([]string, 0, len(apis))
			for _, api := range apis {
				routes = append(routes, fmt.Sprintf("%s %s", api.Method, api.Path))
			}

			pluginPaths = append(pluginPaths, copyright.PluginEndpoint{
				Name: instance.Name(),
				APIs: routes,
				Auth: needsAuth,
			})
		}
	}

	// 显示版权信息（Copyright）
	status := copyright.SystemStatus{
		Version:        version.GetVersion(),
		RedisStatus:    redisClient != nil,
		MongoDBStatus:  mongodb != nil,
		PostgresStatus: db != nil,
		Plugins:        pluginNames,
		PluginPaths:    pluginPaths,
		Apps:           appsIDs,
		UserCount:      userCount,
		LogCount:       totalLogs,
		NewAdmin:       isNewAdmin,
		AdminUser:      adminUser,
		AdminPass:      adminPass,
	}
	copyright.PrintCopyright(status)

	// 启动服务器（Server）
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logger.Info("Starting server on %s", addr)
	if err := r.Run(addr); err != nil {
		logger.Fatal("Failed to start server: %v", err)
	}
}
