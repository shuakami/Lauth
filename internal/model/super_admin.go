package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SuperAdmin 超级管理员模型
type SuperAdmin struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid"`
	UserID    string    `json:"user_id" gorm:"uniqueIndex;type:uuid"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联User
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 设置表名
func (SuperAdmin) TableName() string {
	return "super_admins"
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID
func (sa *SuperAdmin) BeforeCreate(tx *gorm.DB) error {
	if sa.ID == "" {
		sa.ID = uuid.New().String()
	}
	return nil
}
