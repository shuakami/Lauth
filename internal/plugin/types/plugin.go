package types

import (
	"context"
	"lauth/internal/model"
)

// Plugin 验证插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string

	// GetMetadata 返回插件元数据
	GetMetadata() *PluginMetadata

	// Load 加载插件
	Load(config map[string]interface{}) error

	// Unload 卸载插件
	Unload() error

	// Execute 执行插件逻辑
	Execute(ctx context.Context, params map[string]interface{}) error
}

// PluginMetadata 插件元数据
type PluginMetadata struct {
	Name        string   `json:"name"`        // 插件名称
	Description string   `json:"description"` // 插件描述
	Version     string   `json:"version"`     // 插件版本
	Author      string   `json:"author"`      // 插件作者
	Required    bool     `json:"required"`    // 是否默认必需
	Stage       string   `json:"stage"`       // 默认执行阶段
	Actions     []string `json:"actions"`     // 支持的动作列表
}

// Manager 插件管理器接口
type Manager interface {
	// LoadPlugin 加载插件
	LoadPlugin(appID string, p Plugin, config map[string]interface{}) error

	// UnloadPlugin 卸载插件
	UnloadPlugin(appID string, name string) error

	// GetPlugin 获取插件
	GetPlugin(appID string, name string) (Plugin, bool)

	// ExecutePlugin 执行指定插件
	ExecutePlugin(ctx context.Context, appID string, name string, params map[string]interface{}) error

	// ListPlugins 列出App的所有插件
	ListPlugins(appID string) []string

	// InitPlugins 初始化插件（从数据库加载插件配置）
	InitPlugins(ctx context.Context) error

	// GetPluginConfigs 获取应用的所有插件配置
	GetPluginConfigs(ctx context.Context, appID string) ([]*model.PluginConfig, error)
}
