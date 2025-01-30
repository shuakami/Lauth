package middleware

import (
	"net/http"
	"strings"
	"time"

	"lauth/internal/model"
	"lauth/internal/service"

	"log"

	"github.com/gin-gonic/gin"
)

const (
	// BearerSchema Bearer认证方案
	BearerSchema = "Bearer "
	// ContextKeyUser 上下文中用户信息的键
	ContextKeyUser = "user"
	// CookieAccessToken Cookie中访问令牌的键
	CookieAccessToken = "access_token"
)

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	tokenService service.TokenService
	enabled      bool
}

// NewAuthMiddleware 创建认证中间件实例
func NewAuthMiddleware(tokenService service.TokenService, enabled bool) *AuthMiddleware {
	return &AuthMiddleware{
		tokenService: tokenService,
		enabled:      enabled,
	}
}

// HandleAuth 处理认证
func (m *AuthMiddleware) HandleAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果认证被禁用，直接放行
		if !m.enabled {
			log.Printf("Auth middleware is disabled")
			c.Next()
			return
		}

		// 获取token
		token := m.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		// 验证token
		claims, err := m.tokenService.ValidateToken(c.Request.Context(), token, model.AccessToken)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
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
			c.Abort()
			return
		}

		// 设置过期时间响应头
		remainingTime := time.Until(claims.GetExpiresAt())
		if remainingTime > 0 {
			c.Header("X-Token-Expires-In", remainingTime.String())
		}

		log.Printf("Token validated successfully, claims: %+v", claims)
		// 将用户信息存入上下文
		c.Set(ContextKeyUser, claims)
		// 设置常用字段
		c.Set("user_id", claims.UserID)
		c.Set("app_id", claims.AppID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

// extractToken 从请求中提取token
func (m *AuthMiddleware) extractToken(c *gin.Context) string {
	// 1. 尝试从Authorization头获取
	auth := c.GetHeader("Authorization")
	if auth != "" && strings.HasPrefix(auth, BearerSchema) {
		log.Printf("Found token in Authorization header: %s", auth[len(BearerSchema):])
		return auth[len(BearerSchema):]
	}

	// 2. 尝试从Cookie获取
	if cookie, err := c.Cookie(CookieAccessToken); err == nil {
		log.Printf("Found token in Cookie: %s", cookie)
		return cookie
	} else {
		log.Printf("No token found in Cookie, error: %v", err)
	}

	log.Printf("No token found in request")
	return ""
}
