package main

import (
	"fmt"
	"lauth/internal/model"
	"log"

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

	// 查询所有用户
	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		log.Fatalf("查询用户失败: %v", err)
	}

	// 打印用户信息
	fmt.Println("全部用户:")
	for _, user := range users {
		fmt.Printf("ID: %s, 用户名: %s, 状态: %d, 哈希密码长度: %d\n",
			user.ID, user.Username, user.Status, len(user.Password))
	}

	// 查询admin用户
	var adminUser model.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		log.Printf("未找到admin用户: %v", err)
	} else {
		fmt.Printf("\nadmin用户详情:\n")
		fmt.Printf("ID: %s\n", adminUser.ID)
		fmt.Printf("用户名: %s\n", adminUser.Username)
		fmt.Printf("密码哈希: %s\n", adminUser.Password)
		fmt.Printf("状态: %d\n", adminUser.Status)
		fmt.Printf("是否首次登录: %t\n", adminUser.IsFirstLogin)
		fmt.Printf("创建时间: %s\n", adminUser.CreatedAt)
		fmt.Printf("更新时间: %s\n", adminUser.UpdatedAt)
	}
}
