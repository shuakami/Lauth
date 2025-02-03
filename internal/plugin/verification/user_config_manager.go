package verification

import (
	"context"
	"fmt"
	"sync"

	"lauth/internal/model"
	"lauth/internal/plugin/types"
	"lauth/internal/repository"
)

// DefaultUserConfigManager 默认的用户配置管理器实现
type DefaultUserConfigManager struct {
	mu         sync.RWMutex
	handlers   []types.ConfigHandler
	repo       repository.PluginUserConfigRepository
	appID      string
	pluginName string
}

// NewDefaultUserConfigManager 创建默认用户配置管理器
func NewDefaultUserConfigManager(
	repo repository.PluginUserConfigRepository,
	appID string,
	pluginName string,
) types.UserConfigManager {
	return &DefaultUserConfigManager{
		repo:       repo,
		appID:      appID,
		pluginName: pluginName,
		handlers:   make([]types.ConfigHandler, 0),
	}
}

// GetConfig 获取用户配置
func (m *DefaultUserConfigManager) GetConfig(ctx context.Context, userID string) (map[string]interface{}, error) {
	config, err := m.repo.GetUserConfig(ctx, m.appID, userID, m.pluginName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user config: %v", err)
	}
	if config == nil {
		return make(map[string]interface{}), nil
	}

	// 调用处理器
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, handler := range m.handlers {
		if err := handler.OnConfigGet(config.Config); err != nil {
			return nil, fmt.Errorf("config handler error: %v", err)
		}
	}

	return config.Config, nil
}

// SaveConfig 保存用户配置
func (m *DefaultUserConfigManager) SaveConfig(ctx context.Context, userID string, config map[string]interface{}) error {
	if err := m.ValidateConfig(config); err != nil {
		return err
	}

	// 调用处理器
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, handler := range m.handlers {
		if err := handler.OnConfigSave(config); err != nil {
			return fmt.Errorf("config handler error: %v", err)
		}
	}

	return m.repo.SaveUserConfig(ctx, &model.PluginUserConfig{
		AppID:  m.appID,
		UserID: userID,
		Plugin: m.pluginName,
		Config: config,
	})
}

// UpdateConfig 更新用户配置
func (m *DefaultUserConfigManager) UpdateConfig(ctx context.Context, userID string, updates map[string]interface{}) error {
	// 获取当前配置
	oldConfig, err := m.GetConfig(ctx, userID)
	if err != nil {
		return err
	}

	// 合并配置
	newConfig := make(map[string]interface{})
	for k, v := range oldConfig {
		newConfig[k] = v
	}
	for k, v := range updates {
		newConfig[k] = v
	}

	// 验证新配置
	if err := m.ValidateConfig(newConfig); err != nil {
		return err
	}

	// 调用处理器
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, handler := range m.handlers {
		if err := handler.OnConfigUpdate(oldConfig, newConfig); err != nil {
			return fmt.Errorf("config handler error: %v", err)
		}
	}

	return m.SaveConfig(ctx, userID, newConfig)
}

// ValidateConfig 验证配置
func (m *DefaultUserConfigManager) ValidateConfig(config map[string]interface{}) error {
	// 基础验证：确保配置不为nil
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	return nil
}

// RegisterHandler 注册配置处理器
func (m *DefaultUserConfigManager) RegisterHandler(handler types.ConfigHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// SetAppID 设置应用ID
func (m *DefaultUserConfigManager) SetAppID(appID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.appID = appID
}
