package types

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
