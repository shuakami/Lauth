package v1

import (
	"net/http"
	"strings"

	"lauth/internal/model"
	"lauth/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler 创建认证处理器实例
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register 注册路由
func (h *AuthHandler) Register(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)
		auth.GET("/validate", h.ValidateToken)
	}
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	appID := c.Query("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), appID, &req)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		case service.ErrUserDisabled:
			c.JSON(http.StatusForbidden, gin.H{"error": "user is disabled"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RefreshToken 刷新访问令牌
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// 从Authorization头获取刷新令牌
	auth := c.GetHeader("Authorization")
	if auth == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
		return
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization scheme"})
		return
	}

	refreshToken := auth[len("Bearer "):]

	// 刷新令牌
	resp, err := h.authService.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		switch err {
		case service.ErrInvalidToken:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		case service.ErrTokenExpired:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		case service.ErrTokenRevoked:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to refresh token"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// 从Authorization头获取访问令牌
	auth := c.GetHeader("Authorization")
	if auth == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
		return
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization scheme"})
		return
	}

	accessToken := auth[len("Bearer "):]

	// 执行登出
	if err := h.authService.Logout(c.Request.Context(), accessToken); err != nil {
		switch err {
		case service.ErrInvalidToken:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		case service.ErrTokenExpired:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		case service.ErrTokenRevoked:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// ValidateToken 验证Token并返回用户信息
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	// 从Authorization头获取访问令牌
	auth := c.GetHeader("Authorization")
	if auth == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
		return
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization scheme"})
		return
	}

	accessToken := auth[len("Bearer "):]

	// 验证Token并获取用户信息
	userInfo, err := h.authService.ValidateTokenAndGetUser(c.Request.Context(), accessToken)
	if err != nil {
		switch err {
		case service.ErrInvalidToken:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		case service.ErrTokenExpired:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		case service.ErrTokenRevoked:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate token"})
		}
		return
	}

	c.JSON(http.StatusOK, userInfo)
}
