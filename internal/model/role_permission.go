package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RolePermission 角色-权限关联实体
type RolePermission struct {
	ID           string    `gorm:"type:uuid;primary_key" json:"id"`
	RoleID       string    `gorm:"type:uuid;not null" json:"role_id"`       // 角色ID
	PermissionID string    `gorm:"type:uuid;not null" json:"permission_id"` // 权限ID
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// 关联
	Role       *Role       `gorm:"foreignKey:RoleID" json:"-"`
	Permission *Permission `gorm:"foreignKey:PermissionID" json:"-"`
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID
func (rp *RolePermission) BeforeCreate(tx *gorm.DB) error {
	if rp.ID == "" {
		rp.ID = uuid.New().String()
	}
	return nil
}

// TableName 指定表名
func (RolePermission) TableName() string {
	return "role_permissions"
}
