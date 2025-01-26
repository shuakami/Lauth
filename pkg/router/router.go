package router

import (
	v1 "lauth/api/v1"
	"lauth/pkg/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Router 路由管理器
type Router struct {
	engine             *gin.Engine
	authMiddleware     *middleware.AuthMiddleware
	authHandler        *v1.AuthHandler
	appHandler         *v1.AppHandler
	userHandler        *v1.UserHandler
	permissionHandler  *v1.PermissionHandler
	roleHandler        *v1.RoleHandler
	ruleHandler        *v1.RuleHandler
	oauthClientHandler *v1.OAuthClientHandler
	authzHandler       *v1.AuthorizationHandler
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
) *Router {
	return &Router{
		engine:             engine,
		authMiddleware:     authMiddleware,
		authHandler:        authHandler,
		appHandler:         appHandler,
		userHandler:        userHandler,
		permissionHandler:  permissionHandler,
		roleHandler:        roleHandler,
		ruleHandler:        ruleHandler,
		oauthClientHandler: oauthClientHandler,
		authzHandler:       authzHandler,
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
	}
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
	apps := group.Group("/apps")
	{
		apps.POST("", r.authMiddleware.HandleAuth(), r.appHandler.CreateApp)
		apps.GET("", r.authMiddleware.HandleAuth(), r.appHandler.ListApps)
		apps.GET("/:id", r.authMiddleware.HandleAuth(), r.appHandler.GetApp)
		apps.PUT("/:id", r.authMiddleware.HandleAuth(), r.appHandler.UpdateApp)
		apps.DELETE("/:id", r.authMiddleware.HandleAuth(), r.appHandler.DeleteApp)
	}
}

// registerUserRoutes 注册用户相关路由
func (r *Router) registerUserRoutes(group *gin.RouterGroup) {
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
}

// registerPermissionRoutes 注册权限相关路由
func (r *Router) registerPermissionRoutes(group *gin.RouterGroup) {
	r.permissionHandler.Register(group, r.authMiddleware)
}

// registerRoleRoutes 注册角色相关路由
func (r *Router) registerRoleRoutes(group *gin.RouterGroup) {
	r.roleHandler.Register(group, r.authMiddleware)
}

// registerRuleRoutes 注册规则相关路由
func (r *Router) registerRuleRoutes(group *gin.RouterGroup) {
	r.ruleHandler.Register(group, r.authMiddleware)
}

// registerOAuthRoutes 注册OAuth相关路由
func (r *Router) registerOAuthRoutes(group *gin.RouterGroup) {
	r.oauthClientHandler.Register(group, r.authMiddleware)
}

// registerAuthorizationRoutes 注册OAuth授权相关路由
func (r *Router) registerAuthorizationRoutes(group *gin.RouterGroup) {
	r.authzHandler.Register(group, r.authMiddleware)
}
