package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
)

// LoadRSAKeys 从文件加载RSA密钥对
func LoadRSAKeys(privateKeyPath, publicKeyPath string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// 读取私钥
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		log.Printf("读取私钥文件失败: %v", err)
		return nil, nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// 解析私钥
	privateKeyBlock, _ := pem.Decode(privateKeyBytes)
	if privateKeyBlock == nil {
		log.Printf("解析私钥PEM失败")
		return nil, nil, fmt.Errorf("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		log.Printf("解析私钥失败: %v", err)
		return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// 读取公钥
	publicKeyBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		log.Printf("读取公钥文件失败: %v", err)
		return nil, nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	// 解析公钥
	publicKeyBlock, _ := pem.Decode(publicKeyBytes)
	if publicKeyBlock == nil {
		log.Printf("解析公钥PEM失败")
		return nil, nil, fmt.Errorf("failed to decode public key PEM")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	if err != nil {
		log.Printf("解析公钥失败: %v", err)
		return nil, nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		log.Printf("公钥类型不是RSA公钥")
		return nil, nil, fmt.Errorf("not an RSA public key")
	}

	return privateKey, publicKey, nil
}

// GenerateRSAKeyPair 生成新的RSA密钥对并保存到文件
func GenerateRSAKeyPair(privateKeyPath, publicKeyPath string) error {
	// 生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// 将私钥编码为PEM格式
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	// 保存私钥到文件
	privateKeyFile, err := os.Create(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateKeyFile.Close()

	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to write private key file: %w", err)
	}

	// 将公钥编码为PEM格式
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	// 保存公钥到文件
	publicKeyFile, err := os.Create(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicKeyFile.Close()

	if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
		return fmt.Errorf("failed to write public key file: %w", err)
	}

	return nil
}
