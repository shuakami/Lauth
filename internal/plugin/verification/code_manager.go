package verification

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"lauth/internal/plugin/types"
)

// DefaultCodeManager 默认的验证码管理器实现
type DefaultCodeManager struct {
	mu     sync.RWMutex
	store  map[string]types.VerificationCode
	config *types.VerificationConfig
	sender types.CodeSender
}

// NewDefaultCodeManager 创建默认验证码管理器
func NewDefaultCodeManager(config *types.VerificationConfig, sender types.CodeSender) types.VerificationCodeManager {
	return &DefaultCodeManager{
		store:  make(map[string]types.VerificationCode),
		config: config,
		sender: sender,
	}
}

// Generate 生成验证码
func (m *DefaultCodeManager) Generate() string {
	const charset = "0123456789"
	code := make([]byte, m.config.CodeLength)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

// Send 发送验证码
func (m *DefaultCodeManager) Send(to string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成验证码
	code := m.Generate()

	// 发送验证码
	if err := m.sender.Send(
		to,
		code,
		int(m.config.ExpireTime.Minutes()),
	); err != nil {
		return fmt.Errorf("failed to send code: %v", err)
	}

	// 保存验证码
	m.store[to] = types.VerificationCode{
		Code:      code,
		ExpiredAt: time.Now().Add(m.config.ExpireTime),
	}

	return nil
}

// Verify 验证验证码
func (m *DefaultCodeManager) Verify(to string, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stored, exists := m.store[to]
	if !exists {
		return fmt.Errorf("no verification code found for: %s", to)
	}

	if time.Now().After(stored.ExpiredAt) {
		delete(m.store, to)
		return fmt.Errorf("verification code expired")
	}

	if stored.Code != code {
		return fmt.Errorf("invalid verification code")
	}

	delete(m.store, to)
	return nil
}

// IsValid 检查验证码是否有效
func (m *DefaultCodeManager) IsValid(to string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if stored, exists := m.store[to]; exists {
		return !time.Now().After(stored.ExpiredAt)
	}
	return false
}
