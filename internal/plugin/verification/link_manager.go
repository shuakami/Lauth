package verification

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"lauth/internal/plugin/types"
)

// DefaultLinkManager 默认的验证链接管理器实现
type DefaultLinkManager struct {
	mu         sync.RWMutex
	store      map[string]types.VerificationLink
	config     *types.VerificationConfig
	linkConfig *types.LinkConfig
	sender     types.LinkSender
}

// NewDefaultLinkManager 创建默认验证链接管理器
func NewDefaultLinkManager(config *types.VerificationConfig, linkConfig *types.LinkConfig, sender types.LinkSender) types.VerificationLinkManager {
	return &DefaultLinkManager{
		store:      make(map[string]types.VerificationLink),
		config:     config,
		linkConfig: linkConfig,
		sender:     sender,
	}
}

// Generate 生成验证链接
func (m *DefaultLinkManager) Generate(to string, sessionID string) (*types.VerificationLink, error) {
	// 生成随机token
	token, err := m.generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %v", err)
	}

	// 将session_id编码到token中
	combinedToken := fmt.Sprintf("%s.%s", token, sessionID)

	// 构建完整的验证链接
	link := fmt.Sprintf("%s?token=%s&email=%s", m.linkConfig.BaseURL, combinedToken, to)

	// 创建验证链接信息
	verificationLink := &types.VerificationLink{
		Token:     combinedToken,
		Link:      link,
		ExpiredAt: time.Now().Add(m.linkConfig.ExpireTime),
	}

	// 保存到store
	m.mu.Lock()
	m.store[to] = *verificationLink
	m.mu.Unlock()

	return verificationLink, nil
}

// Send 发送验证链接
func (m *DefaultLinkManager) Send(to string, sessionID string) error {
	// 生成验证链接
	link, err := m.Generate(to, sessionID)
	if err != nil {
		return err
	}

	// 发送验证链接
	return m.sender.Send(
		to,
		link.Link,
		int(m.linkConfig.ExpireTime.Minutes()),
	)
}

// Verify 验证token
func (m *DefaultLinkManager) Verify(to string, combinedToken string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stored, exists := m.store[to]
	if !exists {
		return "", fmt.Errorf("no verification link found for: %s", to)
	}

	if time.Now().After(stored.ExpiredAt) {
		delete(m.store, to)
		return "", fmt.Errorf("verification link expired")
	}

	if stored.Token != combinedToken {
		return "", fmt.Errorf("invalid verification token")
	}

	// 从token中提取session_id
	parts := strings.Split(combinedToken, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token format")
	}
	sessionID := parts[1]

	delete(m.store, to)
	return sessionID, nil
}

// IsValid 检查token是否有效
func (m *DefaultLinkManager) IsValid(to string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if stored, exists := m.store[to]; exists {
		return !time.Now().After(stored.ExpiredAt)
	}
	return false
}

// generateToken 生成随机token
func (m *DefaultLinkManager) generateToken() (string, error) {
	// 生成随机字节
	bytes := make([]byte, m.linkConfig.TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// 使用base64 URL安全编码
	token := base64.URLEncoding.EncodeToString(bytes)
	return token[:m.linkConfig.TokenLength], nil
}
