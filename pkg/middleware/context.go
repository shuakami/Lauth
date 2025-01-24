package middleware

import (
	"lauth/internal/model"

	"github.com/gin-gonic/gin"
)

// GetUserFromContext 从上下文中获取用户信息
func GetUserFromContext(c *gin.Context) *model.TokenClaims {
	if user, exists := c.Get(ContextKeyUser); exists {
		if claims, ok := user.(*model.TokenClaims); ok {
			return claims
		}
	}
	return nil
}

// MustGetUserFromContext 从上下文中获取用户信息，如果不存在则panic
func MustGetUserFromContext(c *gin.Context) *model.TokenClaims {
	claims := GetUserFromContext(c)
	if claims == nil {
		panic("user not found in context")
	}
	return claims
}
