package user

import (
	"context"
	"fmt"
	"sync"

	"lauth/internal/model"
	"lauth/internal/plugin/types"
	"lauth/internal/repository"
)

// ConfigService 用户配置服务
type ConfigService struct {
	repo     repository.PluginUserConfigRepository
	mu       sync.RWMutex
	handlers []types.ConfigHandler
}

// NewConfigService 创建用户配置服务
func NewConfigService(repo repository.PluginUserConfigRepository) types.UserConfigManager {
	return &ConfigService{
		repo:     repo,
		handlers: make([]types.ConfigHandler, 0),
	}
}

// GetConfig 获取用户配置
func (s *ConfigService) GetConfig(ctx context.Context, userID string) (map[string]interface{}, error) {
	// 参数校验
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	// 从全局注册表获取所有已注册的插件
	registeredPlugins := types.GetRegisteredPlugins()

	// 创建结果映射
	configs := make(map[string]interface{})

	// 从上下文中获取appID
	appID := ctx.Value("app_id").(string)
	if appID == "" {
		appID = "default" // 如果上下文中没有appID，才使用默认值
	}

	// 遍历所有已注册的插件，获取每个插件的用户配置
	for _, plugin := range registeredPlugins {
		config, err := s.repo.GetUserConfig(ctx, appID, userID, plugin.Name)
		if err != nil {
			// 检测上下文取消错误并传播
			if ctx.Err() != nil {
				return nil, fmt.Errorf("context error while getting config for plugin %s: %w", plugin.Name, ctx.Err())
			}

			// 根据repository的实现，GetUserConfig在出现数据库错误时会返回错误
			// 如果是其他错误，我们记录并继续处理其他插件
			continue
		}

		// 根据repository的实现，当配置不存在时，GetUserConfig返回nil, nil
		// 只有当配置存在时，我们才添加到结果映射中
		if config != nil && config.Config != nil {
			configs[plugin.Name] = config.Config
		}
	}

	// 调用所有已注册的处理器
	s.mu.RLock()
	handlers := make([]types.ConfigHandler, len(s.handlers))
	copy(handlers, s.handlers)
	s.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler.OnConfigGet(configs); err != nil {
			return nil, fmt.Errorf("handler error: %w", err)
		}
	}

	return configs, nil
}

// SaveConfig 保存用户配置
func (s *ConfigService) SaveConfig(ctx context.Context, userID string, config map[string]interface{}) error {
	// 参数校验
	if userID == "" {
		return fmt.Errorf("userID cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if len(config) == 0 {
		return nil // 空配置，直接返回成功
	}

	// 检查上下文是否已取消
	if ctx.Err() != nil {
		return fmt.Errorf("context error: %w", ctx.Err())
	}

	// 调用所有已注册的处理器
	s.mu.RLock()
	handlers := make([]types.ConfigHandler, len(s.handlers))
	copy(handlers, s.handlers)
	s.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler.OnConfigSave(config); err != nil {
			return fmt.Errorf("handler error: %w", err)
		}
	}

	// 保存用户配置到仓库
	for plugin, pluginConfig := range config {
		if plugin == "" {
			continue // 跳过空插件名
		}

		configMap, ok := pluginConfig.(map[string]interface{})
		if !ok {
			// 如果不是map类型，创建一个包含该值的map
			configMap = map[string]interface{}{"value": pluginConfig}
		}

		pluginConfig := &model.PluginUserConfig{
			AppID:  "default", // 使用默认应用ID
			UserID: userID,
			Plugin: plugin,
			Config: configMap,
		}

		if err := s.repo.SaveUserConfig(ctx, pluginConfig); err != nil {
			return fmt.Errorf("failed to save config for plugin %s: %w", plugin, err)
		}
	}

	return nil
}

// validateUpdateParams 验证更新参数
func (s *ConfigService) validateUpdateParams(ctx context.Context, userID string, updates map[string]interface{}) (string, error) {
	// 参数校验
	if userID == "" {
		return "", fmt.Errorf("userID cannot be empty")
	}
	if updates == nil {
		return "", fmt.Errorf("updates cannot be nil")
	}
	if len(updates) == 0 {
		return "", nil // 空更新，直接返回成功
	}

	// 检查上下文是否已取消
	if ctx.Err() != nil {
		return "", fmt.Errorf("context error: %w", ctx.Err())
	}

	// 从上下文中获取appID
	appID := ctx.Value("app_id").(string)
	if appID == "" {
		appID = "default" // 如果上下文中没有appID，才使用默认值
	}

	return appID, nil
}

// convertToConfigMap 转换配置为映射
func (s *ConfigService) convertToConfigMap(update interface{}) map[string]interface{} {
	updateMap, ok := update.(map[string]interface{})
	if !ok {
		// 如果不是map类型，创建一个包含该值的map
		updateMap = map[string]interface{}{"value": update}
	}
	return updateMap
}

// callConfigHandlers 调用配置处理器
func (s *ConfigService) callConfigHandlers(oldConfig, newConfig map[string]interface{}) error {
	s.mu.RLock()
	handlers := make([]types.ConfigHandler, len(s.handlers))
	copy(handlers, s.handlers)
	s.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler.OnConfigUpdate(oldConfig, newConfig); err != nil {
			return fmt.Errorf("handler error: %w", err)
		}
	}
	return nil
}

// updatePluginConfig 更新插件配置
func (s *ConfigService) updatePluginConfig(ctx context.Context, appID, userID, plugin string, update interface{}) error {
	// 查找现有配置
	existingConfig, err := s.repo.GetUserConfig(ctx, appID, userID, plugin)
	if err != nil {
		// 如果上下文已取消，直接返回错误
		if ctx.Err() != nil {
			return fmt.Errorf("context error while getting config for plugin %s: %w", plugin, ctx.Err())
		}

		// 如果不存在则创建新配置
		newConfig := map[string]interface{}{
			plugin: update,
		}
		return s.SaveConfig(ctx, userID, newConfig)
	}

	// 获取现有配置的副本用于处理器
	oldConfig := make(map[string]interface{})
	if existingConfig != nil && existingConfig.Config != nil {
		oldConfig[plugin] = existingConfig.Config
	}

	// 更新现有配置
	updateMap := s.convertToConfigMap(update)

	// 确保Config映射已初始化
	if existingConfig.Config == nil {
		existingConfig.Config = make(map[string]interface{})
	}

	// 更新现有配置
	for k, v := range updateMap {
		existingConfig.Config[k] = v
	}

	// 创建新配置的副本用于处理器
	newConfig := make(map[string]interface{})
	newConfig[plugin] = existingConfig.Config

	// 调用配置处理器
	if err := s.callConfigHandlers(oldConfig, newConfig); err != nil {
		return err
	}

	// 保存更新后的配置
	return s.repo.SaveUserConfig(ctx, existingConfig)
}

// UpdateConfig 更新用户配置
func (s *ConfigService) UpdateConfig(ctx context.Context, userID string, updates map[string]interface{}) error {
	// 验证参数并获取appID
	appID, err := s.validateUpdateParams(ctx, userID, updates)
	if err != nil {
		return err
	}
	if appID == "" {
		return nil // 空更新，直接返回成功
	}

	// 对于每个插件配置更新或创建
	for plugin, update := range updates {
		if plugin == "" {
			continue // 跳过空插件名
		}

		if err := s.updatePluginConfig(ctx, appID, userID, plugin, update); err != nil {
			return fmt.Errorf("failed to update config for plugin %s: %w", plugin, err)
		}
	}

	return nil
}

// ValidateConfig 验证配置
func (s *ConfigService) ValidateConfig(config map[string]interface{}) error {
	// 参数校验
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// 遍历所有插件配置进行验证
	for plugin, pluginConfig := range config {
		if plugin == "" {
			return fmt.Errorf("plugin name cannot be empty")
		}

		// 确保配置值不为nil
		if pluginConfig == nil {
			return fmt.Errorf("config for plugin %s cannot be nil", plugin)
		}

		// 如果配置是map类型，进行深度检查
		if configMap, ok := pluginConfig.(map[string]interface{}); ok {
			// 空的配置映射也是有效的
			if len(configMap) == 0 {
				continue
			}
		}
	}

	return nil
}

// RegisterHandler 注册配置处理器
func (s *ConfigService) RegisterHandler(handler types.ConfigHandler) {
	if handler == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handler)
}
