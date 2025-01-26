package v1

import (
	"net/http"
	"strconv"

	"lauth/internal/model"
	"lauth/internal/service"
	"lauth/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// OAuthClientHandler OAuth客户端处理器
type OAuthClientHandler struct {
	service service.OAuthClientService
}

// NewOAuthClientHandler 创建OAuth客户端处理器实例
func NewOAuthClientHandler(clientService service.OAuthClientService) *OAuthClientHandler {
	return &OAuthClientHandler{
		service: clientService,
	}
}

// Register 注册路由
func (h *OAuthClientHandler) Register(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	apps := group.Group("/apps")
	{
		// OAuth客户端管理
		apps.POST("/:id/oauth/clients", authMiddleware.HandleAuth(), h.CreateClient)
		apps.GET("/:id/oauth/clients/:client_id", authMiddleware.HandleAuth(), h.GetClient)
		apps.PUT("/:id/oauth/clients/:client_id", authMiddleware.HandleAuth(), h.UpdateClient)
		apps.DELETE("/:id/oauth/clients/:client_id", authMiddleware.HandleAuth(), h.DeleteClient)
		apps.GET("/:id/oauth/clients", authMiddleware.HandleAuth(), h.ListClients)

		// OAuth客户端秘钥管理
		apps.POST("/:id/oauth/clients/:client_id/secrets", authMiddleware.HandleAuth(), h.CreateClientSecret)
		apps.DELETE("/:id/oauth/clients/:client_id/secrets/:secret_id", authMiddleware.HandleAuth(), h.DeleteClientSecret)
		apps.GET("/:id/oauth/clients/:client_id/secrets", authMiddleware.HandleAuth(), h.ListClientSecrets)
	}
}

// CreateClient 创建OAuth客户端
func (h *OAuthClientHandler) CreateClient(c *gin.Context) {
	appID := c.Param("id")
	var req model.CreateOAuthClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.CreateClient(c.Request.Context(), appID, &req)
	if err != nil {
		switch err {
		case service.ErrClientExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// GetClient 获取OAuth客户端
func (h *OAuthClientHandler) GetClient(c *gin.Context) {
	clientID := c.Param("client_id")

	resp, err := h.service.GetClient(c.Request.Context(), clientID)
	if err != nil {
		switch err {
		case service.ErrClientNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateClient 更新OAuth客户端
func (h *OAuthClientHandler) UpdateClient(c *gin.Context) {
	clientID := c.Param("client_id")
	var req model.UpdateOAuthClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.UpdateClient(c.Request.Context(), clientID, &req)
	if err != nil {
		switch err {
		case service.ErrClientNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteClient 删除OAuth客户端
func (h *OAuthClientHandler) DeleteClient(c *gin.Context) {
	clientID := c.Param("client_id")

	if err := h.service.DeleteClient(c.Request.Context(), clientID); err != nil {
		switch err {
		case service.ErrClientNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// ListClients 获取OAuth客户端列表
func (h *OAuthClientHandler) ListClients(c *gin.Context) {
	appID := c.Param("id")

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}

	clients, total, err := h.service.ListClients(c.Request.Context(), appID, page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": clients,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// CreateClientSecret 创建客户端秘钥
func (h *OAuthClientHandler) CreateClientSecret(c *gin.Context) {
	var req model.CreateClientSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clientID := c.Param("client_id")
	resp, err := h.service.CreateClientSecret(c.Request.Context(), clientID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteClientSecret 删除客户端秘钥
func (h *OAuthClientHandler) DeleteClientSecret(c *gin.Context) {
	clientID := c.Param("client_id")
	secretID := c.Param("secret_id")

	if err := h.service.DeleteClientSecret(c.Request.Context(), clientID, secretID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListClientSecrets 获取客户端秘钥列表
func (h *OAuthClientHandler) ListClientSecrets(c *gin.Context) {
	clientID := c.Param("client_id")

	secrets, err := h.service.ListClientSecrets(c.Request.Context(), clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, secrets)
}
