package service

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"time"

	"lauth/internal/model"
	"lauth/internal/repository"
	"lauth/pkg/config"

	"github.com/golang-jwt/jwt/v5"
)

// OIDCService OIDC服务接口
type OIDCService interface {
	// GenerateIDToken 生成ID Token
	GenerateIDToken(ctx context.Context, user *model.User, client *model.OAuthClient, nonce string) (string, error)

	// GetUserInfo 获取用户信息
	GetUserInfo(ctx context.Context, userID string, scope string) (*model.OIDCClaims, error)

	// GetConfiguration 获取OIDC配置
	GetConfiguration(ctx context.Context) (*model.OIDCConfiguration, error)

	// GetJWKS 获取JSON Web Key Set
	GetJWKS(ctx context.Context) (map[string]interface{}, error)
}

// oidcService OIDC服务实现
type oidcService struct {
	userRepo     repository.UserRepository
	tokenService TokenService
	config       *config.Config
	privateKey   *rsa.PrivateKey
	publicKey    *rsa.PublicKey
}

// NewOIDCService 创建OIDC服务实例
func NewOIDCService(
	userRepo repository.UserRepository,
	tokenService TokenService,
	config *config.Config,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
) OIDCService {
	return &oidcService{
		userRepo:     userRepo,
		tokenService: tokenService,
		config:       config,
		privateKey:   privateKey,
		publicKey:    publicKey,
	}
}

// GenerateIDToken 生成ID Token
func (s *oidcService) GenerateIDToken(ctx context.Context, user *model.User, client *model.OAuthClient, nonce string) (string, error) {
	now := time.Now()

	claims := &model.OIDCClaims{
		Issuer:    s.config.OIDC.Issuer,
		Subject:   user.ID,
		Audience:  client.ClientID,
		ExpiresAt: now.Add(time.Hour).Unix(),
		IssuedAt:  now.Unix(),
		AuthTime:  now.Unix(),
		Nonce:     nonce,

		// 用户信息Claims
		Name:              user.Name,
		PreferredUsername: user.Username,
		Email:             user.Email,
		EmailVerified:     user.EmailVerified,
		PhoneNumber:       user.Phone,
		PhoneVerified:     user.PhoneVerified,
		UpdatedAt:         user.UpdatedAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "default" // 使用默认的密钥ID

	return token.SignedString(s.privateKey)
}

// GetUserInfo 获取用户信息
func (s *oidcService) GetUserInfo(ctx context.Context, userID string, scope string) (*model.OIDCClaims, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	now := time.Now()
	claims := &model.OIDCClaims{
		Issuer:    s.config.OIDC.Issuer,
		Subject:   user.ID,
		Audience:  user.AppID,
		ExpiresAt: now.Add(time.Hour).Unix(),
		IssuedAt:  now.Unix(),
	}

	// 根据scope返回相应的用户信息
	scopes := map[string]bool{}
	for _, s := range splitScope(scope) {
		scopes[s] = true
	}

	if scopes[model.ScopeProfile] {
		// 必需字段
		claims.Name = user.Name
		claims.PreferredUsername = user.Username
		claims.UpdatedAt = user.UpdatedAt.Unix()

		// 可选字段
		if user.Nickname != "" {
			claims.Nickname = user.Nickname
		}
		if user.Picture != "" {
			claims.Picture = user.Picture
		}
		if user.Gender != "" {
			claims.Gender = user.Gender
		}
		if user.Birthdate != "" {
			claims.Birthdate = user.Birthdate
		}
		if user.Zoneinfo != "" {
			claims.Zoneinfo = user.Zoneinfo
		}
		if user.Locale != "" {
			claims.Locale = user.Locale
		}
		if user.Website != "" {
			claims.Website = user.Website
		}
	}

	if scopes[model.ScopeEmail] {
		claims.Email = user.Email
		claims.EmailVerified = user.EmailVerified
	}

	if scopes[model.ScopePhone] {
		claims.PhoneNumber = user.Phone
		claims.PhoneVerified = user.PhoneVerified
	}

	return claims, nil
}

// GetConfiguration 获取OIDC配置
func (s *oidcService) GetConfiguration(ctx context.Context) (*model.OIDCConfiguration, error) {
	return &model.OIDCConfiguration{
		Issuer:                           s.config.OIDC.Issuer,
		AuthorizationEndpoint:            s.config.OIDC.Issuer + "/oauth/authorize",
		TokenEndpoint:                    s.config.OIDC.Issuer + "/oauth/token",
		UserInfoEndpoint:                 s.config.OIDC.Issuer + "/userinfo",
		JWKSUri:                          s.config.OIDC.Issuer + "/.well-known/jwks.json",
		ScopesSupported:                  []string{model.ScopeOpenID, model.ScopeProfile, model.ScopeEmail, model.ScopePhone, model.ScopeAddress},
		ResponseTypesSupported:           []string{"code", "id_token", "code id_token"},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
		ClaimsSupported: []string{
			"iss", "sub", "aud", "exp", "iat", "auth_time",
			"nonce", "name", "preferred_username", "email",
			"email_verified", "phone_number", "phone_verified",
		},
	}, nil
}

// GetJWKS 获取JSON Web Key Set
func (s *oidcService) GetJWKS(ctx context.Context) (map[string]interface{}, error) {
	// 将公钥转换为JWKS格式
	n := base64.RawURLEncoding.EncodeToString(s.publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(s.publicKey.E)).Bytes())

	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kid": "default",
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"n":   n,
				"e":   e,
			},
		},
	}

	return jwks, nil
}

// splitScope 分割scope字符串
func splitScope(scope string) []string {
	if scope == "" {
		return nil
	}
	return strings.Split(scope, " ")
}
