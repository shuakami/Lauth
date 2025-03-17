package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

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

	// 获取用户输入的密码
	var password string
	fmt.Println("请输入要测试的密码:")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		password = strings.TrimSpace(scanner.Text())
	}

	if password == "" {
		// 使用默认密码进行测试
		password = "Admin@123"
		fmt.Printf("使用默认密码 '%s' 进行测试\n", password)
	}

	// 显示存储的密码哈希
	fmt.Printf("存储的密码哈希: %s\n", adminUser.Password)

	// 使用bcrypt比较密码
	err = bcrypt.CompareHashAndPassword([]byte(adminUser.Password), []byte(password))
	if err != nil {
		fmt.Printf("密码验证失败: %v\n", err)
	} else {
		fmt.Println("密码验证成功!")
	}
}
