package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RuleType 规则类型
type RuleType string

const (
	RuleTypeStatic  RuleType = "static"  // 静态规则
	RuleTypeDynamic RuleType = "dynamic" // 动态规则
)

// Operator 操作符类型
type Operator string

const (
	OperatorEqual              Operator = "eq"           // 等于
	OperatorNotEqual           Operator = "ne"           // 不等于
	OperatorGreaterThan        Operator = "gt"           // 大于
	OperatorGreaterThanOrEqual Operator = "gte"          // 大于等于
	OperatorLessThan           Operator = "lt"           // 小于
	OperatorLessThanOrEqual    Operator = "lte"          // 小于等于
	OperatorIn                 Operator = "in"           // 在列表中
	OperatorNotIn              Operator = "not_in"       // 不在列表中
	OperatorContains           Operator = "contains"     // 包含
	OperatorNotContains        Operator = "not_contains" // 不包含
)

// Rule 规则实体
type Rule struct {
	ID          string          `gorm:"type:uuid;primary_key" json:"id"`
	AppID       string          `gorm:"type:uuid;not null" json:"app_id"`                                                // 关联的应用ID
	Name        string          `gorm:"type:varchar(100);not null;uniqueIndex:idx_app_rule_name,priority:2" json:"name"` // 规则名称，在应用内唯一
	Description string          `gorm:"type:varchar(500)" json:"description"`                                            // 规则描述
	Type        RuleType        `gorm:"type:varchar(20);not null" json:"type"`                                           // 规则类型
	Priority    int             `gorm:"type:int;default:0" json:"priority"`                                              // 优先级，数字越大优先级越高
	IsEnabled   bool            `gorm:"type:boolean;default:true" json:"is_enabled"`                                     // 是否启用
	App         *App            `gorm:"foreignKey:AppID" json:"-"`                                                       // 关联的应用
	Conditions  []RuleCondition `gorm:"foreignKey:RuleID" json:"conditions"`                                             // 规则条件
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// RuleCondition 规则条件实体
type RuleCondition struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	RuleID    string    `gorm:"type:uuid;not null" json:"rule_id"`         // 关联的规则ID
	Field     string    `gorm:"type:varchar(100);not null" json:"field"`   // 字段名
	Operator  Operator  `gorm:"type:varchar(20);not null" json:"operator"` // 操作符
	Value     string    `gorm:"type:text;not null" json:"value"`           // 值
	Rule      *Rule     `gorm:"foreignKey:RuleID" json:"-"`                // 关联的规则
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID
func (r *Rule) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID
func (c *RuleCondition) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// TableName 指定表名
func (Rule) TableName() string {
	return "rules"
}

// TableName 指定表名
func (RuleCondition) TableName() string {
	return "rule_conditions"
}
