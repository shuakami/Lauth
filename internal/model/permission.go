package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ActionType 操作类型
type ActionType string

const (
	ActionCreate ActionType = "create"
	ActionRead   ActionType = "read"
	ActionUpdate ActionType = "update"
	ActionDelete ActionType = "delete"
	ActionList   ActionType = "list"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceApp  ResourceType = "app"
	ResourceUser ResourceType = "user"
	ResourceRole ResourceType = "role"
)

// Permission 权限实体
type Permission struct {
	ID           string       `gorm:"type:uuid;primary_key" json:"id"`
	AppID        string       `gorm:"type:uuid;not null" json:"app_id"`                        // 关联的应用ID
	Name         string       `gorm:"type:varchar(100);not null" json:"name"`                  // 权限名称
	Description  string       `gorm:"type:text" json:"description"`                            // 权限描述
	ResourceType ResourceType `gorm:"type:varchar(50);not null" json:"resource_type"`          // 资源类型
	Action       ActionType   `gorm:"type:varchar(50);not null" json:"action"`                 // 操作类型
	Effect       string       `gorm:"type:varchar(50);not null;default:'allow'" json:"effect"` // 效果：allow/deny
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`

	// 关联
	Roles []Role `gorm:"many2many:role_permissions;" json:"-"`
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID
func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// TableName 指定表名
func (Permission) TableName() string {
	return "permissions"
}
