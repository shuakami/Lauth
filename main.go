package main

import (
	"context"
	"fmt"
	"log"
	"time"

	v1 "lauth/api/v1"
	"lauth/internal/audit"
	"lauth/internal/model"
	"lauth/internal/plugin"
	"lauth/internal/repository"
	"lauth/internal/service"
	"lauth/pkg/config"
	"lauth/pkg/crypto"
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

	// 连接MongoDB
	mongodbConfig := &database.MongoDBConfig{
		URI:         cfg.MongoDB.URI,
		Database:    cfg.MongoDB.Database,
		MaxPoolSize: cfg.MongoDB.MaxPoolSize,
		MinPoolSize: cfg.MongoDB.MinPoolSize,
	}
	mongodb, err := database.NewMongoClient(mongodbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	log.Printf("Successfully connected to MongoDB")

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
		&model.OAuthClient{},
		&model.OAuthClientSecret{},
		&model.AuthorizationCode{},
		&model.PluginStatus{},
		&model.PluginConfig{},
		&model.VerificationSession{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 获取底层sqlDB以便在程序结束时关闭
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying *sql.DB: %v", err)
	}
	defer sqlDB.Close()
	defer mongodb.Close(nil)

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
	oauthClientRepo := repository.NewOAuthClientRepository(db)
	oauthClientSecretRepo := repository.NewOAuthClientSecretRepository(db)
	authCodeRepo := repository.NewAuthorizationCodeRepository(db)
	pluginStatusRepo := repository.NewPluginStatusRepository(db)
	pluginConfigRepo := repository.NewPluginConfigRepository(db)
	verificationSessionRepo := repository.NewVerificationSessionRepository(db)

	// 初始化MongoDB仓储层
	profileRepo := repository.NewProfileRepository(mongodb)
	fileRepo := repository.NewFileRepository(mongodb)

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

	// 初始化插件管理器
	pluginManager := plugin.NewManager(pluginConfigRepo)

	// 加载插件配置
	if err := pluginManager.InitPlugins(context.Background()); err != nil {
		log.Fatalf("Failed to init plugins: %v", err)
	}

	// 初始化服务层
	appService := service.NewAppService(appRepo)
	fileService := service.NewFileService(fileRepo)
	profileService := service.NewProfileService(profileRepo, fileRepo)
	userService := service.NewUserService(userRepo, appRepo, profileService)
	ruleService := service.NewRuleService(ruleRepo, ruleEngine)
	verificationService := service.NewVerificationService(pluginManager, pluginStatusRepo, verificationSessionRepo)
	authService := service.NewAuthService(userRepo, tokenService, ruleService, verificationService)
	roleService := service.NewRoleService(roleRepo, permissionRepo)
	permissionService := service.NewPermissionService(permissionRepo, roleRepo)
	oauthClientService := service.NewOAuthClientService(oauthClientRepo, oauthClientSecretRepo)

	// 初始化OIDC服务
	privateKey, publicKey, err := crypto.LoadRSAKeys(cfg.OIDC.PrivateKeyPath, cfg.OIDC.PublicKeyPath)
	if err != nil {
		log.Fatalf("Failed to load RSA keys: %v", err)
	}
	oidcService := service.NewOIDCService(userRepo, tokenService, cfg, privateKey, publicKey)

	// 初始化授权服务
	authorizationService := service.NewAuthorizationService(
		oauthClientRepo,
		oauthClientSecretRepo,
		authCodeRepo,
		userRepo,
		tokenService,
		oidcService,
	)

	// 初始化审计系统组件
	hashChain := audit.NewHashChain()
	auditWriter, err := audit.NewWriter(audit.WriterConfig{
		BaseDir:    cfg.Audit.LogDir,
		RotateSize: cfg.Audit.RotationSize,
		HashChain:  hashChain,
	})
	if err != nil {
		log.Fatalf("Failed to create audit writer: %v", err)
	}
	defer auditWriter.Close()

	auditReader := audit.NewReader(cfg.Audit.LogDir)
	wsConfig := &audit.WebSocketConfig{
		PingInterval:   time.Duration(cfg.Audit.WebSocket.PingInterval) * time.Second,
		WriteWait:      time.Duration(cfg.Audit.WebSocket.WriteWait) * time.Second,
		ReadWait:       time.Duration(cfg.Audit.WebSocket.ReadWait) * time.Second,
		MaxMessageSize: int64(cfg.Audit.WebSocket.MaxMessageSize),
	}
	wsServer := audit.NewWebSocketServer(wsConfig)
	go wsServer.Start()

	// 创建默认的gin引擎
	r := gin.Default()

	// 添加CORS中间件
	r.Use(middleware.CORSMiddleware())

	// 初始化认证中间件
	authMiddleware := middleware.NewAuthMiddleware(tokenService, cfg.Server.AuthEnabled)

	// 初始化审计中间件
	auditMiddleware := middleware.NewAuditMiddleware(auditWriter, wsServer)
	r.Use(auditMiddleware.Handle())

	// 初始化审计日志权限中间件
	auditPermissionMiddleware := audit.NewAuditPermissionMiddleware(roleService)

	// 初始化处理器
	appHandler := v1.NewAppHandler(appService)
	userHandler := v1.NewUserHandler(userService)
	authHandler := v1.NewAuthHandler(authService)
	roleHandler := v1.NewRoleHandler(roleService)
	permissionHandler := v1.NewPermissionHandler(permissionService)
	ruleHandler := v1.NewRuleHandler(ruleService)
	oauthClientHandler := v1.NewOAuthClientHandler(oauthClientService)
	authorizationHandler := v1.NewAuthorizationHandler(authorizationService)
	profileHandler := v1.NewProfileHandler(profileService)
	fileHandler := v1.NewFileHandler(fileService)
	oidcHandler := v1.NewOIDCHandler(oidcService, tokenService)
	auditHandler := v1.NewAuditHandler(auditReader, wsServer)
	pluginHandler := v1.NewPluginHandler(pluginManager, verificationService)

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
		oauthClientHandler,
		authorizationHandler,
		profileHandler,
		fileHandler,
		oidcHandler,
		auditHandler,
		pluginHandler,
		auditPermissionMiddleware,
	)

	// 注册所有路由
	router.RegisterRoutes()

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
