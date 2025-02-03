package v1

import (
	"fmt"
	"net/http"

	"lauth/internal/model"
	"lauth/internal/plugin/types"
	"lauth/internal/repository"
	"lauth/internal/service"
	"lauth/pkg/config"

	"github.com/gin-gonic/gin"
)

// PluginHandler 插件处理器
type PluginHandler struct {
	pluginManager    types.Manager
	verifyService    service.VerificationService
	userConfigRepo   repository.PluginUserConfigRepository
	verificationRepo repository.PluginVerificationRecordRepository
	smtpConfig       *config.SMTPConfig
}

// NewPluginHandler 创建插件处理器实例
func NewPluginHandler(
	manager types.Manager,
	verifyService service.VerificationService,
	userConfigRepo repository.PluginUserConfigRepository,
	verificationRepo repository.PluginVerificationRecordRepository,
	smtpConfig *config.SMTPConfig,
) *PluginHandler {
	return &PluginHandler{
		pluginManager:    manager,
		verifyService:    verifyService,
		userConfigRepo:   userConfigRepo,
		verificationRepo: verificationRepo,
		smtpConfig:       smtpConfig,
	}
}

// LoadPluginRequest 加载插件请求
type LoadPluginRequest struct {
	Name   string                 `json:"name" binding:"required"`
	Config map[string]interface{} `json:"config"`
}

// InstallPluginRequest 安装插件请求
type InstallPluginRequest struct {
	Name   string                 `json:"name" binding:"required"`   // 插件名称
	Config map[string]interface{} `json:"config" binding:"required"` // 插件配置
}

// InstallPlugin 安装插件
func (h *PluginHandler) InstallPlugin(c *gin.Context) {
	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	var req InstallPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 安装插件
	if err := h.pluginManager.InstallPlugin(c.Request.Context(), appID, req.Name, req.Config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// UninstallPlugin 卸载插件
func (h *PluginHandler) UninstallPlugin(c *gin.Context) {
	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	name := c.Param("name")
	if err := h.pluginManager.UninstallPlugin(c.Request.Context(), appID, name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// LoadPlugin 加载插件(已弃用,请使用InstallPlugin)
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

	// 使用InstallPlugin替代
	if err := h.pluginManager.InstallPlugin(c.Request.Context(), appID, req.Name, req.Config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// ExecutePluginRequest 执行插件请求
type ExecutePluginRequest struct {
	Operation      string                 `json:"operation" binding:"required"` // 操作类型（如：send、verify）
	Params         map[string]interface{} `json:"params" binding:"required"`    // 操作参数
	Identifier     string                 `json:"identifier"`                   // 标识符（邮箱/手机号）
	IdentifierType string                 `json:"identifier_type"`              // 标识符类型
	SessionID      string                 `json:"session_id"`                   // 会话ID
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

	var session *model.VerificationSession
	var err error

	// 获取当前用户ID（如果有）
	userID := c.GetString("user_id")

	// 优先使用session_id获取会话
	if req.SessionID != "" {
		session, err = h.verifyService.GetSessionByID(c.Request.Context(), req.SessionID)
	} else if userID != "" {
		session, err = h.verifyService.GetSession(c.Request.Context(), appID, userID)
	} else if req.Identifier != "" && req.IdentifierType != "" {
		session, err = h.verifyService.GetSessionByIdentifier(c.Request.Context(), appID, req.Identifier, req.IdentifierType)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "either session_id, user_id or identifier is required"})
		return
	}

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

	// 验证业务场景是否支持
	metadata := plugin.GetMetadata()
	actionSupported := false
	for _, action := range metadata.Actions {
		if action == session.Action {
			actionSupported = true
			break
		}
	}
	if !actionSupported {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("plugin doesn't support %s action", session.Action)})
		return
	}

	// 将操作类型添加到参数中
	req.Params["operation"] = req.Operation

	// 执行插件
	if err := h.pluginManager.ExecutePlugin(c.Request.Context(), appID, name, req.Params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 如果是verify操作且执行成功，更新插件状态为已完成
	if req.Operation == "verify" {
		if err := h.verifyService.UpdatePluginStatusBySession(
			c.Request.Context(),
			session.ID,
			name,
			model.PluginStatusCompleted,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update plugin status"})
			return
		}
	}

	c.Status(http.StatusOK)
}

// PluginInfo 插件信息
type PluginInfo struct {
	Name        string                    `json:"name"`        // 插件名称
	Description string                    `json:"description"` // 插件描述
	Version     string                    `json:"version"`     // 插件版本
	Author      string                    `json:"author"`      // 插件作者
	Required    bool                      `json:"required"`    // 是否必需
	Stage       string                    `json:"stage"`       // 执行阶段
	Actions     []string                  `json:"actions"`     // 支持的业务场景
	Operations  []types.OperationMetadata `json:"operations"`  // 支持的操作类型
	APIs        []types.APIInfo           `json:"apis"`        // API接口信息
	Enabled     bool                      `json:"enabled"`     // 是否启用
	Config      map[string]interface{}    `json:"config"`      // 配置信息
}

// GetSmartPlugin 获取SmartPlugin实例
func (h *PluginHandler) GetSmartPlugin(appID string, name string) (types.SmartPlugin, bool) {
	return h.pluginManager.GetSmartPlugin(appID, name)
}

// RegisterPluginRoutes 注册插件路由
func (h *PluginHandler) RegisterPluginRoutes(appID string, routerGroup *gin.RouterGroup) error {
	return h.pluginManager.RegisterPluginRoutes(appID, routerGroup)
}

// ListPlugins 列出所有插件
func (h *PluginHandler) ListPlugins(c *gin.Context) {
	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	// 获取插件配置列表
	configs, err := h.pluginManager.GetPluginConfigs(c.Request.Context(), appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get plugin configs"})
		return
	}

	// 构建插件信息列表
	var pluginInfos []PluginInfo
	for _, config := range configs {
		// 获取插件实例以获取元数据
		plugin, exists := h.pluginManager.GetPlugin(appID, config.Name)
		if !exists {
			continue
		}

		// 获取插件元数据
		metadata := plugin.GetMetadata()

		// 构建插件信息
		pluginInfo := PluginInfo{
			Name:        metadata.Name,
			Description: metadata.Description,
			Version:     metadata.Version,
			Author:      metadata.Author,
			Required:    metadata.Required,
			Stage:       metadata.Stage,
			Actions:     metadata.Actions,
			Operations:  metadata.Operations,
			Enabled:     config.Enabled,
			Config:      config.Config,
		}

		// 如果是SmartPlugin,添加API信息
		if sp, ok := plugin.(types.SmartPlugin); ok {
			pluginInfo.APIs = sp.GetAPIInfo()
		}

		pluginInfos = append(pluginInfos, pluginInfo)
	}

	c.JSON(http.StatusOK, pluginInfos)
}

// UpdatePluginConfigRequest 更新插件配置请求
type UpdatePluginConfigRequest struct {
	Name   string                 `json:"name" binding:"required"`   // 插件名称
	Config map[string]interface{} `json:"config" binding:"required"` // 插件配置
}

// UpdatePluginConfig 更新插件配置
func (h *PluginHandler) UpdatePluginConfig(c *gin.Context) {
	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}

	var req UpdatePluginConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取当前配置
	config, err := h.pluginManager.GetPluginConfigs(c.Request.Context(), appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get plugin config"})
		return
	}

	// 查找要更新的插件配置
	var pluginConfig *model.PluginConfig
	for _, cfg := range config {
		if cfg.Name == req.Name {
			pluginConfig = cfg
			break
		}
	}

	if pluginConfig == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plugin not found"})
		return
	}

	// 更新配置
	pluginConfig.Config = req.Config

	// 保存配置
	if err := h.pluginManager.SavePluginConfig(c.Request.Context(), pluginConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save plugin config: %v", err)})
		return
	}

	c.Status(http.StatusOK)
}
