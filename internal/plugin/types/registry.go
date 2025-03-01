package types

import (
	"fmt"
)

// PluginDescriptor 插件描述符
type PluginDescriptor struct {
	Name         string          // 插件名称
	Version      string          // 插件版本
	Factory      PluginFactory   // 插件工厂函数
	Metadata     *PluginMetadata // 插件元数据
	Dependencies []string        // 依赖的服务
}

// PluginFactory 插件工厂函数类型
type PluginFactory func() Plugin

// PluginRegistry 插件注册表接口
type PluginRegistry interface {
	// Register 注册插件
	Register(descriptor *PluginDescriptor) error

	// Unregister 注销插件
	Unregister(name string) error

	// Get 获取插件描述符
	Get(name string) (*PluginDescriptor, bool)

	// List 列出所有已注册的插件
	List() []*PluginDescriptor
}

// globalRegistry 全局插件注册表
var globalRegistry []struct {
	Name    string
	Factory PluginFactory
}

// RegisterPlugin 注册插件到全局注册表
// 供init函数使用，实现自动注册
func RegisterPlugin(name string, factory PluginFactory) {
	globalRegistry = append(globalRegistry, struct {
		Name    string
		Factory PluginFactory
	}{
		Name:    name,
		Factory: factory,
	})
}

// GetRegisteredPlugins 获取所有已注册的插件
func GetRegisteredPlugins() []struct {
	Name    string
	Factory PluginFactory
} {
	// 添加调试日志
	fmt.Printf("已注册的插件数量: %d\n", len(globalRegistry))
	for i, plugin := range globalRegistry {
		fmt.Printf("插件 #%d: %s\n", i+1, plugin.Name)
	}
	return globalRegistry
}
