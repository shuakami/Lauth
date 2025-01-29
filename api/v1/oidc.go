package v1

import (
	"net/http"

	"lauth/internal/model"
	"lauth/internal/service"
	"lauth/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// OIDCHandler OIDC处理器
type OIDCHandler struct {
	oidcService  service.OIDCService
	tokenService service.TokenService
}

// NewOIDCHandler 创建OIDC处理器实例
func NewOIDCHandler(oidcService service.OIDCService, tokenService service.TokenService) *OIDCHandler {
	return &OIDCHandler{
		oidcService:  oidcService,
		tokenService: tokenService,
	}
}

// Register 注册路由
func (h *OIDCHandler) Register(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	oidc := group.Group("/oidc")
	{
		// OIDC发现端点
		oidc.GET("/.well-known/openid-configuration", h.GetConfiguration)
		// JWKS端点
		oidc.GET("/.well-known/jwks.json", h.GetJWKS)
		// UserInfo端点
		oidc.GET("/userinfo", authMiddleware.HandleAuth(), h.GetUserInfo)
	}
}

// GetConfiguration 处理OIDC配置请求
func (h *OIDCHandler) GetConfiguration(c *gin.Context) {
	config, err := h.oidcService.GetConfiguration(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get configuration"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetJWKS 处理JWKS请求
func (h *OIDCHandler) GetJWKS(c *gin.Context) {
	jwks, err := h.oidcService.GetJWKS(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get jwks"})
		return
	}
	c.JSON(http.StatusOK, jwks)
}

// GetUserInfo 处理UserInfo请求
func (h *OIDCHandler) GetUserInfo(c *gin.Context) {
	// 从认证中间件获取用户信息
	claims := middleware.GetUserFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 从Authorization头获取访问令牌
	auth := c.GetHeader("Authorization")
	if auth == "" || len(auth) <= 7 || auth[:7] != "Bearer " {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}
	token := auth[7:]

	// 获取令牌关联的scope
	tokenClaims, err := h.tokenService.ValidateToken(c.Request.Context(), token, model.AccessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}

	// 获取用户信息
	userInfo, err := h.oidcService.GetUserInfo(c.Request.Context(), claims.UserID, tokenClaims.Scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	c.JSON(http.StatusOK, userInfo)
}
