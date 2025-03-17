package model

import (
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserStatus 用户状态
type UserStatus int

const (
	UserStatusDisabled UserStatus = iota // 禁用
	UserStatusEnabled                    // 启用
)

// User 用户模型
type User struct {
	ID        string     `json:"id" gorm:"primaryKey;type:uuid"`
	AppID     string     `json:"app_id" gorm:"index;type:uuid"`
	Username  string     `json:"username" gorm:"type:varchar(100);uniqueIndex:idx_app_username,priority:2"`
	Password  string     `json:"-" gorm:"type:varchar(100)"`
	Status    UserStatus `json:"status" gorm:"type:int;default:1"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// 首次登录和密码安全
	IsFirstLogin      bool       `json:"is_first_login" gorm:"default:true"`      // 是否首次登录
	PasswordExpiresAt *time.Time `json:"password_expires_at" gorm:"default:null"` // 密码过期时间
	LastLoginAt       *time.Time `json:"last_login_at" gorm:"default:null"`       // 最后登录时间
	IsSuperAdmin      bool       `json:"-" gorm:"-"`                              // 是否是超级管理员（非数据库字段）

	// OIDC相关字段
	Name          string `json:"name" gorm:"type:varchar(100)"`
	Nickname      string `json:"nickname" gorm:"type:varchar(100)"`
	Email         string `json:"email" gorm:"type:varchar(100)"`
	EmailVerified bool   `json:"email_verified" gorm:"default:false"`
	Phone         string `json:"phone" gorm:"type:varchar(20)"`
	PhoneVerified bool   `json:"phone_verified" gorm:"default:false"`
	Picture       string `json:"picture" gorm:"type:varchar(500)"`
	Locale        string `json:"locale" gorm:"type:varchar(10)"`
	Birthdate     string `json:"birthdate" gorm:"type:varchar(10)"`
	Gender        string `json:"gender" gorm:"type:varchar(10)"`
	Website       string `json:"website" gorm:"type:varchar(200)"`
	Zoneinfo      string `json:"zoneinfo" gorm:"type:varchar(50)"`

	// 角色关联
	Roles []Role `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

// BeforeCreate GORM的钩子，在创建记录前自动生成UUID和加密密码
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	if u.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return nil
}

// BeforeUpdate GORM的钩子，在更新记录前加密密码
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	if u.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashedPassword)
	}
	return nil
}

// ValidatePassword 验证密码
func (u *User) ValidatePassword(password string) bool {
	// 比较哈希和密码
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		log.Printf("[DEBUG] [User %s] 密码验证失败: %v", u.ID, err)
		return false
	}
	log.Printf("[DEBUG] [User %s] 密码验证成功", u.ID)
	return true
}
