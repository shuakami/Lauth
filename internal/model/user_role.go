package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRole 用户-角色关联实体
type UserRole struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	UserID    string    `gorm:"type:uuid;not null" json:"user_id"` // 用户ID
	RoleID    string    `gorm:"type:uuid;not null" json:"role_id"` // 角色ID
	AppID     string    `gorm:"type:uuid;not null" json:"app_id"`  // 应用ID
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"-"`
	Role *Role `gorm:"foreignKey:RoleID" json:"-"`
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID
func (ur *UserRole) BeforeCreate(tx *gorm.DB) error {
	if ur.ID == "" {
		ur.ID = uuid.New().String()
	}
	return nil
}

// TableName 指定表名
func (UserRole) TableName() string {
	return "user_roles"
}

// 更新User模型，添加角色关联
func init() {
	// 在User模型中添加Roles字段
	type User struct {
		Roles []Role `gorm:"many2many:user_roles;" json:"roles,omitempty"`
	}
}
