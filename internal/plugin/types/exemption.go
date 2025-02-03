package types

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ExemptionType 豁免类型
type ExemptionType string

// ExemptionResult 豁免检查结果
type ExemptionResult struct {
	Exempt   bool        // 是否豁免
	Reason   string      // 豁免原因
	Metadata interface{} // 额外元数据
}

// ExemptionRule 豁免规则接口
type ExemptionRule interface {
	// Type 返回规则类型
	Type() ExemptionType

	// Priority 返回规则优先级(0-100,越大优先级越高)
	Priority() int

	// Match 检查是否匹配
	Match(ctx context.Context, value interface{}, userExempts map[string]interface{}, globalExempts map[string]interface{}) (*ExemptionResult, error)
}

// ExemptionMatcher 豁免匹配器
type ExemptionMatcher interface {
	// Match 执行匹配
	Match(value interface{}, pattern interface{}) (bool, error)
}

// BaseRule 基础规则实现
type BaseRule struct {
	ruleType    ExemptionType
	priority    int
	matcher     ExemptionMatcher
	description string
}

// Type 获取规则类型
func (r *BaseRule) Type() ExemptionType {
	return r.ruleType
}

// Priority 获取优先级
func (r *BaseRule) Priority() int {
	return r.priority
}

// NewBaseRule 创建基础规则
func NewBaseRule(ruleType ExemptionType, priority int, matcher ExemptionMatcher, description string) *BaseRule {
	return &BaseRule{
		ruleType:    ruleType,
		priority:    priority,
		matcher:     matcher,
		description: description,
	}
}

// ExemptionManager 豁免管理器
type ExemptionManager struct {
	mu         sync.RWMutex
	rules      map[ExemptionType][]ExemptionRule
	middleware []ExemptionMiddleware
}

// ExemptionMiddleware 豁免中间件
type ExemptionMiddleware func(next ExemptionHandler) ExemptionHandler

// ExemptionHandler 豁免处理函数
type ExemptionHandler func(ctx context.Context, ruleType ExemptionType, value interface{}, userExempts map[string]interface{}, globalExempts map[string]interface{}) (*ExemptionResult, error)

// NewExemptionManager 创建豁免管理器
func NewExemptionManager() *ExemptionManager {
	return &ExemptionManager{
		rules: make(map[ExemptionType][]ExemptionRule),
	}
}

// AddRule 添加豁免规则
func (m *ExemptionManager) AddRule(rule ExemptionRule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ruleType := rule.Type()
	rules := m.rules[ruleType]
	rules = append(rules, rule)

	// 按优先级排序
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority() > rules[j].Priority()
	})

	m.rules[ruleType] = rules
}

// AddMiddleware 添加中间件
func (m *ExemptionManager) AddMiddleware(middleware ExemptionMiddleware) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.middleware = append(m.middleware, middleware)
}

// CheckExemption 检查豁免
func (m *ExemptionManager) CheckExemption(ctx context.Context, ruleType ExemptionType, value interface{}, userExempts map[string]interface{}, globalExempts map[string]interface{}) (*ExemptionResult, error) {
	m.mu.RLock()
	rules := m.rules[ruleType]
	middleware := m.middleware
	m.mu.RUnlock()

	if len(rules) == 0 {
		return &ExemptionResult{Exempt: false}, nil
	}

	// 构建处理链
	handler := func(ctx context.Context, ruleType ExemptionType, value interface{}, userExempts map[string]interface{}, globalExempts map[string]interface{}) (*ExemptionResult, error) {
		// 按优先级检查规则
		for _, rule := range rules {
			result, err := rule.Match(ctx, value, userExempts, globalExempts)
			if err != nil {
				return nil, fmt.Errorf("rule match failed: %v", err)
			}
			if result.Exempt {
				return result, nil
			}
		}
		return &ExemptionResult{Exempt: false}, nil
	}

	// 应用中间件
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}

	return handler(ctx, ruleType, value, userExempts, globalExempts)
}

// 预定义的豁免类型
const (
	ExemptionTypeIP     ExemptionType = "ip"
	ExemptionTypeDevice ExemptionType = "device"
)

// ExactMatcher 精确匹配器
type ExactMatcher struct{}

// Match 执行精确匹配
func (m *ExactMatcher) Match(value interface{}, pattern interface{}) (bool, error) {
	return value == pattern, nil
}

// IPRule IP豁免规则
type IPRule struct {
	*BaseRule
}

// NewIPRule 创建IP豁免规则
func NewIPRule(priority int) ExemptionRule {
	return &IPRule{
		BaseRule: NewBaseRule(ExemptionTypeIP, priority, &ExactMatcher{}, "IP exact match"),
	}
}

// Match 实现IP匹配
func (r *IPRule) Match(ctx context.Context, value interface{}, userExempts map[string]interface{}, globalExempts map[string]interface{}) (*ExemptionResult, error) {
	ip, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("value is not a string")
	}

	// 检查用户豁免
	if userIPs, ok := userExempts["exempt_ips"].([]interface{}); ok {
		for _, exemptIP := range userIPs {
			if ipStr, ok := exemptIP.(string); ok {
				if match, _ := r.matcher.Match(ip, ipStr); match {
					return &ExemptionResult{
						Exempt: true,
						Reason: "user ip exempt",
						Metadata: map[string]interface{}{
							"type": "user",
							"ip":   ip,
						},
					}, nil
				}
			}
		}
	}

	// 检查全局豁免
	if globalIPs, ok := globalExempts["exempt_ips"].([]interface{}); ok {
		for _, exemptIP := range globalIPs {
			if ipStr, ok := exemptIP.(string); ok {
				if match, _ := r.matcher.Match(ip, ipStr); match {
					return &ExemptionResult{
						Exempt: true,
						Reason: "global ip exempt",
						Metadata: map[string]interface{}{
							"type": "global",
							"ip":   ip,
						},
					}, nil
				}
			}
		}
	}

	return &ExemptionResult{Exempt: false}, nil
}

// DeviceRule 设备豁免规则
type DeviceRule struct {
	*BaseRule
}

// NewDeviceRule 创建设备豁免规则
func NewDeviceRule(priority int) ExemptionRule {
	return &DeviceRule{
		BaseRule: NewBaseRule(ExemptionTypeDevice, priority, &ExactMatcher{}, "Device exact match"),
	}
}

// Match 实现设备匹配
func (r *DeviceRule) Match(ctx context.Context, value interface{}, userExempts map[string]interface{}, globalExempts map[string]interface{}) (*ExemptionResult, error) {
	deviceID, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("value is not a string")
	}

	// 如果设备ID为空,直接返回不豁免
	if deviceID == "" {
		return &ExemptionResult{
			Exempt: false,
			Reason: "empty device id",
		}, nil
	}

	// 检查用户豁免
	if userDevices, ok := userExempts["exempt_devices"].([]interface{}); ok {
		for _, exemptDevice := range userDevices {
			if deviceStr, ok := exemptDevice.(string); ok {
				if match, _ := r.matcher.Match(deviceID, deviceStr); match {
					return &ExemptionResult{
						Exempt: true,
						Reason: "user device exempt",
						Metadata: map[string]interface{}{
							"type":   "user",
							"device": deviceID,
						},
					}, nil
				}
			}
		}
	}

	// 检查全局豁免
	if globalDevices, ok := globalExempts["exempt_devices"].([]interface{}); ok {
		for _, exemptDevice := range globalDevices {
			if deviceStr, ok := exemptDevice.(string); ok {
				if match, _ := r.matcher.Match(deviceID, deviceStr); match {
					return &ExemptionResult{
						Exempt: true,
						Reason: "global device exempt",
						Metadata: map[string]interface{}{
							"type":   "global",
							"device": deviceID,
						},
					}, nil
				}
			}
		}
	}

	return &ExemptionResult{Exempt: false}, nil
}

// LoggingMiddleware 日志中间件
func LoggingMiddleware(logger Logger) ExemptionMiddleware {
	return func(next ExemptionHandler) ExemptionHandler {
		return func(ctx context.Context, ruleType ExemptionType, value interface{}, userExempts map[string]interface{}, globalExempts map[string]interface{}) (*ExemptionResult, error) {
			logger.Debug("checking exemption",
				"type", ruleType,
				"value", value,
			)

			result, err := next(ctx, ruleType, value, userExempts, globalExempts)

			if err != nil {
				logger.Error("exemption check failed",
					"type", ruleType,
					"value", value,
					"error", err,
				)
			} else {
				logger.Debug("exemption check result",
					"type", ruleType,
					"value", value,
					"exempt", result.Exempt,
					"reason", result.Reason,
				)
			}

			return result, err
		}
	}
}
