package types

import "time"

// CodeSender 验证码发送接口
type CodeSender interface {
	Send(to string, code string, expireMinutes int) error
}

// VerificationCode 验证码信息
type VerificationCode struct {
	Code      string    // 验证码
	ExpiredAt time.Time // 过期时间
}

// VerificationCodeManager 验证码管理器接口
type VerificationCodeManager interface {
	// Generate 生成验证码
	Generate() string

	// Send 发送验证码
	Send(to string) error

	// Verify 验证验证码
	Verify(to string, code string) error

	// IsValid 检查验证码是否有效
	IsValid(to string) bool
}

// VerificationConfig 验证码配置
type VerificationConfig struct {
	CodeLength int           // 验证码长度
	ExpireTime time.Duration // 验证码过期时间
}
