package types

import (
	"fmt"
)

// ConfigValidator 配置验证器接口
type ConfigValidator interface {
	// Validate 验证配置
	Validate(config map[string]interface{}) error
}

// ConfigManager 配置管理器接口
type ConfigManager interface {
	// Load 加载配置
	Load() (map[string]interface{}, error)

	// Save 保存配置
	Save(config map[string]interface{}) error

	// Watch 监听配置变更
	Watch() (<-chan ConfigEvent, error)

	// Close 关闭配置管理器
	Close() error
}

// ConfigEvent 配置事件
type ConfigEvent struct {
	// Type 事件类型
	Type ConfigEventType

	// OldConfig 旧配置
	OldConfig map[string]interface{}

	// NewConfig 新配置
	NewConfig map[string]interface{}
}

// ConfigEventType 配置事件类型
type ConfigEventType int

const (
	// ConfigEventUpdate 配置更新
	ConfigEventUpdate ConfigEventType = iota

	// ConfigEventDelete 配置删除
	ConfigEventDelete
)

// BaseConfigValidator 基础配置验证器
type BaseConfigValidator struct {
	// validators 验证函数列表
	validators []ValidateFunc
}

// ValidateFunc 验证函数类型
type ValidateFunc func(config map[string]interface{}) error

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

// ConfigValidationError 配置验证错误
type ConfigValidationError struct {
	Field string // 字段名
	Msg   string // 错误信息
}

// Error 实现error接口
func (e *ConfigValidationError) Error() string {
	return "config validation error: " + e.Field + " " + e.Msg
}
