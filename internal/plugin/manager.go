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
	"lauth/pkg/config"
	"lauth/pkg/container"

	"github.com/gin-gonic/gin"
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

	// userConfigRepo 用户配置存储
	userConfigRepo repository.PluginUserConfigRepository

	// verificationRepo 验证记录存储
	verificationRepo repository.PluginVerificationRecordRepository

	// registry 插件注册表
	registry types.PluginRegistry

	// locationService 位置服务
	locationService types.LocationService

	// 依赖注入容器
	container container.PluginContainer
}

// NewManager 创建插件管理器实例
func NewManager(
	configRepo repository.PluginConfigRepository,
	userConfigRepo repository.PluginUserConfigRepository,
	verificationRepo repository.PluginVerificationRecordRepository,
	locationService types.LocationService,
	smtpConfig *config.SMTPConfig,
) types.Manager {
	m := &manager{
		configRepo:       configRepo,
		userConfigRepo:   userConfigRepo,
		verificationRepo: verificationRepo,
		registry:         NewRegistry(),
		locationService:  locationService,
		container:        container.NewPluginContainer(),
	}

	// 注册全局服务
	m.container.Register("location_service", locationService, true)
	m.container.Register("config_repo", configRepo, true)
	m.container.Register("user_config_repo", userConfigRepo, true)
	m.container.Register("verification_repo", verificationRepo, true)
	m.container.Register("smtp_config", smtpConfig, true)

	// 注册内置插件
	emailPlugin := email.NewEmailPlugin()
	metadata := emailPlugin.GetMetadata()
	var dependencies []string
	if injectable, ok := emailPlugin.(types.Injectable); ok {
		dependencies = injectable.GetDependencies()
	}
	if err := m.RegisterPlugin(&types.PluginDescriptor{
		Name:         metadata.Name,
		Version:      metadata.Version,
		Factory:      func() types.Plugin { return email.NewEmailPlugin() },
		Metadata:     metadata,
		Dependencies: dependencies,
	}); err != nil {
		log.Printf("Failed to register email plugin: %v", err)
	}

	return m
}

// RegisterPlugin 注册插件
func (m *manager) RegisterPlugin(descriptor *types.PluginDescriptor) error {
	return m.registry.Register(descriptor)
}

// createPlugin 创建插件实例
func (m *manager) createPlugin(name string, appID string) (types.Plugin, error) {
	// 从注册表获取插件描述符
	descriptor, exists := m.registry.Get(name)
	if !exists {
		return nil, fmt.Errorf("plugin %s not registered", name)
	}

	// 创建插件实例
	p := descriptor.Factory()
	if p == nil {
		return nil, fmt.Errorf("plugin factory returned nil")
	}

	// 注册插件特定的服务
	if err := m.container.RegisterPluginService(name, "app_id", appID); err != nil {
		return nil, fmt.Errorf("failed to register app_id service: %v", err)
	}

	// 配置插件（注入依赖）
	if injectable, ok := p.(types.Injectable); ok {
		if err := injectable.Configure(m.container); err != nil {
			return nil, fmt.Errorf("failed to configure plugin: %v", err)
		}
	}

	return p, nil
}

// GetSmartPlugin 获取SmartPlugin实例
func (m *manager) GetSmartPlugin(appID string, name string) (types.SmartPlugin, bool) {
	p, exists := m.GetPlugin(appID, name)
	if !exists {
		return nil, false
	}

	// 尝试类型转换
	if sp, ok := p.(types.SmartPlugin); ok {
		return sp, true
	}
	return nil, false
}

// RegisterPluginRoutes 注册插件路由
func (m *manager) RegisterPluginRoutes(appID string, routerGroup *gin.RouterGroup) error {
	// 获取App的所有插件
	plugins := m.ListPlugins(appID)
	for _, name := range plugins {
		// 获取SmartPlugin实例
		p, exists := m.GetSmartPlugin(appID, name)
		if !exists {
			continue // 不是SmartPlugin,跳过
		}

		// 创建插件路由组
		pluginGroup := routerGroup.Group(fmt.Sprintf("/%s", name))

		// 注册插件路由
		p.RegisterRoutes(pluginGroup)
	}
	return nil
}

// InstallPlugin 安装插件
func (m *manager) InstallPlugin(ctx context.Context, appID string, name string, config map[string]interface{}) error {
	// 创建插件实例
	p, err := m.createPlugin(name, appID)
	if err != nil {
		return fmt.Errorf("failed to create plugin: %v", err)
	}

	// 如果是SmartPlugin,调用OnInstall
	if sp, ok := p.(types.SmartPlugin); ok {
		if err := sp.OnInstall(appID); err != nil {
			return fmt.Errorf("failed to install plugin: %v", err)
		}
	}

	// 加载插件
	return m.LoadPlugin(appID, p, config)
}

// UninstallPlugin 卸载插件
func (m *manager) UninstallPlugin(ctx context.Context, appID string, name string) error {
	// 获取SmartPlugin实例
	p, exists := m.GetSmartPlugin(appID, name)
	if exists {
		// 调用OnUninstall
		if err := p.OnUninstall(appID); err != nil {
			return fmt.Errorf("failed to uninstall plugin: %v", err)
		}
	}

	// 卸载插件
	return m.UnloadPlugin(appID, name)
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

	// 启动插件
	if err := p.Start(); err != nil {
		return fmt.Errorf("failed to start plugin %s for app %s: %v", name, appID, err)
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

	// 注销插件服务
	if err := m.container.UnregisterPluginServices(name); err != nil {
		log.Printf("Warning: failed to unregister plugin services: %v", err)
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
	p, err := m.createPlugin(name, appID)
	if err != nil {
		log.Printf("Failed to create plugin %s: %v", name, err)
		return nil, false
	}

	// 加载插件配置
	if err := p.Load(config.Config); err != nil {
		log.Printf("Failed to load plugin %s for app %s: %v", name, appID, err)
		return nil, false
	}

	// 启动插件
	if err := p.Start(); err != nil {
		log.Printf("Failed to start plugin %s for app %s: %v", name, appID, err)
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

	executable, ok := p.(types.Executable)
	if !ok {
		return fmt.Errorf("plugin %s does not implement Executable interface", name)
	}

	return executable.Execute(ctx, params)
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
			p, err := m.createPlugin(config.Name, appID)
			if err != nil {
				log.Printf("Failed to create plugin %s: %v", config.Name, err)
				continue
			}

			// 加载插件
			if err := p.Load(config.Config); err != nil {
				log.Printf("Failed to load plugin %s for app %s: %v", config.Name, appID, err)
				continue
			}

			// 启动插件
			if err := p.Start(); err != nil {
				log.Printf("Failed to start plugin %s for app %s: %v", config.Name, appID, err)
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

// SavePluginConfig 保存插件配置
func (m *manager) SavePluginConfig(ctx context.Context, config *model.PluginConfig) error {
	// 先卸载现有插件
	if plugin, exists := m.GetPlugin(config.AppID, config.Name); exists {
		// 先停止插件
		if err := plugin.Stop(); err != nil {
			return fmt.Errorf("stop plugin failed: %w", err)
		}

		// 再卸载插件
		if err := plugin.Unload(); err != nil {
			return fmt.Errorf("unload plugin failed: %w", err)
		}

		// 从内存中移除插件
		if appPluginMapValue, exists := m.appPlugins.Load(config.AppID); exists {
			appPluginMap := appPluginMapValue.(*sync.Map)
			appPluginMap.Delete(config.Name)
		}
	}

	// 保存新配置
	if err := m.configRepo.SaveConfig(ctx, config); err != nil {
		return fmt.Errorf("save plugin config failed: %w", err)
	}

	// 创建新的插件实例
	plugin, err := m.createPlugin(config.Name, config.AppID)
	if err != nil {
		return fmt.Errorf("create plugin failed: %w", err)
	}

	// 加载新配置
	if err := m.LoadPlugin(config.AppID, plugin, config.Config); err != nil {
		return fmt.Errorf("reload plugin config failed: %w", err)
	}

	return nil
}
