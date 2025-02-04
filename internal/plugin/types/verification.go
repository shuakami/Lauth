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

// LinkSender 验证链接发送接口
type LinkSender interface {
	Send(to string, link string, expireMinutes int) error
}

// VerificationLink 验证链接信息
type VerificationLink struct {
	Token     string    // 验证token
	Link      string    // 完整的验证链接
	ExpiredAt time.Time // 过期时间
}

// VerificationLinkManager 验证链接管理器接口
type VerificationLinkManager interface {
	// Generate 生成验证链接
	Generate(to string, sessionID string) (*VerificationLink, error)

	// Send 发送验证链接
	Send(to string, sessionID string) error

	// Verify 验证token
	Verify(to string, token string) (string, error)

	// IsValid 检查token是否有效
	IsValid(to string) bool
}

// LinkConfig 链接验证配置
type LinkConfig struct {
	BaseURL     string        `json:"base_url"`     // 验证链接的基础URL
	TokenLength int           `json:"token_length"` // 验证token长度
	ExpireTime  time.Duration `json:"expire_time"`  // 链接过期时间
}
