package boot

import (
	"context"
	"time"

	"lauth/internal/plugin"
	"lauth/internal/plugin/types"
	"lauth/internal/repository"
	"lauth/internal/service"
	"lauth/pkg/config"
	"lauth/pkg/crypto"
	"lauth/pkg/engine"
	"lauth/pkg/middleware"
	"lauth/pkg/redis"

	"gorm.io/gorm"
)

// Services 包含所有服务实例
type Services struct {
	AppService                   service.AppService
	FileService                  service.FileService
	ProfileService               service.ProfileService
	UserService                  service.UserService
	RuleService                  service.RuleService
	VerificationService          service.VerificationService
	AuthService                  service.AuthService
	RoleService                  service.RoleService
	PermissionService            service.PermissionService
	OAuthClientService           service.OAuthClientService
	OIDCService                  service.OIDCService
	AuthorizationService         service.AuthorizationService
	IPLocationService            service.IPLocationService
	LoginLocationService         service.LoginLocationService
	TokenService                 service.TokenService
	SuperAdminService            service.SuperAdminService
	PluginManager                types.Manager
	PluginUserConfigRepo         repository.PluginUserConfigRepository
	PluginVerificationRecordRepo repository.PluginVerificationRecordRepository
}

// InitServices 初始化所有服务实例
func InitServices(cfg *config.Config, repos *Repositories, redisClient *redis.Client, db *gorm.DB) (*Services, error) {
	// 初始化IP地理位置服务
	ipLocationService := service.NewIPLocationService("data/ip2region/ip2region.xdb")

	// 初始化登录位置服务
	loginLocationService := service.NewLoginLocationService(repos.LoginLocationRepo, ipLocationService)

	// 初始化Token服务
	tokenService := service.NewTokenService(
		redisClient,
		cfg.JWT.Secret,
		time.Duration(cfg.JWT.AccessTokenExpire)*time.Hour,
		time.Duration(cfg.JWT.RefreshTokenExpire)*time.Second,
	)

	// 初始化认证中间件
	authMiddleware := middleware.NewAuthMiddleware(tokenService, cfg.Server.AuthEnabled)

	// 初始化规则引擎
	ruleParser := engine.NewParser()
	ruleExecutor := engine.NewExecutor()
	ruleCache := engine.NewCache(redisClient)
	ruleEngine := engine.NewEngine(ruleParser, ruleExecutor, ruleCache, repos.RuleRepo)

	// 初始化插件管理器
	pluginManager := plugin.NewManager(
		repos.PluginConfigRepo,
		repos.PluginUserConfigRepo,
		repos.PluginVerificationRecordRepo,
		repos.VerificationSessionRepo,
		loginLocationService,
		&cfg.SMTP,
		authMiddleware,
	)

	// 加载插件配置
	if err := pluginManager.InitPlugins(context.Background()); err != nil {
		return nil, err
	}

	// 初始化超级管理员服务
	superAdminService := service.NewSuperAdminService(repos.SuperAdminRepo, repos.UserRepo, repos.AppRepo)

	// 初始化基础服务
	appService := service.NewAppService(repos.AppRepo)
	fileService := service.NewFileService(repos.FileRepo)
	profileService := service.NewProfileService(repos.ProfileRepo, repos.FileRepo)
	userService := service.NewUserService(repos.UserRepo, repos.AppRepo, profileService)
	ruleService := service.NewRuleService(repos.RuleRepo, ruleEngine)
	verificationService := service.NewVerificationService(pluginManager, repos.PluginStatusRepo, repos.VerificationSessionRepo)
	authService := service.NewAuthService(
		repos.UserRepo,
		repos.AppRepo,
		tokenService,
		ruleService,
		verificationService,
		profileService,
		loginLocationService,
		superAdminService,
		db,
	)
	roleService := service.NewRoleService(repos.RoleRepo, repos.PermissionRepo, superAdminService)
	permissionService := service.NewPermissionService(repos.PermissionRepo, repos.RoleRepo)
	oauthClientService := service.NewOAuthClientService(repos.OAuthClientRepo, repos.OAuthClientSecretRepo)

	// 初始化OIDC服务
	privateKey, publicKey, err := crypto.LoadRSAKeys(cfg.OIDC.PrivateKeyPath, cfg.OIDC.PublicKeyPath)
	if err != nil {
		return nil, err
	}
	oidcService := service.NewOIDCService(repos.UserRepo, tokenService, cfg, privateKey, publicKey)

	// 初始化授权服务
	authorizationService := service.NewAuthorizationService(
		repos.OAuthClientRepo,
		repos.OAuthClientSecretRepo,
		repos.AuthCodeRepo,
		repos.UserRepo,
		tokenService,
		oidcService,
	)

	return &Services{
		AppService:                   appService,
		FileService:                  fileService,
		ProfileService:               profileService,
		UserService:                  userService,
		RuleService:                  ruleService,
		VerificationService:          verificationService,
		AuthService:                  authService,
		RoleService:                  roleService,
		PermissionService:            permissionService,
		OAuthClientService:           oauthClientService,
		OIDCService:                  oidcService,
		AuthorizationService:         authorizationService,
		IPLocationService:            ipLocationService,
		LoginLocationService:         loginLocationService,
		TokenService:                 tokenService,
		SuperAdminService:            superAdminService,
		PluginManager:                pluginManager,
		PluginUserConfigRepo:         repos.PluginUserConfigRepo,
		PluginVerificationRecordRepo: repos.PluginVerificationRecordRepo,
	}, nil
}
