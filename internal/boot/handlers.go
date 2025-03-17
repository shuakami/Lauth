package boot

import (
	v1 "lauth/api/v1"
	"lauth/internal/service"
	"lauth/pkg/config"
	"lauth/pkg/middleware"
	"lauth/pkg/router"

	"github.com/gin-gonic/gin"
)

// Handlers 包含所有HTTP处理器
type Handlers struct {
	AppHandler           *v1.AppHandler
	UserHandler          *v1.UserHandler
	AuthHandler          *v1.AuthHandler
	RoleHandler          *v1.RoleHandler
	PermissionHandler    *v1.PermissionHandler
	RuleHandler          *v1.RuleHandler
	OAuthClientHandler   *v1.OAuthClientHandler
	AuthorizationHandler *v1.AuthorizationHandler
	ProfileHandler       *v1.ProfileHandler
	FileHandler          *v1.FileHandler
	OIDCHandler          *v1.OIDCHandler
	AuditHandler         *v1.AuditHandler
	PluginHandler        *v1.PluginHandler
	LoginLocationHandler *v1.LoginLocationHandler
	SuperAdminHandler    *v1.SuperAdminHandler
}

// InitHandlers 初始化所有HTTP处理器
func InitHandlers(services *Services, repos *Repositories, auditComponents *AuditComponents, cfg *config.Config) *Handlers {
	return &Handlers{
		AppHandler:           v1.NewAppHandler(services.AppService),
		UserHandler:          v1.NewUserHandler(services.UserService, services.AuthService, services.SuperAdminService),
		AuthHandler:          v1.NewAuthHandler(services.AuthService),
		RoleHandler:          v1.NewRoleHandler(services.RoleService),
		PermissionHandler:    v1.NewPermissionHandler(services.PermissionService),
		RuleHandler:          v1.NewRuleHandler(services.RuleService),
		OAuthClientHandler:   v1.NewOAuthClientHandler(services.OAuthClientService),
		AuthorizationHandler: v1.NewAuthorizationHandler(services.AuthorizationService),
		ProfileHandler:       v1.NewProfileHandler(services.ProfileService),
		FileHandler:          v1.NewFileHandler(services.FileService),
		OIDCHandler:          v1.NewOIDCHandler(services.OIDCService, services.TokenService),
		AuditHandler:         v1.NewAuditHandler(auditComponents.Reader, auditComponents.WebSocketServer),
		PluginHandler: v1.NewPluginHandler(
			services.PluginManager,
			services.VerificationService,
			repos.PluginUserConfigRepo,
			repos.PluginVerificationRecordRepo,
			&cfg.SMTP,
		),
		LoginLocationHandler: v1.NewLoginLocationHandler(services.LoginLocationService),
		SuperAdminHandler:    v1.NewSuperAdminHandler(services.SuperAdminService, services.UserService),
	}
}

// InitRouter 初始化路由
func InitRouter(
	engine *gin.Engine,
	handlers *Handlers,
	tokenService service.TokenService,
	ipLocationService service.IPLocationService,
	auditComponents *AuditComponents,
	cfg *config.Config,
	services *Services,
) *router.Router {
	// 初始化认证中间件
	authMiddleware := middleware.NewAuthMiddleware(tokenService, cfg.Server.AuthEnabled)

	// 初始化超级管理员中间件
	superAdminMiddleware := middleware.NewSuperAdminMiddleware(tokenService, services.SuperAdminService)

	// 初始化审计中间件
	auditMiddleware := middleware.NewAuditMiddleware(
		auditComponents.Writer,
		auditComponents.WebSocketServer,
		ipLocationService,
	)

	// 添加全局中间件
	engine.Use(middleware.CORSMiddleware())
	engine.Use(auditMiddleware.Handle())

	// 初始化路由管理器
	r := router.NewRouter(
		engine,
		authMiddleware,
		handlers.AuthHandler,
		handlers.AppHandler,
		handlers.UserHandler,
		handlers.PermissionHandler,
		handlers.RoleHandler,
		handlers.RuleHandler,
		handlers.OAuthClientHandler,
		handlers.AuthorizationHandler,
		handlers.ProfileHandler,
		handlers.FileHandler,
		handlers.OIDCHandler,
		handlers.AuditHandler,
		handlers.PluginHandler,
		auditComponents.AuditPermissionMiddleware,
		handlers.LoginLocationHandler,
		handlers.SuperAdminHandler,
		superAdminMiddleware,
	)

	// 注册所有路由
	r.RegisterRoutes()

	return r
}
