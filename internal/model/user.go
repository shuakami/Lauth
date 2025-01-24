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

// User 用户实体
type User struct {
	ID        string     `gorm:"type:uuid;primary_key" json:"id"`
	AppID     string     `gorm:"type:uuid;not null" json:"app_id"`                                                   // 关联的应用ID
	Username  string     `gorm:"type:varchar(100);not null;uniqueIndex:idx_app_username,priority:2" json:"username"` // 用户名，在应用内唯一
	Password  string     `gorm:"type:varchar(100);not null" json:"-"`                                                // 密码，json中隐藏
	Nickname  string     `gorm:"type:varchar(100)" json:"nickname"`                                                  // 昵称
	Email     string     `gorm:"type:varchar(100)" json:"email"`                                                     // 邮箱
	Phone     string     `gorm:"type:varchar(20)" json:"phone"`                                                      // 手机号
	Status    UserStatus `gorm:"type:int;default:1" json:"status"`                                                   // 状态
	App       *App       `gorm:"foreignKey:AppID" json:"-"`                                                          // 关联的应用
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
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
	log.Printf("验证密码: 存储的哈希=%s, 输入的密码长度=%d", u.Password, len(password))
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		log.Printf("密码验证失败: %v", err)
		return false
	}
	log.Printf("密码验证成功")
	return true
}
