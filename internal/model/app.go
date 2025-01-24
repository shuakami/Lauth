package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AppStatus 应用状态
type AppStatus int

const (
	AppStatusDisabled AppStatus = iota // 禁用
	AppStatusEnabled                   // 启用
)

// App 应用实体
type App struct {
	ID          string    `gorm:"type:uuid;primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null;unique" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	AppKey      string    `gorm:"type:varchar(64);not null;unique" json:"app_key"`
	AppSecret   string    `gorm:"type:varchar(64);not null" json:"app_secret"`
	Status      AppStatus `gorm:"type:int;default:1" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID和密钥
func (a *App) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.AppKey == "" {
		a.AppKey = uuid.New().String()
	}
	if a.AppSecret == "" {
		a.AppSecret = uuid.NewString()
	}
	return nil
}
