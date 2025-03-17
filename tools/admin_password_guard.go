package main

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"lauth/internal/model"
)

// 这个脚本可以在系统启动时自动运行，确保admin密码始终如预期
func main() {
	// 连接数据库
	dsn := "host=localhost user=postgres password=123456 dbname=lauth port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}

	// 等待主服务器启动
	time.Sleep(3 * time.Second)

	// 查找所有admin用户
	var adminUsers []model.User
	if err := db.Where("username = ?", "admin").Find(&adminUsers).Error; err != nil {
		log.Fatalf("查询admin用户失败: %v", err)
	}

	if len(adminUsers) == 0 {
		log.Println("没有找到admin用户，无需修复")
		return
	}

	// 设置预期的密码
	expectedPassword := "Admin@123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(expectedPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("生成密码哈希失败: %v", err)
	}

	// 更新所有admin用户的密码
	for _, user := range adminUsers {
		log.Printf("更新用户 %s (%s) 的密码...", user.Username, user.ID)
		if err := db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
			log.Printf("  更新失败: %v", err)
		} else {
			log.Printf("  更新成功!")
		}
	}

	log.Println("守护脚本执行完成")
}
