package v1

import (
	"log"
	"net/http"

	"lauth/internal/model"
	"lauth/internal/service"
	"lauth/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// AuthorizationHandler 授权处理器
type AuthorizationHandler struct {
	authService service.AuthorizationService
}

// NewAuthorizationHandler 创建授权处理器实例
func NewAuthorizationHandler(authService service.AuthorizationService) *AuthorizationHandler {
	return &AuthorizationHandler{
		authService: authService,
	}
}

// Register 注册路由
func (h *AuthorizationHandler) Register(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	oauth := group.Group("/oauth")
	{
		// 授权端点
		oauth.GET("/authorize", authMiddleware.HandleAuth(), h.HandleAuthorize)
	}
}

// HandleAuthorize 处理授权请求
func (h *AuthorizationHandler) HandleAuthorize(c *gin.Context) {
	var req model.AuthorizationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从认证中间件获取用户信息
	log.Printf("Authorization header: %s", c.GetHeader("Authorization"))
	claims := middleware.GetUserFromContext(c)
	log.Printf("User claims from context: %+v", claims)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 处理授权请求
	redirectURL, err := h.authService.Authorize(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		switch err {
		case service.ErrInvalidClient:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
		case service.ErrInvalidRedirectURI:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_redirect_uri"})
		case service.ErrInvalidScope:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_scope"})
		case service.ErrUnsupportedGrantType:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		}
		return
	}

	// 重定向到客户端
	c.Redirect(http.StatusFound, redirectURL)
}
