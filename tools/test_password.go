package main

import (
	"fmt"
	"lauth/internal/model"
	"log"

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

	// 测试默认密码
	testPassword := "Admin@123"
	fmt.Printf("测试密码 '%s' 与数据库中存储的哈希是否匹配\n", testPassword)
	fmt.Printf("存储的密码哈希: %s\n", adminUser.Password)

	// 使用bcrypt比较
	err = bcrypt.CompareHashAndPassword([]byte(adminUser.Password), []byte(testPassword))
	if err != nil {
		fmt.Printf("密码验证失败: %v\n", err)
	} else {
		fmt.Println("密码验证成功!")
	}

	// 测试其他可能的密码
	otherPasswords := []string{"admin", "password", "123456", "Admin123", "Admin123!"}
	fmt.Println("\n测试其他可能的密码:")
	for _, pwd := range otherPasswords {
		err = bcrypt.CompareHashAndPassword([]byte(adminUser.Password), []byte(pwd))
		if err != nil {
			fmt.Printf("密码 '%s' 验证失败\n", pwd)
		} else {
			fmt.Printf("密码 '%s' 验证成功!\n", pwd)
		}
	}

	// 允许用户输入密码进行测试
	fmt.Println("\n如果您知道正确的密码，请运行以下命令测试:")
	fmt.Printf("echo \"你的密码\" | go run tools/check_password.go\n")
}
