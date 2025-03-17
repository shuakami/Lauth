package middleware

import (
	"context"
	"errors"
	"lauth/internal/model"
	"lauth/internal/service"
	"lauth/pkg/api"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SuperAdminMiddleware 超级管理员权限检查中间件
type SuperAdminMiddleware struct {
	tokenService      service.TokenService
	superAdminService service.SuperAdminService
}

// NewSuperAdminMiddleware 创建超级管理员中间件
func NewSuperAdminMiddleware(tokenService service.TokenService, superAdminService service.SuperAdminService) *SuperAdminMiddleware {
	return &SuperAdminMiddleware{
		tokenService:      tokenService,
		superAdminService: superAdminService,
	}
}

// extractToken 从请求中提取token
func extractToken(c *gin.Context) (string, error) {
	// 1. 尝试从Authorization头获取
	auth := c.GetHeader("Authorization")
	if auth != "" && strings.HasPrefix(auth, BearerSchema) {
		return auth[len(BearerSchema):], nil
	}

	// 2. 尝试从Cookie获取
	if cookie, err := c.Cookie(CookieAccessToken); err == nil {
		return cookie, nil
	}

	return "", errors.New("未找到token")
}

// CheckSuperAdmin 检查请求用户是否为超级管理员
func (m *SuperAdminMiddleware) CheckSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求中获取token
		token, err := extractToken(c)
		if err != nil {
			api.Error(c, http.StatusUnauthorized, "无效的访问令牌", err)
			c.Abort()
			return
		}

		// 验证token
		claims, err := m.tokenService.ValidateToken(context.Background(), token, model.AccessToken)
		if err != nil {
			api.Error(c, http.StatusUnauthorized, "访问令牌验证失败", err)
			c.Abort()
			return
		}

		// 检查用户是否为超级管理员
		isSuperAdmin, err := m.superAdminService.IsSuperAdmin(c.Request.Context(), claims.UserID)
		if err != nil {
			api.Error(c, http.StatusInternalServerError, "检查超级管理员权限失败", err)
			c.Abort()
			return
		}

		if !isSuperAdmin {
			api.Error(c, http.StatusForbidden, "需要超级管理员权限", nil)
			c.Abort()
			return
		}

		// 将用户ID和角色添加到上下文中
		c.Set("userID", claims.UserID)
		c.Set("isSuperAdmin", true)

		c.Next()
	}
}
