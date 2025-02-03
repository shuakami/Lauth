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
		auth.POST("/login/continue", h.ContinueLogin)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)
		auth.GET("/validate", h.ValidateToken)
		auth.POST("/validate-rule", h.ValidateTokenAndRule)
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

	// 收集验证上下文信息
	req.ClientIP = c.ClientIP()
	req.UserAgent = c.Request.UserAgent()

	// 从请求头中获取设备信息
	req.DeviceID = c.GetHeader("X-Device-ID")
	req.DeviceType = c.GetHeader("X-Device-Type")

	resp, err := h.authService.Login(c.Request.Context(), appID, &req)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		case service.ErrUserDisabled:
			c.JSON(http.StatusForbidden, gin.H{"error": "user is disabled"})
		case service.ErrPluginRequired:
			// 当需要插件验证时，返回验证相关信息
			c.JSON(http.StatusAccepted, gin.H{
				"auth_status": resp.AuthStatus,
				"plugins":     resp.Plugins,
				"next_plugin": resp.NextPlugin,
				"session_id":  resp.SessionID,
			})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 检查是否使用Cookie认证
	useCookie := c.GetHeader("Authorization") == ""

	if useCookie {
		// 设置HttpOnly Cookie
		c.SetCookie("access_token", resp.AccessToken, int(resp.ExpiresIn), "/", "", false, true)
		c.SetCookie("refresh_token", resp.RefreshToken, int(resp.ExpiresIn)*2, "/", "", false, true)

		// 返回不含token的响应
		c.JSON(http.StatusOK, model.LoginCookieResponse{
			User:      resp.User,
			ExpiresIn: resp.ExpiresIn,
		})
	} else {
		// 返回完整响应（包含token）
		c.JSON(http.StatusOK, resp)
	}
}

// RefreshToken 刷新访问令牌
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var refreshToken string
	useCookie := false

	// 优先从Cookie中获取refresh_token
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		// 如果Cookie中没有，则从Authorization头获取
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
			return
		}

		if !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization scheme"})
			return
		}

		refreshToken = auth[len("Bearer "):]
	} else {
		useCookie = true
	}

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

	if useCookie {
		// 更新HttpOnly Cookie
		c.SetCookie("access_token", resp.AccessToken, int(resp.ExpiresIn), "/", "", false, true)
		c.SetCookie("refresh_token", resp.RefreshToken, int(resp.ExpiresIn)*2, "/", "", false, true)

		// 返回不含token的响应
		c.JSON(http.StatusOK, model.LoginCookieResponse{
			User:      resp.User,
			ExpiresIn: resp.ExpiresIn,
		})
	} else {
		// 返回完整响应（包含token）
		c.JSON(http.StatusOK, resp)
	}
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	var accessToken string

	// 优先从Cookie中获取access_token
	accessToken, err := c.Cookie("access_token")
	if err != nil {
		// 如果Cookie中没有，则从Authorization头获取
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing access token"})
			return
		}

		if !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization scheme"})
			return
		}

		accessToken = auth[len("Bearer "):]
	}

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

	// 清除Cookie
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.Status(http.StatusNoContent)
}

// ValidateToken 验证Token并返回用户信息
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	var accessToken string

	// 优先从Cookie中获取access_token
	accessToken, err := c.Cookie("access_token")
	if err != nil {
		// 如果Cookie中没有，则从Authorization头获取
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing access token"})
			return
		}

		if !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization scheme"})
			return
		}

		accessToken = auth[len("Bearer "):]
	}

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

// ValidateTokenAndRuleRequest 组合验证请求
type ValidateTokenAndRuleRequest struct {
	Data map[string]interface{} `json:"data" binding:"required"`
}

// ValidateTokenAndRule 组合验证令牌和规则
func (h *AuthHandler) ValidateTokenAndRule(c *gin.Context) {
	var accessToken string

	// 优先从Cookie中获取access_token
	accessToken, err := c.Cookie("access_token")
	if err != nil {
		// 如果Cookie中没有，则从Authorization头获取
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing access token"})
			return
		}

		if !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization scheme"})
			return
		}

		accessToken = auth[len("Bearer "):]
	}

	// 解析请求体
	var req ValidateTokenAndRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证Token并获取用户信息
	userInfo, err := h.authService.ValidateTokenAndRuleWithUser(c.Request.Context(), accessToken, req.Data)
	if err != nil {
		switch err {
		case service.ErrInvalidToken:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		case service.ErrTokenExpired:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		case service.ErrTokenRevoked:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate"})
		}
		return
	}

	c.JSON(http.StatusOK, userInfo)
}

// ContinueLoginRequest 继续登录请求
type ContinueLoginRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// ContinueLogin 继续登录
func (h *AuthHandler) ContinueLogin(c *gin.Context) {
	var req ContinueLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authService.ContinueLogin(c.Request.Context(), req.SessionID)
	if err != nil {
		switch err {
		case service.ErrPluginRequired:
			// 当需要插件验证时，返回验证相关信息
			c.JSON(http.StatusAccepted, gin.H{
				"auth_status": resp.AuthStatus,
				"plugins":     resp.Plugins,
				"next_plugin": resp.NextPlugin,
				"session_id":  resp.SessionID,
			})
			return
		case service.ErrUserDisabled:
			c.JSON(http.StatusForbidden, gin.H{"error": "user is disabled"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 检查是否使用Cookie认证
	useCookie := c.GetHeader("Authorization") == ""

	if useCookie {
		// 设置HttpOnly Cookie
		c.SetCookie("access_token", resp.AccessToken, int(resp.ExpiresIn), "/", "", false, true)
		c.SetCookie("refresh_token", resp.RefreshToken, int(resp.ExpiresIn)*2, "/", "", false, true)

		// 返回不含token的响应
		c.JSON(http.StatusOK, model.LoginCookieResponse{
			User:      resp.User,
			ExpiresIn: resp.ExpiresIn,
		})
	} else {
		// 返回完整响应（包含token）
		c.JSON(http.StatusOK, resp)
	}
}
