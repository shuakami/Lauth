package model

import "time"

// TokenType 令牌类型
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// TokenClaims JWT令牌的声明
type TokenClaims struct {
	UserID    string    `json:"user_id"`
	AppID     string    `json:"app_id"`
	Username  string    `json:"username"`
	Type      TokenType `json:"type"`
	ExpiresAt time.Time `json:"expires_at"`
	Scope     string    `json:"scope,omitempty"`
}

// GetExpiresAt 获取过期时间
func (tc *TokenClaims) GetExpiresAt() time.Time {
	return tc.ExpiresAt
}

// TokenPair 令牌对
type TokenPair struct {
	AccessToken          string        `json:"access_token"`
	RefreshToken         string        `json:"refresh_token"`
	AccessTokenExpireIn  time.Duration `json:"access_token_expire_in"`
	RefreshTokenExpireIn time.Duration `json:"refresh_token_expire_in"`
}

// TokenUserInfo Token中包含的用户信息（快速接口使用）
type TokenUserInfo struct {
	UserID   string `json:"user_id"`
	AppID    string `json:"app_id"`
	Username string `json:"username"`
}
