package container

import (
	"fmt"
	"sync"
)

// ServiceFactory 服务工厂函数类型
type ServiceFactory func() interface{}

// Container 依赖注入容器接口
type Container interface {
	// Register 注册服务
	// name: 服务名称
	// service: 服务实例或工厂函数
	// singleton: 是否单例
	Register(name string, service interface{}, singleton bool) error

	// Resolve 解析服务
	// name: 服务名称
	// 返回服务实例
	Resolve(name string) (interface{}, error)

	// Has 检查服务是否存在
	Has(name string) bool
}

// serviceEntry 服务条目
type serviceEntry struct {
	instance  interface{}    // 服务实例（单例模式）
	factory   ServiceFactory // 服务工厂（工厂模式）
	singleton bool           // 是否单例
	mu        sync.Mutex     // 互斥锁（用于单例模式）
}

// container 依赖注入容器实现
type container struct {
	services sync.Map // 服务映射
}

// NewContainer 创建容器实例
func NewContainer() Container {
	return &container{}
}

// Register 注册服务
func (c *container) Register(name string, service interface{}, singleton bool) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if service == nil {
		return fmt.Errorf("service cannot be nil")
	}

	var factory ServiceFactory
	if f, ok := service.(ServiceFactory); ok {
		// 如果service是工厂函数，直接使用
		factory = f
	} else {
		// 否则创建返回固定实例的工厂函数
		instance := service
		factory = func() interface{} {
			return instance
		}
	}

	entry := &serviceEntry{
		factory:   factory,
		singleton: singleton,
	}

	c.services.Store(name, entry)
	return nil
}

// Resolve 解析服务
func (c *container) Resolve(name string) (interface{}, error) {
	if name == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}

	value, ok := c.services.Load(name)
	if !ok {
		return nil, fmt.Errorf("service not found: %s", name)
	}

	entry := value.(*serviceEntry)

	if entry.singleton {
		// 单例模式
		entry.mu.Lock()
		defer entry.mu.Unlock()

		if entry.instance == nil {
			entry.instance = entry.factory()
		}
		return entry.instance, nil
	}

	// 工厂模式
	return entry.factory(), nil
}

// Has 检查服务是否存在
func (c *container) Has(name string) bool {
	_, ok := c.services.Load(name)
	return ok
}
