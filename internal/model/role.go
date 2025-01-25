package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role 角色实体
type Role struct {
	ID          string    `gorm:"type:uuid;primary_key" json:"id"`
	AppID       string    `gorm:"type:uuid;not null" json:"app_id"`                                                // 关联的应用ID
	Name        string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_app_role_name,priority:2" json:"name"` // 角色名称，在应用内唯一
	Description string    `gorm:"type:text" json:"description"`                                                    // 角色描述
	IsSystem    bool      `gorm:"type:boolean;default:false" json:"is_system"`                                     // 是否为系统角色
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

// TableName 指定表名
func (Role) TableName() string {
	return "roles"
}
