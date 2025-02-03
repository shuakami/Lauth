package types

import "context"

// UserConfigManager 用户配置管理器接口
type UserConfigManager interface {
	// GetConfig 获取用户配置
	GetConfig(ctx context.Context, userID string) (map[string]interface{}, error)

	// SaveConfig 保存用户配置
	SaveConfig(ctx context.Context, userID string, config map[string]interface{}) error

	// UpdateConfig 更新用户配置(合并现有配置)
	UpdateConfig(ctx context.Context, userID string, updates map[string]interface{}) error

	// ValidateConfig 验证配置
	ValidateConfig(config map[string]interface{}) error

	// RegisterHandler 注册配置处理器
	RegisterHandler(handler ConfigHandler)
}

// ConfigHandler 配置处理器接口
type ConfigHandler interface {
	// OnConfigGet 获取配置时的处理
	OnConfigGet(config map[string]interface{}) error

	// OnConfigSave 保存配置时的处理
	OnConfigSave(config map[string]interface{}) error

	// OnConfigUpdate 更新配置时的处理
	OnConfigUpdate(oldConfig, newConfig map[string]interface{}) error
}

// BaseConfigHandler 基础配置处理器实现
type BaseConfigHandler struct{}

func (h *BaseConfigHandler) OnConfigGet(config map[string]interface{}) error      { return nil }
func (h *BaseConfigHandler) OnConfigSave(config map[string]interface{}) error     { return nil }
func (h *BaseConfigHandler) OnConfigUpdate(old, new map[string]interface{}) error { return nil }
