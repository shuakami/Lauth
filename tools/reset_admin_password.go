package main

import (
	"fmt"
	"log"

	"lauth/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 连接数据库
	dsn := "host=localhost user=postgres password=123456 dbname=lauth port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}

	// 查询admin用户
	var adminUser model.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		log.Fatalf("未找到admin用户: %v", err)
	}

	fmt.Printf("找到用户: ID=%s, 用户名=%s\n", adminUser.ID, adminUser.Username)

	// 生成新密码的哈希
	newPassword := "Admin@123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("生成密码哈希失败: %v", err)
	}

	// 更新密码
	result := db.Model(&adminUser).Update("password", string(hashedPassword))
	if result.Error != nil {
		log.Fatalf("更新密码失败: %v", result.Error)
	}

	fmt.Printf("成功重置admin用户密码为: %s\n", newPassword)
	fmt.Printf("新的密码哈希: %s\n", string(hashedPassword))

	// 验证新密码
	var updatedUser model.User
	if err := db.Where("username = ?", "admin").First(&updatedUser).Error; err != nil {
		log.Fatalf("获取更新后的用户失败: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(updatedUser.Password), []byte(newPassword))
	if err != nil {
		fmt.Printf("验证失败: %v\n", err)
	} else {
		fmt.Println("新密码验证成功!")
	}
}
