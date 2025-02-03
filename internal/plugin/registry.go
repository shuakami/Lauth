package plugin

import (
	"fmt"
	"sync"

	"lauth/internal/plugin/types"
)

// registry 插件注册表实现
type registry struct {
	plugins sync.Map // 存储插件描述符
}

// NewRegistry 创建注册表实例
func NewRegistry() types.PluginRegistry {
	return &registry{}
}

// Register 注册插件
func (r *registry) Register(descriptor *types.PluginDescriptor) error {
	if descriptor == nil {
		return fmt.Errorf("plugin descriptor cannot be nil")
	}

	if descriptor.Name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if descriptor.Factory == nil {
		return fmt.Errorf("plugin factory cannot be nil")
	}

	// 检查插件名称是否已存在
	if _, exists := r.plugins.Load(descriptor.Name); exists {
		return fmt.Errorf("plugin %s already registered", descriptor.Name)
	}

	// 创建临时实例验证元数据
	p := descriptor.Factory()
	if p == nil {
		return fmt.Errorf("plugin factory returned nil")
	}

	metadata := p.GetMetadata()
	if metadata == nil {
		return fmt.Errorf("plugin metadata cannot be nil")
	}

	// 验证元数据
	if metadata.Name != descriptor.Name {
		return fmt.Errorf("plugin name mismatch: %s != %s", metadata.Name, descriptor.Name)
	}

	if metadata.Version != descriptor.Version {
		return fmt.Errorf("plugin version mismatch: %s != %s", metadata.Version, descriptor.Version)
	}

	// 存储描述符
	r.plugins.Store(descriptor.Name, descriptor)
	return nil
}

// Unregister 注销插件
func (r *registry) Unregister(name string) error {
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if _, exists := r.plugins.Load(name); !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	r.plugins.Delete(name)
	return nil
}

// Get 获取插件描述符
func (r *registry) Get(name string) (*types.PluginDescriptor, bool) {
	if name == "" {
		return nil, false
	}

	if value, exists := r.plugins.Load(name); exists {
		return value.(*types.PluginDescriptor), true
	}
	return nil, false
}

// List 列出所有已注册的插件
func (r *registry) List() []*types.PluginDescriptor {
	var descriptors []*types.PluginDescriptor

	r.plugins.Range(func(key, value interface{}) bool {
		descriptors = append(descriptors, value.(*types.PluginDescriptor))
		return true
	})

	return descriptors
}
