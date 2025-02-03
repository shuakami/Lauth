package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoginLocation 登录位置记录
type LoginLocation struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	AppID     string    `gorm:"type:uuid;not null;index" json:"app_id"`
	UserID    string    `gorm:"type:uuid;not null;index:idx_user_login_time,priority:1" json:"user_id"`
	IP        string    `gorm:"type:varchar(50);not null" json:"ip"`
	Country   string    `gorm:"type:varchar(100)" json:"country"`
	Province  string    `gorm:"type:varchar(100)" json:"province"`
	City      string    `gorm:"type:varchar(100)" json:"city"`
	ISP       string    `gorm:"type:varchar(100)" json:"isp"`
	LoginTime time.Time `gorm:"not null;index:idx_user_login_time,priority:2" json:"login_time"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID
func (l *LoginLocation) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}

// TableName 返回表名
func (LoginLocation) TableName() string {
	return "login_locations"
}

// LoginLocationResponse 登录位置响应
type LoginLocationResponse struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	Country   string    `json:"country"`
	Province  string    `json:"province"`
	City      string    `json:"city"`
	ISP       string    `json:"isp"`
	LoginTime time.Time `json:"login_time"`
}
