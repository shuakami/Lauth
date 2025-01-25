package engine

import (
	"fmt"
	"strings"

	"lauth/internal/model"
)

// parser 规则解析器实现
type parser struct{}

// NewParser 创建规则解析器实例
func NewParser() Parser {
	return &parser{}
}

// Parse 解析规则条件
func (p *parser) Parse(conditions []model.RuleCondition) (Expression, error) {
	if len(conditions) == 0 {
		return nil, fmt.Errorf("no conditions provided")
	}

	// 创建AND表达式，所有条件都必须满足
	exprs := make([]Expression, 0, len(conditions))
	for _, condition := range conditions {
		expr, err := p.parseCondition(condition)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, expr)
	}

	return &andExpression{exprs: exprs}, nil
}

// parseCondition 解析单个条件
func (p *parser) parseCondition(condition model.RuleCondition) (Expression, error) {
	switch condition.Operator {
	case model.OperatorEqual:
		return &equalExpression{field: condition.Field, value: condition.Value}, nil
	case model.OperatorNotEqual:
		return &notEqualExpression{field: condition.Field, value: condition.Value}, nil
	case model.OperatorGreaterThan:
		return &greaterThanExpression{field: condition.Field, value: condition.Value}, nil
	case model.OperatorGreaterThanOrEqual:
		return &greaterThanOrEqualExpression{field: condition.Field, value: condition.Value}, nil
	case model.OperatorLessThan:
		return &lessThanExpression{field: condition.Field, value: condition.Value}, nil
	case model.OperatorLessThanOrEqual:
		return &lessThanOrEqualExpression{field: condition.Field, value: condition.Value}, nil
	case model.OperatorIn:
		return &inExpression{field: condition.Field, value: condition.Value}, nil
	case model.OperatorNotIn:
		return &notInExpression{field: condition.Field, value: condition.Value}, nil
	case model.OperatorContains:
		return &containsExpression{field: condition.Field, value: condition.Value}, nil
	case model.OperatorNotContains:
		return &notContainsExpression{field: condition.Field, value: condition.Value}, nil
	default:
		return nil, fmt.Errorf("unsupported operator: %s", condition.Operator)
	}
}

// andExpression AND表达式
type andExpression struct {
	exprs []Expression
}

func (e *andExpression) Evaluate(data map[string]interface{}) (bool, error) {
	for _, expr := range e.exprs {
		result, err := expr.Evaluate(data)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

func (e *andExpression) String() string {
	exprs := make([]string, len(e.exprs))
	for i, expr := range e.exprs {
		exprs[i] = expr.String()
	}
	return fmt.Sprintf("(%s)", strings.Join(exprs, " AND "))
}

// equalExpression 等于表达式
type equalExpression struct {
	field string
	value interface{}
}

func (e *equalExpression) Evaluate(data map[string]interface{}) (bool, error) {
	value, ok := data[e.field]
	if !ok {
		return false, fmt.Errorf("field not found: %s", e.field)
	}
	return value == e.value, nil
}

func (e *equalExpression) String() string {
	return fmt.Sprintf("%s == %v", e.field, e.value)
}

// notEqualExpression 不等于表达式
type notEqualExpression struct {
	field string
	value interface{}
}

func (e *notEqualExpression) Evaluate(data map[string]interface{}) (bool, error) {
	value, ok := data[e.field]
	if !ok {
		return false, fmt.Errorf("field not found: %s", e.field)
	}
	return value != e.value, nil
}

func (e *notEqualExpression) String() string {
	return fmt.Sprintf("%s != %v", e.field, e.value)
}

// greaterThanExpression 大于表达式
type greaterThanExpression struct {
	field string
	value interface{}
}

func (e *greaterThanExpression) Evaluate(data map[string]interface{}) (bool, error) {
	value, ok := data[e.field]
	if !ok {
		return false, fmt.Errorf("field not found: %s", e.field)
	}
	return compareValues(value, e.value) > 0, nil
}

func (e *greaterThanExpression) String() string {
	return fmt.Sprintf("%s > %v", e.field, e.value)
}

// greaterThanOrEqualExpression 大于等于表达式
type greaterThanOrEqualExpression struct {
	field string
	value interface{}
}

func (e *greaterThanOrEqualExpression) Evaluate(data map[string]interface{}) (bool, error) {
	value, ok := data[e.field]
	if !ok {
		return false, fmt.Errorf("field not found: %s", e.field)
	}
	return compareValues(value, e.value) >= 0, nil
}

func (e *greaterThanOrEqualExpression) String() string {
	return fmt.Sprintf("%s >= %v", e.field, e.value)
}

// lessThanExpression 小于表达式
type lessThanExpression struct {
	field string
	value interface{}
}

func (e *lessThanExpression) Evaluate(data map[string]interface{}) (bool, error) {
	value, ok := data[e.field]
	if !ok {
		return false, fmt.Errorf("field not found: %s", e.field)
	}
	return compareValues(value, e.value) < 0, nil
}

func (e *lessThanExpression) String() string {
	return fmt.Sprintf("%s < %v", e.field, e.value)
}

// lessThanOrEqualExpression 小于等于表达式
type lessThanOrEqualExpression struct {
	field string
	value interface{}
}

func (e *lessThanOrEqualExpression) Evaluate(data map[string]interface{}) (bool, error) {
	value, ok := data[e.field]
	if !ok {
		return false, fmt.Errorf("field not found: %s", e.field)
	}
	return compareValues(value, e.value) <= 0, nil
}

func (e *lessThanOrEqualExpression) String() string {
	return fmt.Sprintf("%s <= %v", e.field, e.value)
}

// inExpression IN表达式
type inExpression struct {
	field string
	value interface{}
}

func (e *inExpression) Evaluate(data map[string]interface{}) (bool, error) {
	value, ok := data[e.field]
	if !ok {
		return false, fmt.Errorf("field not found: %s", e.field)
	}

	values, ok := e.value.([]interface{})
	if !ok {
		return false, fmt.Errorf("invalid IN values: %v", e.value)
	}

	for _, v := range values {
		if value == v {
			return true, nil
		}
	}
	return false, nil
}

func (e *inExpression) String() string {
	return fmt.Sprintf("%s IN %v", e.field, e.value)
}

// notInExpression NOT IN表达式
type notInExpression struct {
	field string
	value interface{}
}

func (e *notInExpression) Evaluate(data map[string]interface{}) (bool, error) {
	value, ok := data[e.field]
	if !ok {
		return false, fmt.Errorf("field not found: %s", e.field)
	}

	values, ok := e.value.([]interface{})
	if !ok {
		return false, fmt.Errorf("invalid NOT IN values: %v", e.value)
	}

	for _, v := range values {
		if value == v {
			return false, nil
		}
	}
	return true, nil
}

func (e *notInExpression) String() string {
	return fmt.Sprintf("%s NOT IN %v", e.field, e.value)
}

// containsExpression CONTAINS表达式
type containsExpression struct {
	field string
	value interface{}
}

func (e *containsExpression) Evaluate(data map[string]interface{}) (bool, error) {
	value, ok := data[e.field]
	if !ok {
		return false, fmt.Errorf("field not found: %s", e.field)
	}

	switch v := value.(type) {
	case string:
		return strings.Contains(v, fmt.Sprintf("%v", e.value)), nil
	case []interface{}:
		for _, item := range v {
			if item == e.value {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unsupported type for CONTAINS: %T", value)
	}
}

func (e *containsExpression) String() string {
	return fmt.Sprintf("%s CONTAINS %v", e.field, e.value)
}

// notContainsExpression NOT CONTAINS表达式
type notContainsExpression struct {
	field string
	value interface{}
}

func (e *notContainsExpression) Evaluate(data map[string]interface{}) (bool, error) {
	value, ok := data[e.field]
	if !ok {
		return false, fmt.Errorf("field not found: %s", e.field)
	}

	switch v := value.(type) {
	case string:
		return !strings.Contains(v, fmt.Sprintf("%v", e.value)), nil
	case []interface{}:
		for _, item := range v {
			if item == e.value {
				return false, nil
			}
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported type for NOT CONTAINS: %T", value)
	}
}

func (e *notContainsExpression) String() string {
	return fmt.Sprintf("%s NOT CONTAINS %v", e.field, e.value)
}

// compareValues 比较两个值
func compareValues(a, b interface{}) int {
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case float64:
		if vb, ok := b.(float64); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case string:
		if vb, ok := b.(string); ok {
			return strings.Compare(va, vb)
		}
	}
	return 0
}
