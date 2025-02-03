package types

import (
	"context"
)

// PluginHooks 插件钩子接口
type PluginHooks interface {
	// 生命周期钩子
	OnLoad(config map[string]interface{}) error         // 加载时
	OnUnload() error                                    // 卸载时
	OnStart() error                                     // 启动时
	OnStop() error                                      // 停止时
	OnConfigUpdate(config map[string]interface{}) error // 配置更新时

	// 执行钩子
	OnExecute(ctx context.Context, params map[string]interface{}) error // 执行时
	OnExecuteComplete(ctx context.Context, err error) error             // 执行完成时

	// 验证钩子
	OnValidate(ctx context.Context, userID string, action string) error                       // 验证时
	OnValidateComplete(ctx context.Context, userID string, action string, success bool) error // 验证完成时

	// 状态钩子
	OnStateChange(oldState, newState PluginState) error // 状态变更时
	OnError(err error) error                            // 发生错误时

	// 资源钩子
	OnResourceCreate(ctx context.Context, resource interface{}) error                 // 资源创建时
	OnResourceUpdate(ctx context.Context, oldResource, newResource interface{}) error // 资源更新时
	OnResourceDelete(ctx context.Context, resource interface{}) error                 // 资源删除时
}

// BaseHooks 基础钩子实现
type BaseHooks struct{}

// OnLoad 加载时
func (h *BaseHooks) OnLoad(config map[string]interface{}) error { return nil }

// OnUnload 卸载时
func (h *BaseHooks) OnUnload() error { return nil }

// OnStart 启动时
func (h *BaseHooks) OnStart() error { return nil }

// OnStop 停止时
func (h *BaseHooks) OnStop() error { return nil }

// OnConfigUpdate 配置更新时
func (h *BaseHooks) OnConfigUpdate(config map[string]interface{}) error { return nil }

// OnExecute 执行时
func (h *BaseHooks) OnExecute(ctx context.Context, params map[string]interface{}) error { return nil }

// OnExecuteComplete 执行完成时
func (h *BaseHooks) OnExecuteComplete(ctx context.Context, err error) error { return nil }

// OnValidate 验证时
func (h *BaseHooks) OnValidate(ctx context.Context, userID string, action string) error { return nil }

// OnValidateComplete 验证完成时
func (h *BaseHooks) OnValidateComplete(ctx context.Context, userID string, action string, success bool) error {
	return nil
}

// OnStateChange 状态变更时
func (h *BaseHooks) OnStateChange(oldState, newState PluginState) error { return nil }

// OnError 发生错误时
func (h *BaseHooks) OnError(err error) error { return nil }

// OnResourceCreate 资源创建时
func (h *BaseHooks) OnResourceCreate(ctx context.Context, resource interface{}) error { return nil }

// OnResourceUpdate 资源更新时
func (h *BaseHooks) OnResourceUpdate(ctx context.Context, oldResource, newResource interface{}) error {
	return nil
}

// OnResourceDelete 资源删除时
func (h *BaseHooks) OnResourceDelete(ctx context.Context, resource interface{}) error { return nil }

// PluginMiddleware 插件中间件函数类型
type PluginMiddleware func(next ExecuteFunc) ExecuteFunc

// ExecuteFunc 执行函数类型
type ExecuteFunc func(ctx context.Context, params map[string]interface{}) error

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	Handle(err error)
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	Counter(name string, value float64, labels map[string]string)
	Gauge(name string, value float64, labels map[string]string)
	Histogram(name string, value float64, labels map[string]string)
}

// Storage 存储接口
type Storage interface {
	Get(key string) (interface{}, error)
	Set(key string, value interface{}) error
	Delete(key string) error
}

// EventEmitter 事件发射器接口
type EventEmitter interface {
	Emit(event string, data interface{})
	On(event string, handler func(data interface{}))
	Off(event string, handler func(data interface{}))
}
