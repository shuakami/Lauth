package middleware

import (
	"net/http"
	"strings"

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

		// 从Authorization头获取token
		auth := c.GetHeader("Authorization")
		log.Printf("Received Authorization header: %s", auth)
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// 验证Bearer方案
		if !strings.HasPrefix(auth, BearerSchema) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization scheme"})
			c.Abort()
			return
		}

		// 提取token
		token := auth[len(BearerSchema):]
		log.Printf("Extracted token: %s", token)

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

		log.Printf("Token validated successfully, claims: %+v", claims)
		// 将用户信息存入上下文
		c.Set(ContextKeyUser, claims)
		c.Next()
	}
}
