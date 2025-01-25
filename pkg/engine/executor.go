package engine

import (
	"context"
	"fmt"
)

// executor 规则执行器实现
type executor struct{}

// NewExecutor 创建规则执行器实例
func NewExecutor() Executor {
	return &executor{}
}

// Execute 执行规则
func (e *executor) Execute(ctx context.Context, expr Expression, data map[string]interface{}) (bool, error) {
	select {
	case <-ctx.Done():
		return false, fmt.Errorf("execution cancelled: %v", ctx.Err())
	default:
		return expr.Evaluate(data)
	}
}
