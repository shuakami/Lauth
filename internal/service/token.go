package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"lauth/internal/model"
	"lauth/pkg/redis"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
	ErrTokenRevoked = errors.New("token revoked")
)

// TokenService Token服务接口
type TokenService interface {
	// GenerateTokenPair 生成访问令牌和刷新令牌对
	GenerateTokenPair(ctx context.Context, user *model.User, scope string) (*model.TokenPair, error)

	// ValidateToken 验证令牌
	ValidateToken(ctx context.Context, tokenString string, tokenType model.TokenType) (*model.TokenClaims, error)

	// RefreshToken 刷新访问令牌
	RefreshToken(ctx context.Context, refreshToken string) (*model.TokenPair, error)

	// RevokeToken 吊销令牌
	RevokeToken(ctx context.Context, tokenString string, tokenType model.TokenType) error
}

// tokenService Token服务实现
type tokenService struct {
	redis         *redis.Client
	jwtSecret     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewTokenService 创建Token服务实例
func NewTokenService(redisClient *redis.Client, jwtSecret string, accessExpiry, refreshExpiry time.Duration) TokenService {
	return &tokenService{
		redis:         redisClient,
		jwtSecret:     []byte(jwtSecret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// generateToken 生成JWT令牌
func (s *tokenService) generateToken(claims *model.TokenClaims, expiry time.Duration) (string, error) {
	expiresAt := time.Now().Add(expiry)
	claims.ExpiresAt = expiresAt

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":    claims.UserID,
		"app_id":     claims.AppID,
		"username":   claims.Username,
		"type":       claims.Type,
		"exp":        expiresAt.Unix(),
		"expires_at": expiresAt,
		"scope":      claims.Scope,
	})

	return token.SignedString(s.jwtSecret)
}

// GenerateTokenPair 生成访问令牌和刷新令牌对
func (s *tokenService) GenerateTokenPair(ctx context.Context, user *model.User, scope string) (*model.TokenPair, error) {
	// 生成访问令牌
	accessClaims := &model.TokenClaims{
		UserID:   user.ID,
		AppID:    user.AppID,
		Username: user.Username,
		Type:     model.AccessToken,
		Scope:    scope,
	}
	accessToken, err := s.generateToken(accessClaims, s.accessExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// 生成刷新令牌
	refreshClaims := &model.TokenClaims{
		UserID:   user.ID,
		AppID:    user.AppID,
		Username: user.Username,
		Type:     model.RefreshToken,
		Scope:    scope,
	}
	refreshToken, err := s.generateToken(refreshClaims, s.refreshExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// 将刷新令牌存储到Redis
	refreshKey := fmt.Sprintf("refresh_token:%s", user.ID)
	if err := s.redis.Set(ctx, refreshKey, refreshToken, s.refreshExpiry); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:          accessToken,
		RefreshToken:         refreshToken,
		AccessTokenExpireIn:  s.accessExpiry,
		RefreshTokenExpireIn: s.refreshExpiry,
	}, nil
}

// ValidateToken 验证令牌
func (s *tokenService) ValidateToken(ctx context.Context, tokenString string, tokenType model.TokenType) (*model.TokenClaims, error) {
	// 解析JWT令牌
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// 验证令牌声明
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// 检查令牌类型
	if claims["type"].(string) != string(tokenType) {
		return nil, ErrInvalidToken
	}

	// 检查是否已被吊销
	revokedKey := fmt.Sprintf("revoked_token:%s", tokenString)
	revoked, err := s.redis.Exists(ctx, revokedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check token revocation: %w", err)
	}
	if revoked {
		return nil, ErrTokenRevoked
	}

	// 获取过期时间
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, ErrInvalidToken
	}
	expiresAt := time.Unix(int64(exp), 0)

	// 检查是否已过期
	if time.Now().After(expiresAt) {
		return nil, ErrTokenExpired
	}

	// 获取 scope 字段
	scope, _ := claims["scope"].(string)

	return &model.TokenClaims{
		UserID:    claims["user_id"].(string),
		AppID:     claims["app_id"].(string),
		Username:  claims["username"].(string),
		Type:      model.TokenType(claims["type"].(string)),
		ExpiresAt: expiresAt,
		Scope:     scope,
	}, nil
}

// RefreshToken 刷新访问令牌
func (s *tokenService) RefreshToken(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	// 验证刷新令牌
	claims, err := s.ValidateToken(ctx, refreshToken, model.RefreshToken)
	if err != nil {
		return nil, err
	}

	// 检查Redis中存储的刷新令牌是否匹配
	refreshKey := fmt.Sprintf("refresh_token:%s", claims.UserID)
	storedToken, err := s.redis.Get(ctx, refreshKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get stored refresh token: %w", err)
	}
	if storedToken != refreshToken {
		return nil, ErrInvalidToken
	}

	// 生成新的令牌对
	user := &model.User{
		ID:       claims.UserID,
		AppID:    claims.AppID,
		Username: claims.Username,
	}
	return s.GenerateTokenPair(ctx, user, claims.Scope)
}

// RevokeToken 吊销令牌
func (s *tokenService) RevokeToken(ctx context.Context, tokenString string, tokenType model.TokenType) error {
	// 验证令牌
	claims, err := s.ValidateToken(ctx, tokenString, tokenType)
	if err != nil {
		return err
	}

	// 将令牌加入吊销列表
	revokedKey := fmt.Sprintf("revoked_token:%s", tokenString)
	var expiry time.Duration
	if tokenType == model.AccessToken {
		expiry = s.accessExpiry
	} else {
		expiry = s.refreshExpiry
	}

	if err := s.redis.Set(ctx, revokedKey, "revoked", expiry); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// 如果是刷新令牌，同时删除存储的刷新令牌
	if tokenType == model.RefreshToken {
		refreshKey := fmt.Sprintf("refresh_token:%s", claims.UserID)
		if err := s.redis.Del(ctx, refreshKey); err != nil {
			return fmt.Errorf("failed to delete refresh token: %w", err)
		}
	}

	return nil
}
