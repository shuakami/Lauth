package v1

import (
	"net/http"

	"lauth/internal/model"
	"lauth/internal/plugin/email"
	"lauth/internal/plugin/types"
	"lauth/internal/service"

	"github.com/gin-gonic/gin"
)

// PluginHandler 插件处理器
type PluginHandler struct {
	pluginManager types.Manager
	verifyService service.VerificationService
}

// NewPluginHandler 创建插件处理器实例
func NewPluginHandler(manager types.Manager, verifyService service.VerificationService) *PluginHandler {
	return &PluginHandler{
		pluginManager: manager,
		verifyService: verifyService,
	}
}

// LoadPluginRequest 加载插件请求
type LoadPluginRequest struct {
	Name   string                 `json:"name" binding:"required"`
	Config map[string]interface{} `json:"config"`
}

// LoadPlugin 加载插件
func (h *PluginHandler) LoadPlugin(c *gin.Context) {
	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	var req LoadPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 根据插件名称创建对应的插件实例
	var p types.Plugin
	switch req.Name {
	case "email_verify":
		p = email.NewEmailPlugin()
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown plugin type"})
		return
	}

	if err := h.pluginManager.LoadPlugin(appID, p, req.Config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// UnloadPlugin 卸载插件
func (h *PluginHandler) UnloadPlugin(c *gin.Context) {
	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	name := c.Param("name")
	if err := h.pluginManager.UnloadPlugin(appID, name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// ExecutePluginRequest 执行插件请求
type ExecutePluginRequest struct {
	Params map[string]interface{} `json:"params" binding:"required"`
}

// ExecutePlugin 执行插件
func (h *PluginHandler) ExecutePlugin(c *gin.Context) {
	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	name := c.Param("name")
	var req ExecutePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取当前用户ID
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	// 获取验证会话
	session, err := h.verifyService.GetSession(c.Request.Context(), appID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get verification session"})
		return
	}
	if session == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active verification session"})
		return
	}

	// 获取插件实例
	plugin, exists := h.pluginManager.GetPlugin(appID, name)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugin not found"})
		return
	}

	// 验证action是否支持
	metadata := plugin.GetMetadata()
	actionSupported := false
	for _, action := range metadata.Actions {
		if action == session.Action {
			actionSupported = true
			break
		}
	}
	if !actionSupported {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action not supported by this plugin"})
		return
	}

	if err := h.pluginManager.ExecutePlugin(c.Request.Context(), appID, name, req.Params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 更新插件状态为已完成
	if err := h.verifyService.UpdatePluginStatus(
		c.Request.Context(),
		appID,
		userID,
		session.Action,
		name,
		model.PluginStatusCompleted,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update plugin status"})
		return
	}

	c.Status(http.StatusOK)
}

// ListPlugins 列出所有插件
func (h *PluginHandler) ListPlugins(c *gin.Context) {
	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	plugins := h.pluginManager.ListPlugins(appID)
	c.JSON(http.StatusOK, gin.H{
		"plugins": plugins,
	})
}
