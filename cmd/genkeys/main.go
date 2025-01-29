package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"lauth/pkg/crypto"
)

func main() {
	// 解析命令行参数
	var (
		keyDir = flag.String("dir", "config/keys", "密钥存储目录")
	)
	flag.Parse()

	// 创建密钥目录
	if err := os.MkdirAll(*keyDir, 0700); err != nil {
		log.Fatalf("Failed to create key directory: %v", err)
	}

	// 设置密钥文件路径
	privateKeyPath := filepath.Join(*keyDir, "oidc.key")
	publicKeyPath := filepath.Join(*keyDir, "oidc.pub")

	// 生成RSA密钥对
	if err := crypto.GenerateRSAKeyPair(privateKeyPath, publicKeyPath); err != nil {
		log.Fatalf("Failed to generate RSA key pair: %v", err)
	}

	log.Printf("Successfully generated RSA key pair:")
	log.Printf("Private key: %s", privateKeyPath)
	log.Printf("Public key: %s", publicKeyPath)
}
