package types

import (
	"context"
	"fmt"
)

// PluginHooks 插件钩子接口
type PluginHooks interface {
	// OnLoad 加载时的钩子
	OnLoad(config map[string]interface{}) error

	// OnUnload 卸载时的钩子
	OnUnload() error

	// OnStart 启动时的钩子
	OnStart() error

	// OnStop 停止时的钩子
	OnStop() error

	// OnExecute 执行插件逻辑时的钩子
	OnExecute(ctx context.Context, params map[string]interface{}) error

	// OnExecuteComplete 执行完成时的钩子
	OnExecuteComplete(ctx context.Context, err error) error
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	// HandleError 处理错误
	HandleError(err error) error
}

// ValidateFunc 验证函数类型
type ValidateFunc func(config map[string]interface{}) error

// ConfigValidationError 配置验证错误
type ConfigValidationError struct {
	Field string // 字段名
	Msg   string // 错误信息
}

// Error 实现error接口
func (e *ConfigValidationError) Error() string {
	return "config validation error: " + e.Field + " " + e.Msg
}

// ConfigValidator 配置验证器接口
type ConfigValidator interface {
	// Validate 验证配置
	Validate(config map[string]interface{}) error
}

// BaseConfigValidator 基础配置验证器
type BaseConfigValidator struct {
	validators []ValidateFunc
}

// NewBaseConfigValidator 创建基础配置验证器
func NewBaseConfigValidator(validators ...ValidateFunc) *BaseConfigValidator {
	return &BaseConfigValidator{
		validators: validators,
	}
}

// Validate 验证配置
func (v *BaseConfigValidator) Validate(config map[string]interface{}) error {
	for _, validator := range v.validators {
		if err := validator(config); err != nil {
			return err
		}
	}
	return nil
}

// AddValidator 添加验证函数
func (v *BaseConfigValidator) AddValidator(validator ValidateFunc) {
	v.validators = append(v.validators, validator)
}

// RequiredValidator 必需字段验证器
func RequiredValidator(fields ...string) ValidateFunc {
	return func(config map[string]interface{}) error {
		for _, field := range fields {
			if _, ok := config[field]; !ok {
				return &ConfigValidationError{
					Field: field,
					Msg:   "field is required",
				}
			}
		}
		return nil
	}
}

// TypeValidator 类型验证器
func TypeValidator(field string, expectedType interface{}) ValidateFunc {
	return func(config map[string]interface{}) error {
		value, ok := config[field]
		if !ok {
			return nil
		}

		fmt.Printf("Validating field %s: value=%v (type=%T), expected type=%T\n", field, value, value, expectedType)

		switch expectedType.(type) {
		case string:
			if _, ok := value.(string); !ok {
				return &ConfigValidationError{
					Field: field,
					Msg:   "field must be a string",
				}
			}
		case int:
			switch v := value.(type) {
			case int:
				// OK
			case float64:
				// 尝试转换float64到int
				if float64(int(v)) == v {
					// 是整数
					config[field] = int(v) // 更新为int类型
				} else {
					return &ConfigValidationError{
						Field: field,
						Msg:   "field must be an integer",
					}
				}
			default:
				return &ConfigValidationError{
					Field: field,
					Msg:   "field must be an integer",
				}
			}
		case bool:
			if _, ok := value.(bool); !ok {
				return &ConfigValidationError{
					Field: field,
					Msg:   "field must be a boolean",
				}
			}
		case []interface{}:
			if _, ok := value.([]interface{}); !ok {
				return &ConfigValidationError{
					Field: field,
					Msg:   "field must be an array",
				}
			}
		case map[string]interface{}:
			if _, ok := value.(map[string]interface{}); !ok {
				return &ConfigValidationError{
					Field: field,
					Msg:   "field must be an object",
				}
			}
		}
		return nil
	}
}

// PluginMiddleware 插件中间件接口
type PluginMiddleware interface {
	// Before 执行前的处理
	Before(ctx map[string]interface{}) error

	// After 执行后的处理
	After(ctx map[string]interface{}) error

	// OnError 错误处理
	OnError(ctx map[string]interface{}, err error) error
}

// Logger 日志接口
type Logger interface {
	// Debug 调试日志
	Debug(msg string, args ...interface{})

	// Info 信息日志
	Info(msg string, args ...interface{})

	// Warn 警告日志
	Warn(msg string, args ...interface{})

	// Error 错误日志
	Error(msg string, args ...interface{})
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	// Counter 计数器
	Counter(name string, labels map[string]string) error

	// Gauge 仪表盘
	Gauge(name string, value float64, labels map[string]string) error

	// Histogram 直方图
	Histogram(name string, value float64, labels map[string]string) error
}

// Storage 存储接口
type Storage interface {
	// Get 获取数据
	Get(key string) (interface{}, error)

	// Set 设置数据
	Set(key string, value interface{}) error

	// Delete 删除数据
	Delete(key string) error

	// List 列出数据
	List(prefix string) ([]string, error)
}

// EventEmitter 事件发射器接口
type EventEmitter interface {
	// Emit 发送事件
	Emit(event string, data interface{}) error

	// On 注册事件处理器
	On(event string, handler func(data interface{}) error)

	// Off 注销事件处理器
	Off(event string)
}

// BaseHooks 基础钩子实现
type BaseHooks struct{}

// OnLoad 加载时的钩子
func (h *BaseHooks) OnLoad(config map[string]interface{}) error {
	return nil
}

// OnUnload 卸载时的钩子
func (h *BaseHooks) OnUnload() error {
	return nil
}

// OnStart 启动时的钩子
func (h *BaseHooks) OnStart() error {
	return nil
}

// OnStop 停止时的钩子
func (h *BaseHooks) OnStop() error {
	return nil
}

// OnExecute 执行插件逻辑时的钩子
func (h *BaseHooks) OnExecute(ctx context.Context, params map[string]interface{}) error {
	return nil
}

// OnExecuteComplete 执行完成时的钩子
func (h *BaseHooks) OnExecuteComplete(ctx context.Context, err error) error {
	return nil
}

// BaseErrorHandler 基础错误处理器实现
type BaseErrorHandler struct{}

// HandleError 处理错误
func (h *BaseErrorHandler) HandleError(err error) error {
	return err
}

// BaseMiddleware 基础中间件实现
type BaseMiddleware struct{}

// Before 执行前的处理
func (m *BaseMiddleware) Before(ctx map[string]interface{}) error {
	return nil
}

// After 执行后的处理
func (m *BaseMiddleware) After(ctx map[string]interface{}) error {
	return nil
}

// OnError 错误处理
func (m *BaseMiddleware) OnError(ctx map[string]interface{}, err error) error {
	return err
}
