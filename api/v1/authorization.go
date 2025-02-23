package v1

import (
	"fmt"
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
		// 令牌端点
		oauth.POST("/token", h.HandleToken)
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

	// 返回重定向URL而不是直接重定向
	c.JSON(http.StatusOK, gin.H{
		"redirect_url": redirectURL,
	})
}

// validateTokenRequest 验证令牌请求参数
func (h *AuthorizationHandler) validateTokenRequest(c *gin.Context) (*model.TokenRequest, error) {
	var req model.TokenRequest
	if err := c.Request.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form data")
	}

	// 手动绑定表单数据
	req.GrantType = c.Request.PostForm.Get("grant_type")
	req.Code = c.Request.PostForm.Get("code")
	req.RedirectURI = c.Request.PostForm.Get("redirect_uri")
	req.ClientID = c.Request.PostForm.Get("client_id")
	req.ClientSecret = c.Request.PostForm.Get("client_secret")
	req.RefreshToken = c.Request.PostForm.Get("refresh_token")

	// 验证必填字段
	if req.GrantType == "" || req.ClientID == "" || req.ClientSecret == "" {
		return nil, fmt.Errorf("missing required parameters")
	}

	// 根据授权类型验证其他必填字段
	if req.GrantType == model.GrantTypeAuthorizationCode {
		if req.Code == "" || req.RedirectURI == "" {
			return nil, fmt.Errorf("code and redirect_uri are required for authorization_code grant type")
		}
	} else if req.GrantType == model.GrantTypeRefreshToken {
		if req.RefreshToken == "" {
			return nil, fmt.Errorf("refresh_token is required for refresh_token grant type")
		}
	}

	return &req, nil
}

// handleTokenError 处理令牌错误响应
func (h *AuthorizationHandler) handleTokenError(c *gin.Context, err error) {
	var statusCode int
	var tokenError model.TokenError

	switch err {
	case service.ErrInvalidClient:
		statusCode = http.StatusUnauthorized
		tokenError = model.TokenError{
			Error:            model.ErrorInvalidClient,
			ErrorDescription: "invalid client credentials",
		}
	case service.ErrInvalidGrant:
		statusCode = http.StatusBadRequest
		tokenError = model.TokenError{
			Error:            model.ErrorInvalidGrant,
			ErrorDescription: "invalid authorization code or refresh token",
		}
	case service.ErrUnsupportedGrantType:
		statusCode = http.StatusBadRequest
		tokenError = model.TokenError{
			Error:            model.ErrorUnsupportedGrantType,
			ErrorDescription: "unsupported grant type",
		}
	default:
		statusCode = http.StatusInternalServerError
		tokenError = model.TokenError{
			Error:            "server_error",
			ErrorDescription: "internal server error",
		}
	}

	c.JSON(statusCode, tokenError)
}

// HandleToken 处理令牌请求
func (h *AuthorizationHandler) HandleToken(c *gin.Context) {
	// 验证请求参数
	req, err := h.validateTokenRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.TokenError{
			Error:            model.ErrorInvalidRequest,
			ErrorDescription: err.Error(),
		})
		return
	}

	// 颁发令牌
	resp, err := h.authService.IssueToken(c.Request.Context(), req)
	if err != nil {
		h.handleTokenError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
