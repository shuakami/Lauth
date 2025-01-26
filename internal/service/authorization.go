package service

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"lauth/internal/model"
	"lauth/internal/repository"
)

var (
	// ErrInvalidClient 无效的客户端
	ErrInvalidClient = errors.New("invalid client")
	// ErrInvalidRedirectURI 无效的重定向URI
	ErrInvalidRedirectURI = errors.New("invalid redirect uri")
	// ErrInvalidScope 无效的权限范围
	ErrInvalidScope = errors.New("invalid scope")
	// ErrUnsupportedGrantType 不支持的授权类型
	ErrUnsupportedGrantType = errors.New("unsupported grant type")
)

// AuthorizationService 授权服务接口
type AuthorizationService interface {
	// Authorize 处理授权请求
	Authorize(ctx context.Context, userID string, req *model.AuthorizationRequest) (string, error)
}

// authorizationService 授权服务实现
type authorizationService struct {
	clientRepo repository.OAuthClientRepository
	codeRepo   repository.AuthorizationCodeRepository
}

// NewAuthorizationService 创建授权服务实例
func NewAuthorizationService(
	clientRepo repository.OAuthClientRepository,
	codeRepo repository.AuthorizationCodeRepository,
) AuthorizationService {
	return &authorizationService{
		clientRepo: clientRepo,
		codeRepo:   codeRepo,
	}
}

// Authorize 处理授权请求
func (s *authorizationService) Authorize(ctx context.Context, userID string, req *model.AuthorizationRequest) (string, error) {
	// 1. 获取并验证客户端
	client, err := s.clientRepo.GetByClientID(ctx, req.ClientID)
	if err != nil {
		return "", err
	}
	if client == nil || !client.Status {
		return "", ErrInvalidClient
	}

	// 2. 验证授权类型
	if !s.containsGrantType(client.GrantTypes, string(model.AuthorizationCodeGrant)) {
		return "", ErrUnsupportedGrantType
	}

	// 3. 验证重定向URI
	if !s.validateRedirectURI(client.RedirectURIs, req.RedirectURI) {
		return "", ErrInvalidRedirectURI
	}

	// 4. 验证权限范围
	if !s.validateScope(client.Scopes, req.Scope) {
		return "", ErrInvalidScope
	}

	// 5. 生成授权码
	authCode := &model.AuthorizationCode{
		ClientID:    req.ClientID,
		UserID:      userID,
		RedirectURI: req.RedirectURI,
		Scope:       req.Scope,
		ExpiresAt:   time.Now().Add(10 * time.Minute), // 授权码10分钟有效
		CreatedAt:   time.Now(),
	}

	if err := s.codeRepo.Create(ctx, authCode); err != nil {
		return "", err
	}

	// 6. 构建重定向URL
	redirectURL, err := url.Parse(req.RedirectURI)
	if err != nil {
		return "", err
	}

	query := redirectURL.Query()
	query.Set("code", authCode.Code)
	if req.State != "" {
		query.Set("state", req.State)
	}
	redirectURL.RawQuery = query.Encode()

	return redirectURL.String(), nil
}

// validateRedirectURI 验证重定向URI
func (s *authorizationService) validateRedirectURI(allowedURIs []string, redirectURI string) bool {
	for _, uri := range allowedURIs {
		if uri == redirectURI {
			return true
		}
	}
	return false
}

// validateScope 验证权限范围
func (s *authorizationService) validateScope(allowedScopes []string, scope string) bool {
	requestedScopes := strings.Split(scope, " ")
	for _, requestedScope := range requestedScopes {
		found := false
		for _, allowedScope := range allowedScopes {
			if requestedScope == allowedScope {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// containsGrantType 检查是否包含指定的授权类型
func (s *authorizationService) containsGrantType(grantTypes []string, targetType string) bool {
	for _, gt := range grantTypes {
		if gt == targetType {
			return true
		}
	}
	return false
}
