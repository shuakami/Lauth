package plugin

import (
	"context"
	"fmt"
	"log"
	"sync"

	"lauth/internal/model"
	"lauth/internal/plugin/email"
	"lauth/internal/plugin/types"
	"lauth/internal/repository"
)

// PluginFactory 插件工厂函数类型
type PluginFactory func() types.Plugin

// manager 插件管理器实现
type manager struct {
	// appPlugins 存储每个App的插件映射（运行时缓存）
	// 格式: map[appID]map[pluginName]Plugin
	appPlugins sync.Map

	// configRepo 插件配置存储
	configRepo repository.PluginConfigRepository

	// pluginFactories 插件工厂函数映射
	pluginFactories map[string]PluginFactory
}

// NewManager 创建插件管理器实例
func NewManager(configRepo repository.PluginConfigRepository) types.Manager {
	m := &manager{
		configRepo:      configRepo,
		pluginFactories: make(map[string]PluginFactory),
	}

	// 注册内置插件
	m.RegisterPlugin("email_verify", func() types.Plugin {
		return email.NewEmailPlugin()
	})

	return m
}

// RegisterPlugin 注册插件
func (m *manager) RegisterPlugin(name string, factory PluginFactory) {
	m.pluginFactories[name] = factory
}

// createPlugin 创建插件实例
func (m *manager) createPlugin(name string) (types.Plugin, error) {
	factory, exists := m.pluginFactories[name]
	if !exists {
		return nil, fmt.Errorf("unknown plugin type: %s", name)
	}
	return factory(), nil
}

// LoadPlugin 加载插件
func (m *manager) LoadPlugin(appID string, p types.Plugin, config map[string]interface{}) error {
	if appID == "" {
		return fmt.Errorf("app_id is required")
	}

	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

	name := p.Name()
	if name == "" {
		return fmt.Errorf("plugin name is empty")
	}

	// 获取或创建App的插件映射
	var appPluginMap *sync.Map
	if v, ok := m.appPlugins.Load(appID); ok {
		appPluginMap = v.(*sync.Map)
	} else {
		appPluginMap = &sync.Map{}
		m.appPlugins.Store(appID, appPluginMap)
	}

	// 检查插件是否已存在
	if _, exists := appPluginMap.Load(name); exists {
		return fmt.Errorf("plugin %s already exists for app %s", name, appID)
	}

	// 加载插件
	if err := p.Load(config); err != nil {
		return fmt.Errorf("failed to load plugin %s for app %s: %v", name, appID, err)
	}

	// 获取插件元数据
	metadata := p.GetMetadata()

	// 存储插件配置到数据库
	pluginConfig := &model.PluginConfig{
		AppID:    appID,
		Name:     name,
		Config:   config,
		Required: metadata.Required,
		Stage:    metadata.Stage,
		Actions:  metadata.Actions,
		Enabled:  true,
	}
	if err := m.configRepo.SaveConfig(context.Background(), pluginConfig); err != nil {
		return fmt.Errorf("failed to save plugin config: %v", err)
	}

	// 存储插件实例到内存
	appPluginMap.Store(name, p)
	return nil
}

// UnloadPlugin 卸载插件
func (m *manager) UnloadPlugin(appID string, name string) error {
	if appID == "" {
		return fmt.Errorf("app_id is required")
	}

	if name == "" {
		return fmt.Errorf("plugin name is empty")
	}

	// 获取App的插件映射
	appPluginMapValue, exists := m.appPlugins.Load(appID)
	if !exists {
		return fmt.Errorf("no plugins found for app %s", appID)
	}
	appPluginMap := appPluginMapValue.(*sync.Map)

	// 获取插件
	p, exists := appPluginMap.Load(name)
	if !exists {
		return fmt.Errorf("plugin %s not found for app %s", name, appID)
	}

	// 卸载插件
	plugin := p.(types.Plugin)
	if err := plugin.Unload(); err != nil {
		return fmt.Errorf("failed to unload plugin %s for app %s: %v", name, appID, err)
	}

	// 从数据库中删除插件配置
	if err := m.configRepo.DeleteConfig(context.Background(), appID, name); err != nil {
		return fmt.Errorf("failed to delete plugin config: %v", err)
	}

	// 从内存中移除插件
	appPluginMap.Delete(name)
	return nil
}

// GetPlugin 获取插件
func (m *manager) GetPlugin(appID string, name string) (types.Plugin, bool) {
	if appID == "" || name == "" {
		return nil, false
	}

	// 首先尝试从内存中获取
	if appPluginMapValue, exists := m.appPlugins.Load(appID); exists {
		appPluginMap := appPluginMapValue.(*sync.Map)
		if p, exists := appPluginMap.Load(name); exists {
			return p.(types.Plugin), true
		}
	}

	// 如果内存中不存在,尝试从数据库加载
	config, err := m.configRepo.GetConfig(context.Background(), appID, name)
	if err != nil || config == nil || !config.Enabled {
		return nil, false
	}

	// 创建新的插件实例
	p, err := m.createPlugin(name)
	if err != nil {
		log.Printf("Failed to create plugin %s: %v", name, err)
		return nil, false
	}

	// 加载插件配置
	if err := p.Load(config.Config); err != nil {
		log.Printf("Failed to load plugin %s for app %s: %v", name, appID, err)
		return nil, false
	}

	// 获取或创建App的插件映射
	var appPluginMap *sync.Map
	if v, ok := m.appPlugins.Load(appID); ok {
		appPluginMap = v.(*sync.Map)
	} else {
		appPluginMap = &sync.Map{}
		m.appPlugins.Store(appID, appPluginMap)
	}

	// 存储到内存中
	appPluginMap.Store(name, p)

	return p, true
}

// ExecutePlugin 执行指定插件
func (m *manager) ExecutePlugin(ctx context.Context, appID string, name string, params map[string]interface{}) error {
	p, exists := m.GetPlugin(appID, name)
	if !exists {
		return fmt.Errorf("plugin %s not found for app %s", name, appID)
	}

	return p.Execute(ctx, params)
}

// ListPlugins 列出App的所有插件
func (m *manager) ListPlugins(appID string) []string {
	var plugins []string

	if appID == "" {
		return plugins
	}

	// 从数据库中获取已启用的插件配置
	configs, err := m.configRepo.ListConfigs(context.Background(), appID)
	if err != nil {
		log.Printf("Failed to list plugin configs for app %s: %v", appID, err)
		return plugins
	}
	log.Printf("Found %d plugin configs for app %s", len(configs), appID)

	// 收集已启用的插件名称
	for _, config := range configs {
		if config.Enabled {
			plugins = append(plugins, config.Name)
			log.Printf("Added enabled plugin: %s", config.Name)
		}
	}

	return plugins
}

// InitPlugins 初始化插件（从数据库加载插件配置）
func (m *manager) InitPlugins(ctx context.Context) error {
	// 获取所有应用的插件配置
	configs, err := m.configRepo.ListConfigs(ctx, "")
	if err != nil {
		log.Printf("Failed to list all plugin configs: %v", err)
		return fmt.Errorf("failed to list plugin configs: %v", err)
	}
	log.Printf("Found %d total plugin configs", len(configs))

	// 按应用ID分组
	appConfigs := make(map[string][]*model.PluginConfig)
	for _, config := range configs {
		if !config.Enabled {
			continue
		}
		appConfigs[config.AppID] = append(appConfigs[config.AppID], config)
		log.Printf("Added config for app %s: plugin=%s", config.AppID, config.Name)
	}

	// 为每个应用加载插件
	for appID, configs := range appConfigs {
		log.Printf("Loading plugins for app %s: count=%d", appID, len(configs))
		// 创建应用的插件映射
		appPluginMap := &sync.Map{}
		m.appPlugins.Store(appID, appPluginMap)

		// 加载每个插件
		for _, config := range configs {
			// 创建插件实例
			p, err := m.createPlugin(config.Name)
			if err != nil {
				log.Printf("Failed to create plugin %s: %v", config.Name, err)
				continue
			}

			// 加载插件
			if err := p.Load(config.Config); err != nil {
				log.Printf("Failed to load plugin %s for app %s: %v", config.Name, appID, err)
				continue
			}

			// 存储插件实例
			appPluginMap.Store(config.Name, p)
		}
	}

	return nil
}

// GetPluginConfigs 获取应用的所有插件配置
func (m *manager) GetPluginConfigs(ctx context.Context, appID string) ([]*model.PluginConfig, error) {
	return m.configRepo.ListConfigs(ctx, appID)
}
