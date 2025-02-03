package container

import (
	"fmt"
	"sync"
)

// PluginContainer 插件容器接口
type PluginContainer interface {
	Container // 继承基本容器接口

	// RegisterPluginService 注册插件服务
	// pluginName: 插件名称
	// serviceName: 服务名称
	// service: 服务实例或工厂函数
	RegisterPluginService(pluginName, serviceName string, service interface{}) error

	// ResolvePluginService 解析插件服务
	// pluginName: 插件名称
	// serviceName: 服务名称
	ResolvePluginService(pluginName, serviceName string) (interface{}, error)

	// GetPluginServices 获取插件的所有服务
	// pluginName: 插件名称
	GetPluginServices(pluginName string) map[string]interface{}

	// UnregisterPluginServices 注销插件的所有服务
	// pluginName: 插件名称
	UnregisterPluginServices(pluginName string) error
}

// pluginContainer 插件容器实现
type pluginContainer struct {
	*container                  // 继承基本容器
	pluginServices sync.Map     // 插件服务映射 map[pluginName]map[serviceName]serviceEntry
	mu             sync.RWMutex // 用于保护pluginServices的并发访问
}

// NewPluginContainer 创建插件容器实例
func NewPluginContainer() PluginContainer {
	return &pluginContainer{
		container: NewContainer().(*container),
	}
}

// RegisterPluginService 注册插件服务
func (c *pluginContainer) RegisterPluginService(pluginName, serviceName string, service interface{}) error {
	if pluginName == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if serviceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if service == nil {
		return fmt.Errorf("service cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 获取或创建插件的服务映射
	var pluginServicesMap *sync.Map
	if value, ok := c.pluginServices.Load(pluginName); ok {
		pluginServicesMap = value.(*sync.Map)
	} else {
		pluginServicesMap = &sync.Map{}
		c.pluginServices.Store(pluginName, pluginServicesMap)
	}

	// 创建服务条目
	var factory ServiceFactory
	if f, ok := service.(ServiceFactory); ok {
		factory = f
	} else {
		instance := service
		factory = func() interface{} {
			return instance
		}
	}

	entry := &serviceEntry{
		factory:   factory,
		singleton: true, // 插件服务默认使用单例模式
	}

	// 存储服务条目
	pluginServicesMap.Store(serviceName, entry)
	return nil
}

// ResolvePluginService 解析插件服务
func (c *pluginContainer) ResolvePluginService(pluginName, serviceName string) (interface{}, error) {
	if pluginName == "" {
		return nil, fmt.Errorf("plugin name cannot be empty")
	}
	if serviceName == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// 获取插件的服务映射
	value, ok := c.pluginServices.Load(pluginName)
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}
	pluginServicesMap := value.(*sync.Map)

	// 获取服务条目
	value, ok = pluginServicesMap.Load(serviceName)
	if !ok {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}
	entry := value.(*serviceEntry)

	// 单例模式
	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.instance == nil {
		entry.instance = entry.factory()
	}
	return entry.instance, nil
}

// GetPluginServices 获取插件的所有服务
func (c *pluginContainer) GetPluginServices(pluginName string) map[string]interface{} {
	if pluginName == "" {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	services := make(map[string]interface{})

	// 获取插件的服务映射
	value, ok := c.pluginServices.Load(pluginName)
	if !ok {
		return services
	}
	pluginServicesMap := value.(*sync.Map)

	// 收集所有服务
	pluginServicesMap.Range(func(key, value interface{}) bool {
		serviceName := key.(string)
		entry := value.(*serviceEntry)

		// 获取或创建服务实例
		entry.mu.Lock()
		if entry.instance == nil {
			entry.instance = entry.factory()
		}
		instance := entry.instance
		entry.mu.Unlock()

		services[serviceName] = instance
		return true
	})

	return services
}

// UnregisterPluginServices 注销插件的所有服务
func (c *pluginContainer) UnregisterPluginServices(pluginName string) error {
	if pluginName == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 删除插件的所有服务
	c.pluginServices.Delete(pluginName)
	return nil
}
