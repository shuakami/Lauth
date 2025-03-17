package router

import (
	v1 "lauth/api/v1"
	"lauth/internal/audit"
	"lauth/internal/plugin/types"
	"lauth/pkg/middleware"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Router 路由管理器
type Router struct {
	engine                    *gin.Engine
	authMiddleware            *middleware.AuthMiddleware
	authHandler               *v1.AuthHandler
	appHandler                *v1.AppHandler
	userHandler               *v1.UserHandler
	permissionHandler         *v1.PermissionHandler
	roleHandler               *v1.RoleHandler
	ruleHandler               *v1.RuleHandler
	oauthClientHandler        *v1.OAuthClientHandler
	authzHandler              *v1.AuthorizationHandler
	profileHandler            *v1.ProfileHandler
	fileHandler               *v1.FileHandler
	oidcHandler               *v1.OIDCHandler
	auditHandler              *v1.AuditHandler
	pluginHandler             *v1.PluginHandler
	auditPermissionMiddleware *audit.AuditPermissionMiddleware
	loginLocationHandler      *v1.LoginLocationHandler
	superAdminHandler         *v1.SuperAdminHandler
	superAdminMiddleware      *middleware.SuperAdminMiddleware
}

// NewRouter 创建路由管理器实例
func NewRouter(
	engine *gin.Engine,
	authMiddleware *middleware.AuthMiddleware,
	authHandler *v1.AuthHandler,
	appHandler *v1.AppHandler,
	userHandler *v1.UserHandler,
	permissionHandler *v1.PermissionHandler,
	roleHandler *v1.RoleHandler,
	ruleHandler *v1.RuleHandler,
	oauthClientHandler *v1.OAuthClientHandler,
	authzHandler *v1.AuthorizationHandler,
	profileHandler *v1.ProfileHandler,
	fileHandler *v1.FileHandler,
	oidcHandler *v1.OIDCHandler,
	auditHandler *v1.AuditHandler,
	pluginHandler *v1.PluginHandler,
	auditPermissionMiddleware *audit.AuditPermissionMiddleware,
	loginLocationHandler *v1.LoginLocationHandler,
	superAdminHandler *v1.SuperAdminHandler,
	superAdminMiddleware *middleware.SuperAdminMiddleware,
) *Router {
	return &Router{
		engine:                    engine,
		authMiddleware:            authMiddleware,
		authHandler:               authHandler,
		appHandler:                appHandler,
		userHandler:               userHandler,
		permissionHandler:         permissionHandler,
		roleHandler:               roleHandler,
		ruleHandler:               ruleHandler,
		oauthClientHandler:        oauthClientHandler,
		authzHandler:              authzHandler,
		profileHandler:            profileHandler,
		fileHandler:               fileHandler,
		oidcHandler:               oidcHandler,
		auditHandler:              auditHandler,
		pluginHandler:             pluginHandler,
		auditPermissionMiddleware: auditPermissionMiddleware,
		loginLocationHandler:      loginLocationHandler,
		superAdminHandler:         superAdminHandler,
		superAdminMiddleware:      superAdminMiddleware,
	}
}

// RegisterRoutes 注册所有路由
func (r *Router) RegisterRoutes() {
	// 健康检查
	r.engine.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// API v1
	api := r.engine.Group("/api/v1")
	{
		// 注册认证相关路由
		r.registerAuthRoutes(api)
		// 注册应用相关路由
		r.registerAppRoutes(api)
		// 注册用户相关路由
		r.registerUserRoutes(api)
		// 注册权限相关路由
		r.registerPermissionRoutes(api)
		// 注册角色相关路由
		r.registerRoleRoutes(api)
		// 注册规则相关路由
		r.registerRuleRoutes(api)
		// 注册OAuth客户端相关路由
		r.registerOAuthRoutes(api)
		// 注册OAuth授权相关路由
		r.registerAuthorizationRoutes(api)
		// 注册Profile相关路由
		r.registerProfileRoutes(api)
		// 注册文件相关路由
		r.registerFileRoutes(api)
		// 注册OIDC相关路由
		r.registerOIDCRoutes(api)
		// 注册审计相关路由
		r.registerAuditRoutes(api)
		// 注册插件相关路由
		r.registerPluginRoutes(api)
		// 注册登录位置相关路由
		r.registerLoginLocationRoutes(api)
		// 注册超级管理员相关路由
		r.registerSuperAdminRoutes(api)
	}

	// OIDC发现端点（必须在根路径）
	r.engine.GET("/.well-known/openid-configuration", r.oidcHandler.GetConfiguration)
	r.engine.GET("/.well-known/jwks.json", r.oidcHandler.GetJWKS)
}

// registerAuthRoutes 注册认证相关路由
func (r *Router) registerAuthRoutes(group *gin.RouterGroup) {
	auth := group.Group("/auth")
	{
		auth.POST("/login", r.authHandler.Login)
		auth.POST("/refresh", r.authHandler.RefreshToken)
		auth.POST("/logout", r.authHandler.Logout)
		auth.GET("/validate", r.authHandler.ValidateToken)
		auth.POST("/validate-rule", r.authHandler.ValidateTokenAndRule)
	}
}

// registerAppRoutes 注册应用相关路由
func (r *Router) registerAppRoutes(group *gin.RouterGroup) {
	appsGroup := group.Group("/apps")
	// 添加超级管理员权限检查 - 创建、更新和删除应用需要超级管理员权限
	appsGroup.POST("", r.superAdminMiddleware.CheckSuperAdmin(), r.appHandler.CreateApp)
	appsGroup.PUT("/:id", r.superAdminMiddleware.CheckSuperAdmin(), r.appHandler.UpdateApp)
	appsGroup.DELETE("/:id", r.superAdminMiddleware.CheckSuperAdmin(), r.appHandler.DeleteApp)
	// 以下API允许普通认证用户访问
	appsGroup.Use(r.authMiddleware.HandleAuth())
	{
		appsGroup.GET("", r.appHandler.ListApps)
		appsGroup.GET("/:id", r.appHandler.GetApp)
	}
}

// registerUserRoutes 注册用户相关路由
func (r *Router) registerUserRoutes(group *gin.RouterGroup) {
	// 基于应用的用户管理路由
	apps := group.Group("/apps")
	{
		// 用户管理路由
		apps.POST("/:id/users", r.userHandler.CreateUser)
		apps.GET("/:id/users", r.authMiddleware.HandleAuth(), r.userHandler.ListUsers)
		apps.GET("/:id/users/:user_id", r.authMiddleware.HandleAuth(), r.userHandler.GetUser)
		apps.PUT("/:id/users/:user_id", r.authMiddleware.HandleAuth(), r.userHandler.UpdateUser)
		apps.PUT("/:id/users/:user_id/password", r.authMiddleware.HandleAuth(), r.userHandler.UpdatePassword)
		apps.DELETE("/:id/users/:user_id", r.authMiddleware.HandleAuth(), r.userHandler.DeleteUser)
	}

	// 用户资源路由（用于OAuth2.0和普通认证）
	users := group.Group("/users")
	users.Use(r.authMiddleware.HandleAuth())
	{
		users.GET("/me", r.userHandler.GetUserInfo)
	}
}

// registerPermissionRoutes 注册权限相关路由
func (r *Router) registerPermissionRoutes(group *gin.RouterGroup) {
	permissions := group.Group("/permissions")
	permissions.Use(r.superAdminMiddleware.CheckSuperAdmin())
	r.permissionHandler.Register(permissions, r.authMiddleware)
}

// registerRoleRoutes 注册角色相关路由
func (r *Router) registerRoleRoutes(group *gin.RouterGroup) {
	r.roleHandler.Register(group, r.authMiddleware)
}

// registerRuleRoutes 注册规则相关路由
func (r *Router) registerRuleRoutes(group *gin.RouterGroup) {
	rules := group.Group("/rules")
	rules.Use(r.superAdminMiddleware.CheckSuperAdmin())
	r.ruleHandler.Register(rules, r.authMiddleware)
}

// registerOAuthRoutes 注册OAuth相关路由
func (r *Router) registerOAuthRoutes(group *gin.RouterGroup) {
	oauth := group.Group("/oauth")
	oauth.Use(r.superAdminMiddleware.CheckSuperAdmin())
	r.oauthClientHandler.Register(oauth, r.authMiddleware)
}

// registerAuthorizationRoutes 注册OAuth授权相关路由
func (r *Router) registerAuthorizationRoutes(group *gin.RouterGroup) {
	r.authzHandler.Register(group, r.authMiddleware)
}

// registerProfileRoutes 注册Profile相关路由
func (r *Router) registerProfileRoutes(group *gin.RouterGroup) {
	r.profileHandler.Register(group, r.authMiddleware)
}

// registerFileRoutes 注册文件相关路由
func (r *Router) registerFileRoutes(group *gin.RouterGroup) {
	r.fileHandler.Register(group, r.authMiddleware)
}

// registerOIDCRoutes 注册OIDC相关路由
func (r *Router) registerOIDCRoutes(group *gin.RouterGroup) {
	r.oidcHandler.Register(group, r.authMiddleware)
}

// registerAuditRoutes 注册审计相关路由
func (r *Router) registerAuditRoutes(group *gin.RouterGroup) {
	audit := group.Group("/audit")
	// 超级管理员权限检查
	audit.Use(r.superAdminMiddleware.CheckSuperAdmin())
	{
		audit.GET("/logs", r.auditHandler.GetLogs)
		audit.GET("/logs/verify", r.auditHandler.VerifyLogFile)
		audit.GET("/stats", r.auditHandler.GetStats)
		audit.GET("/ws", r.auditHandler.HandleWebSocket)
	}
}

// pluginRouter 插件路由器
type pluginRouter struct {
	pluginGroup *gin.RouterGroup
	authGroup   *gin.RouterGroup
	authRoutes  []string
}

// Handle 处理路由注册
func (pr *pluginRouter) Handle(httpMethod, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	// 检查路径是否需要认证
	needAuth := false
	for _, authPath := range pr.authRoutes {
		if authPath == relativePath || authPath == "*" {
			needAuth = true
			break
		}
	}

	log.Printf("[Plugin Router] Registering route: %s %s (Auth Required: %v)", httpMethod, relativePath, needAuth)

	if needAuth {
		return pr.authGroup.Handle(httpMethod, relativePath, handlers...)
	}
	return pr.pluginGroup.Handle(httpMethod, relativePath, handlers...)
}

// POST 注册POST路由
func (pr *pluginRouter) POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return pr.Handle(http.MethodPost, relativePath, handlers...)
}

// GET 注册GET路由
func (pr *pluginRouter) GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return pr.Handle(http.MethodGet, relativePath, handlers...)
}

// DELETE 注册DELETE路由
func (pr *pluginRouter) DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return pr.Handle(http.MethodDelete, relativePath, handlers...)
}

// PATCH 注册PATCH路由
func (pr *pluginRouter) PATCH(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return pr.Handle(http.MethodPatch, relativePath, handlers...)
}

// PUT 注册PUT路由
func (pr *pluginRouter) PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return pr.Handle(http.MethodPut, relativePath, handlers...)
}

// OPTIONS 注册OPTIONS路由
func (pr *pluginRouter) OPTIONS(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return pr.Handle(http.MethodOptions, relativePath, handlers...)
}

// HEAD 注册HEAD路由
func (pr *pluginRouter) HEAD(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return pr.Handle(http.MethodHead, relativePath, handlers...)
}

// Match 匹配路由
func (pr *pluginRouter) Match(methods []string, path string, handlers ...gin.HandlerFunc) gin.IRoutes {
	for _, method := range methods {
		pr.Handle(method, path, handlers...)
	}
	return pr.pluginGroup
}

// Any 注册任意HTTP方法的路由
func (pr *pluginRouter) Any(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	// 注册所有HTTP方法
	pr.GET(relativePath, handlers...)
	pr.POST(relativePath, handlers...)
	pr.PUT(relativePath, handlers...)
	pr.PATCH(relativePath, handlers...)
	pr.HEAD(relativePath, handlers...)
	pr.OPTIONS(relativePath, handlers...)
	pr.DELETE(relativePath, handlers...)
	return pr.pluginGroup
}

// Use 添加中间件
func (pr *pluginRouter) Use(middleware ...gin.HandlerFunc) gin.IRoutes {
	pr.pluginGroup.Use(middleware...)
	pr.authGroup.Use(middleware...)
	return pr.pluginGroup
}

// AsRouterGroup 返回路由组接口
func (pr *pluginRouter) AsRouterGroup() *gin.RouterGroup {
	return pr.pluginGroup
}

// registerGlobalPluginRoutes 注册全局插件路由
func (r *Router) registerGlobalPluginRoutes(group *gin.RouterGroup) {
	group.GET("/plugins/all", r.pluginHandler.ListAllPlugins)
}

// injectAppIDMiddleware 注入appID中间件
func (r *Router) injectAppIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取appID
		appID := c.Param("id")
		if appID == "" {
			c.Next()
			return
		}

		// 获取插件名称
		pluginName := c.Param("name")
		if pluginName == "" {
			c.Next()
			return
		}

		log.Printf("[Plugin Middleware] Processing request for plugin: %s, appID: %s", pluginName, appID)
		// 设置appID到上下文
		c.Set("app_id", appID)
		c.Next()
	}
}

// wrapWithAppID 包装处理函数,注入appID参数
func (r *Router) wrapWithAppID(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Params = append(c.Params, gin.Param{
			Key:   "app_id",
			Value: c.Param("id"),
		})
		handler(c)
	}
}

// registerAppPluginRoutes 注册应用级插件路由
func (r *Router) registerAppPluginRoutes(appPlugins *gin.RouterGroup) {
	// 无需认证的API
	appPlugins.POST("/:name/execute", r.wrapWithAppID(r.pluginHandler.ExecutePlugin))

	// 需要认证的API
	auth := appPlugins.Group("")
	auth.Use(r.authMiddleware.HandleAuth())
	{
		auth.POST("/install", r.wrapWithAppID(r.pluginHandler.InstallPlugin))
		auth.POST("/uninstall/:name", r.wrapWithAppID(r.pluginHandler.UninstallPlugin))
		auth.PUT("/:name/config", r.wrapWithAppID(r.pluginHandler.UpdatePluginConfig))
		auth.GET("/list", r.wrapWithAppID(r.pluginHandler.ListPlugins))
		auth.GET("/all", r.pluginHandler.ListAllPlugins)
		auth.POST("/load", r.wrapWithAppID(r.pluginHandler.LoadPlugin))
	}
}

// registerDynamicPluginRoutes 注册动态插件路由
func (r *Router) registerDynamicPluginRoutes(appPlugins *gin.RouterGroup) {
	// 获取已注册的插件列表(仅仅是所有"插件类型")
	registeredPlugins := types.GetRegisteredPlugins()
	log.Printf("[Plugin Routes] Found %d registered plugins", len(registeredPlugins))

	// 注册每个插件的路由
	for i, pluginMeta := range registeredPlugins {
		pluginName := pluginMeta.Name
		log.Printf("[Plugin Routes] Processing plugin #%d: %s", i+1, pluginName)

		// 为该插件创建独立的路由组
		pluginGroup := appPlugins.Group("/" + pluginName)
		log.Printf("[Plugin Routes] Created group for plugin %s: %s", pluginName, pluginGroup.BasePath())

		// 添加中间件: 获取已安装的插件实例并存入context
		pluginGroup.Use(func(c *gin.Context) {
			appID := c.Param("id")
			if appID == "" {
				log.Printf("[Plugin Routes] Missing :id in path, can't fetch plugin")
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing appID"})
				return
			}

			installedPlugin, ok := r.pluginHandler.GetPlugin(appID, pluginName)
			if !ok {
				log.Printf("[Plugin Routes] Plugin %s not installed for app %s", pluginName, appID)
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "plugin not installed"})
				return
			}
			// 把"已安装的真正插件对象"存进 context
			c.Set("installed_plugin", installedPlugin)

			c.Next()
		})

		// 1) 创建临时插件实例用于注册路由
		p := pluginMeta.Factory()

		// 2) 如果插件支持路由,注册其路由
		if routable, ok := p.(types.Routable); ok {
			r.registerRoutablePluginRoutes(pluginName, routable, pluginGroup)
		} else {
			log.Printf("[Plugin Routes] Plugin %s is not routable", pluginName)
		}
	}
}

// registerRoutablePluginRoutes 注册可路由插件的路由
func (r *Router) registerRoutablePluginRoutes(pluginName string, routable types.Routable, pluginGroup *gin.RouterGroup) {
	// 获取需要认证的路由列表
	authRoutes := routable.GetRoutesRequireAuth()
	log.Printf("[Plugin Routes] Plugin %s has %d auth routes: %v", pluginName, len(authRoutes), authRoutes)

	// 判断是否有需要认证的路由
	if len(authRoutes) > 0 {
		// 检查是否需要对所有路由进行认证
		allRoutesNeedAuth := false
		for _, route := range authRoutes {
			if route == "*" {
				allRoutesNeedAuth = true
				break
			}
		}

		if allRoutesNeedAuth {
			// 所有路由都需要认证
			log.Printf("[Plugin Routes] All routes for plugin %s require auth", pluginName)
			authGroup := pluginGroup.Group("")
			authGroup.Use(r.authMiddleware.HandleAuth())
			routable.RegisterRoutes(authGroup)
		} else {
			// 部分路由需要认证
			log.Printf("[Plugin Routes] Some routes for plugin %s require auth", pluginName)
			// 创建两个路由组：一个需要认证，一个不需要认证
			authGroup := pluginGroup.Group("")
			authGroup.Use(r.authMiddleware.HandleAuth())

			// 注册路由时，根据路径决定使用哪个路由组
			router := &pluginRouter{
				pluginGroup: pluginGroup,
				authGroup:   authGroup,
				authRoutes:  authRoutes,
			}

			// 注册路由
			routable.RegisterRoutes(router.AsRouterGroup())
		}
	} else {
		// 所有路由都不需要认证
		log.Printf("[Plugin Routes] No routes for plugin %s require auth", pluginName)
		routable.RegisterRoutes(pluginGroup)
	}
}

// registerPluginRoutes 注册插件相关路由
func (r *Router) registerPluginRoutes(group *gin.RouterGroup) {
	// 注册全局插件路由
	r.registerGlobalPluginRoutes(group)

	// 应用插件API
	appPlugins := group.Group("/apps/:id/plugins")
	{
		// 注入appID中间件
		appPlugins.Use(r.injectAppIDMiddleware())

		// 注册应用级插件路由
		r.registerAppPluginRoutes(appPlugins)

		// 注册动态插件路由
		r.registerDynamicPluginRoutes(appPlugins)
	}
}

// registerLoginLocationRoutes 注册登录位置相关路由
func (r *Router) registerLoginLocationRoutes(group *gin.RouterGroup) {
	r.loginLocationHandler.Register(group, r.authMiddleware)
}

// registerSuperAdminRoutes 注册超级管理员相关路由
func (r *Router) registerSuperAdminRoutes(group *gin.RouterGroup) {
	// 需要超级管理员权限的路由
	systemGroup := group.Group("/system")
	systemGroup.Use(r.authMiddleware.HandleAuth())

	// 超级管理员相关API
	superAdminGroup := systemGroup.Group("/super-admins")
	superAdminGroup.Use(r.superAdminMiddleware.CheckSuperAdmin())
	{
		superAdminGroup.POST("", r.superAdminHandler.AddSuperAdmin)
		superAdminGroup.GET("", r.superAdminHandler.ListSuperAdmins)
		superAdminGroup.DELETE("/:user_id", r.superAdminHandler.RemoveSuperAdmin)
		superAdminGroup.GET("/check/:user_id", r.superAdminHandler.CheckSuperAdmin)
	}
}
