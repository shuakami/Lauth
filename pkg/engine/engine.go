package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lauth/internal/model"
	"lauth/internal/repository"
)

// Engine 规则引擎接口
type Engine interface {
	// Evaluate 评估规则
	Evaluate(ctx context.Context, appID string, data map[string]interface{}) (*Result, error)
	// EvaluateRule 评估单个规则
	EvaluateRule(ctx context.Context, rule *model.Rule, data map[string]interface{}) (*Result, error)
	// LoadRules 加载规则
	LoadRules(ctx context.Context, appID string) error
	// InvalidateCache 使规则缓存失效
	InvalidateCache(ctx context.Context, appID string) error
}

// Parser 规则解析器接口
type Parser interface {
	// Parse 解析规则条件
	Parse(conditions []model.RuleCondition) (Expression, error)
}

// Executor 规则执行器接口
type Executor interface {
	// Execute 执行规则
	Execute(ctx context.Context, expr Expression, data map[string]interface{}) (bool, error)
}

// Cache 规则缓存接口
type Cache interface {
	// Get 获取规则
	Get(ctx context.Context, appID string) ([]*model.Rule, error)
	// Set 设置规则
	Set(ctx context.Context, appID string, rules []*model.Rule, expiration time.Duration) error
	// Delete 删除规则
	Delete(ctx context.Context, appID string) error
}

// Expression 规则表达式接口
type Expression interface {
	// Evaluate 评估表达式
	Evaluate(data map[string]interface{}) (bool, error)
	// String 返回表达式的字符串表示
	String() string
}

// Result 规则执行结果
type Result struct {
	Allowed bool           `json:"allowed"` // 是否允许
	Rule    *model.Rule    `json:"rule"`    // 匹配的规则
	Data    map[string]any `json:"data"`    // 评估的数据
	Time    time.Time      `json:"time"`    // 评估时间
	Error   string         `json:"error"`   // 错误信息
}

// MarshalJSON 实现json.Marshaler接口
func (r *Result) MarshalJSON() ([]byte, error) {
	type Alias Result
	return json.Marshal(&struct {
		*Alias
		Time string `json:"time"`
	}{
		Alias: (*Alias)(r),
		Time:  r.Time.Format(time.RFC3339),
	})
}

// engine 规则引擎实现
type engine struct {
	parser   Parser
	executor Executor
	cache    Cache
	repo     repository.RuleRepository
}

// NewEngine 创建规则引擎实例
func NewEngine(parser Parser, executor Executor, cache Cache, repo repository.RuleRepository) Engine {
	return &engine{
		parser:   parser,
		executor: executor,
		cache:    cache,
		repo:     repo,
	}
}

// Evaluate 评估规则
func (e *engine) Evaluate(ctx context.Context, appID string, data map[string]interface{}) (*Result, error) {
	// 从缓存获取规则
	rules, err := e.cache.Get(ctx, appID)
	if err != nil {
		// 如果缓存未命中，从数据库加载规则
		if err := e.LoadRules(ctx, appID); err != nil {
			return nil, err
		}
		rules, err = e.cache.Get(ctx, appID)
		if err != nil {
			return nil, err
		}
	}

	// 按优先级排序并评估规则
	for _, rule := range rules {
		if !rule.IsEnabled {
			continue
		}

		result, err := e.EvaluateRule(ctx, rule, data)
		if err != nil {
			continue
		}

		if result.Allowed {
			return result, nil
		}
	}

	// 如果没有规则匹配，默认拒绝
	return &Result{
		Allowed: false,
		Time:    time.Now(),
		Data:    data,
	}, nil
}

// EvaluateRule 评估单个规则
func (e *engine) EvaluateRule(ctx context.Context, rule *model.Rule, data map[string]interface{}) (*Result, error) {
	// 解析规则条件
	expr, err := e.parser.Parse(rule.Conditions)
	if err != nil {
		return &Result{
			Allowed: false,
			Rule:    rule,
			Time:    time.Now(),
			Data:    data,
			Error:   err.Error(),
		}, err
	}

	// 执行规则
	allowed, err := e.executor.Execute(ctx, expr, data)
	if err != nil {
		return &Result{
			Allowed: false,
			Rule:    rule,
			Time:    time.Now(),
			Data:    data,
			Error:   err.Error(),
		}, err
	}

	return &Result{
		Allowed: allowed,
		Rule:    rule,
		Time:    time.Now(),
		Data:    data,
	}, nil
}

// LoadRules 加载规则
func (e *engine) LoadRules(ctx context.Context, appID string) error {
	// 从数据库加载启用的规则
	rules, err := e.repo.GetActiveRules(ctx, appID)
	if err != nil {
		return fmt.Errorf("failed to load rules from database: %v", err)
	}

	// 加载规则的条件
	rulePointers := make([]*model.Rule, len(rules))
	for i := range rules {
		conditions, err := e.repo.GetConditions(ctx, rules[i].ID)
		if err != nil {
			return fmt.Errorf("failed to load conditions for rule %s: %v", rules[i].ID, err)
		}
		rules[i].Conditions = conditions
		rulePointers[i] = &rules[i]
	}

	// 缓存规则
	if err := e.cache.Set(ctx, appID, rulePointers, 0); err != nil {
		return fmt.Errorf("failed to cache rules: %v", err)
	}

	return nil
}

// InvalidateCache 使规则缓存失效
func (e *engine) InvalidateCache(ctx context.Context, appID string) error {
	return e.cache.Delete(ctx, appID)
}
