package audit

import (
	"context"
	"errors"
	"net/http"

	"lauth/internal/model"
	"lauth/internal/service"

	"github.com/gin-gonic/gin"
)

var (
	ErrNoPermission = errors.New("no permission to access audit logs")
)

// AuditPermissionMiddleware 审计日志权限检查中间件
type AuditPermissionMiddleware struct {
	roleService    service.RoleService
	permissionCode string
}

// NewAuditPermissionMiddleware 创建新的审计日志权限检查中间件
func NewAuditPermissionMiddleware(roleService service.RoleService) *AuditPermissionMiddleware {
	return &AuditPermissionMiddleware{
		roleService:    roleService,
		permissionCode: "audit:read", // 默认权限代码
	}
}

// Handle 处理权限检查
func (m *AuditPermissionMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户信息
		claims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		}

		// 获取应用ID参数
		appID := c.Query("app_id")
		if appID == "" {
			appID = c.Param("app_id")
		}

		// 检查用户是否有权限访问审计日志
		tokenClaims := claims.(*model.TokenClaims)
		hasPermission, err := m.checkPermission(tokenClaims.UserID, appID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": ErrNoPermission.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkPermission 检查用户是否有权限访问指定应用的审计日志
func (m *AuditPermissionMiddleware) checkPermission(userID string, appID string) (bool, error) {
	// 检查用户是否有全局审计日志读取权限
	hasGlobalPermission, err := m.roleService.HasPermission(context.Background(), userID, m.permissionCode)
	if err != nil {
		return false, err
	}
	if hasGlobalPermission {
		return true, nil
	}

	// 如果指定了应用ID，检查用户是否有该应用的审计日志读取权限
	if appID != "" {
		appSpecificPermission := m.permissionCode + ":" + appID
		hasAppPermission, err := m.roleService.HasPermission(context.Background(), userID, appSpecificPermission)
		if err != nil {
			return false, err
		}
		return hasAppPermission, nil
	}

	return false, nil
}

// SetPermissionCode 设置权限代码
func (m *AuditPermissionMiddleware) SetPermissionCode(code string) {
	m.permissionCode = code
}
