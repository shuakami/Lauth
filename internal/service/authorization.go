package service

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"lauth/internal/model"
	"lauth/internal/repository"
)

// AuthorizationService 授权服务接口
type AuthorizationService interface {
	// Authorize 处理授权请求
	Authorize(ctx context.Context, userID string, req *model.AuthorizationRequest) (string, error)
	// IssueToken 颁发令牌
	IssueToken(ctx context.Context, req *model.TokenRequest) (*model.TokenResponse, error)
}

// authorizationService 授权服务实现
type authorizationService struct {
	clientRepo   repository.OAuthClientRepository
	secretRepo   repository.OAuthClientSecretRepository
	codeRepo     repository.AuthorizationCodeRepository
	userRepo     repository.UserRepository
	tokenService TokenService
	oidcService  OIDCService
}

// NewAuthorizationService 创建授权服务实例
func NewAuthorizationService(
	clientRepo repository.OAuthClientRepository,
	secretRepo repository.OAuthClientSecretRepository,
	codeRepo repository.AuthorizationCodeRepository,
	userRepo repository.UserRepository,
	tokenService TokenService,
	oidcService OIDCService,
) AuthorizationService {
	return &authorizationService{
		clientRepo:   clientRepo,
		secretRepo:   secretRepo,
		codeRepo:     codeRepo,
		userRepo:     userRepo,
		tokenService: tokenService,
		oidcService:  oidcService,
	}
}

// Authorize 处理授权请求
func (s *authorizationService) Authorize(ctx context.Context, userID string, req *model.AuthorizationRequest) (string, error) {
	log.Printf("Processing authorization request for client_id: %s", req.ClientID)

	// 1. 获取并验证客户端
	client, err := s.clientRepo.GetByClientID(ctx, req.ClientID)
	if err != nil {
		log.Printf("Error getting client: %v", err)
		return "", err
	}
	if client == nil || !client.Status {
		log.Printf("Invalid or inactive client: %s", req.ClientID)
		return "", ErrInvalidClient
	}

	// 2. 验证授权类型
	if !s.containsGrantType(client.GrantTypes, string(model.AuthorizationCodeGrant)) {
		log.Printf("Unsupported grant type for client %s", req.ClientID)
		return "", ErrUnsupportedGrantType
	}

	// 3. 验证重定向URI
	if !s.validateRedirectURI(client.RedirectURIs, req.RedirectURI) {
		log.Printf("Invalid redirect URI: %s", req.RedirectURI)
		return "", ErrInvalidRedirectURI
	}

	// 4. 验证权限范围
	if !s.validateScope(client.Scopes, req.Scope) {
		log.Printf("Invalid scope: %s", req.Scope)
		return "", ErrInvalidScope
	}

	// 5. 生成授权码
	authCode := &model.AuthorizationCode{
		ClientID:    req.ClientID,
		UserID:      userID,
		RedirectURI: req.RedirectURI, // 存储原始URI，不进行编码
		Scope:       req.Scope,
		ExpiresAt:   time.Now().Add(10 * time.Minute), // 授权码10分钟有效
		CreatedAt:   time.Now(),
	}

	if err := s.codeRepo.Create(ctx, authCode); err != nil {
		log.Printf("Failed to create authorization code: %v", err)
		return "", err
	}

	log.Printf("Created authorization code for client %s: %s", req.ClientID, authCode.Code)

	// 6. 构建重定向URL
	redirectURL, err := url.Parse(req.RedirectURI)
	if err != nil {
		log.Printf("Failed to parse redirect URI: %v", err)
		return "", err
	}

	query := redirectURL.Query()
	query.Set("code", authCode.Code) // 授权码已经是base64编码的，不需要额外编码
	if req.State != "" {
		query.Set("state", req.State)
	}

	// 如果响应类型包含 id_token，生成并返回 ID Token
	if req.ResponseType == model.CodeIDTokenResponse {
		// 获取用户信息
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			log.Printf("Failed to get user info: %v", err)
			return "", fmt.Errorf("failed to get user info: %w", err)
		}

		// 生成 ID Token
		idToken, err := s.oidcService.GenerateIDToken(ctx, user, client, req.Nonce)
		if err != nil {
			log.Printf("Failed to generate ID token: %v", err)
			return "", fmt.Errorf("failed to generate ID token: %w", err)
		}
		query.Set("id_token", idToken)
	}

	redirectURL.RawQuery = query.Encode()

	finalURL := redirectURL.String()
	log.Printf("Generated redirect URL: %s", finalURL)

	return finalURL, nil
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

// IssueToken 颁发令牌
func (s *authorizationService) IssueToken(ctx context.Context, req *model.TokenRequest) (*model.TokenResponse, error) {
	// 打印请求参数
	log.Printf("Token request parameters: grant_type=%s, client_id=%s, code=%s, redirect_uri=%s",
		req.GrantType, req.ClientID, req.Code, req.RedirectURI)

	// 验证客户端
	client, err := s.clientRepo.GetByClientID(ctx, req.ClientID)
	if err != nil {
		log.Printf("Error getting client: %v", err)
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	if client == nil {
		log.Printf("Client not found with client_id: %s", req.ClientID)
		return nil, ErrInvalidClient
	}

	// 打印客户端信息（注意不要打印密钥）
	log.Printf("Found client: id=%s, name=%s, type=%s", client.ID, client.Name, client.Type)

	// 验证客户端密钥
	secret, err := s.secretRepo.ValidateSecret(ctx, req.ClientID, req.ClientSecret)
	if err != nil {
		log.Printf("Error validating client secret: %v", err)
		return nil, fmt.Errorf("failed to validate client secret: %w", err)
	}
	if secret == nil {
		log.Printf("Invalid client secret for client_id: %s", req.ClientID)
		return nil, ErrInvalidClient
	}

	// 更新密钥最后使用时间
	if err := s.secretRepo.UpdateLastUsedAt(ctx, secret.ID); err != nil {
		log.Printf("Failed to update secret last used time: %v", err)
	}

	switch req.GrantType {
	case model.GrantTypeAuthorizationCode:
		return s.handleAuthorizationCodeGrant(ctx, req, client)
	case model.GrantTypeRefreshToken:
		return s.handleRefreshTokenGrant(ctx, req, client)
	default:
		log.Printf("Unsupported grant type: %s", req.GrantType)
		return nil, ErrUnsupportedGrantType
	}
}

// handleAuthorizationCodeGrant 处理授权码授权类型
func (s *authorizationService) handleAuthorizationCodeGrant(ctx context.Context, req *model.TokenRequest, client *model.OAuthClient) (*model.TokenResponse, error) {
	// 打印原始授权码
	log.Printf("Received authorization code: %s", req.Code)

	// URL解码授权码
	code, err := url.QueryUnescape(req.Code)
	if err != nil {
		log.Printf("Failed to URL decode authorization code: %v", err)
		return nil, fmt.Errorf("failed to decode authorization code: %w", err)
	}
	log.Printf("URL decoded authorization code: %s", code)

	// 如果授权码末尾没有=，尝试添加
	if !strings.HasSuffix(code, "=") {
		code = code + "="
		log.Printf("Added padding to authorization code: %s", code)
	}

	// 验证授权码
	authCode, err := s.codeRepo.GetByCode(ctx, code)
	if err != nil {
		log.Printf("Error getting authorization code: %v", err)
		return nil, fmt.Errorf("failed to get authorization code: %w", err)
	}
	if authCode == nil {
		log.Printf("Authorization code not found: %s", code)
		return nil, ErrInvalidGrant
	}

	// 验证客户端ID
	if authCode.ClientID != client.ClientID {
		log.Printf("Client ID mismatch: expected %s, got %s", authCode.ClientID, client.ClientID)
		return nil, ErrInvalidGrant
	}

	// 提取基本的重定向URI（去除查询参数）
	requestedURI, err := url.Parse(req.RedirectURI)
	if err != nil {
		log.Printf("Failed to parse request redirect URI: %v", err)
		return nil, fmt.Errorf("failed to parse redirect uri: %w", err)
	}
	requestedURI.RawQuery = ""
	actualURI := requestedURI.String()

	storedURI, err := url.Parse(authCode.RedirectURI)
	if err != nil {
		log.Printf("Failed to parse stored redirect URI: %v", err)
		return nil, fmt.Errorf("failed to parse redirect uri: %w", err)
	}
	storedURI.RawQuery = ""
	expectedURI := storedURI.String()

	if expectedURI != actualURI {
		log.Printf("Redirect URI mismatch: expected %s, got %s", expectedURI, actualURI)
		return nil, ErrInvalidGrant
	}

	// 验证过期时间
	if authCode.ExpiresAt.Before(time.Now()) {
		log.Printf("Authorization code expired at %v", authCode.ExpiresAt)
		return nil, ErrInvalidGrant
	}

	// 生成访问令牌和刷新令牌
	user := &model.User{
		ID:    authCode.UserID,
		AppID: client.AppID,
	}
	tokenPair, err := s.tokenService.GenerateTokenPair(ctx, user, authCode.Scope)
	if err != nil {
		log.Printf("Failed to generate token pair: %v", err)
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	// 删除已使用的授权码
	if err := s.codeRepo.Delete(ctx, authCode.ID); err != nil {
		log.Printf("Failed to delete authorization code: %v", err)
	}

	response := &model.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(tokenPair.AccessTokenExpireIn.Seconds()),
		RefreshToken: tokenPair.RefreshToken,
		Scope:        authCode.Scope,
	}

	// 如果scope包含openid，生成ID Token
	scopes := strings.Split(authCode.Scope, " ")
	for _, scope := range scopes {
		if scope == model.ScopeOpenID {
			// 获取完整的用户信息
			user, err = s.userRepo.GetByID(ctx, authCode.UserID)
			if err != nil {
				log.Printf("Failed to get user info: %v", err)
				return nil, fmt.Errorf("failed to get user info: %w", err)
			}

			idToken, err := s.oidcService.GenerateIDToken(ctx, user, client, req.Nonce)
			if err != nil {
				log.Printf("Failed to generate ID token: %v", err)
				return nil, fmt.Errorf("failed to generate ID token: %w", err)
			}
			response.IDToken = idToken
			break
		}
	}

	return response, nil
}

// handleRefreshTokenGrant 处理刷新令牌授权类型
func (s *authorizationService) handleRefreshTokenGrant(ctx context.Context, req *model.TokenRequest, client *model.OAuthClient) (*model.TokenResponse, error) {
	log.Printf("Processing refresh token grant for client_id: %s", req.ClientID)

	// 验证刷新令牌
	claims, err := s.tokenService.ValidateToken(ctx, req.RefreshToken, model.RefreshToken)
	if err != nil {
		log.Printf("Failed to validate refresh token: %v", err)
		return nil, ErrInvalidGrant
	}

	// 验证客户端ID与令牌中的AppID是否匹配
	if claims.AppID != client.AppID {
		log.Printf("Client AppID mismatch: token AppID=%s, client AppID=%s", claims.AppID, client.AppID)
		return nil, ErrInvalidGrant
	}

	// 使用刷新令牌获取新的令牌对
	tokenPair, err := s.tokenService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		log.Printf("Failed to refresh token: %v", err)
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	log.Printf("Successfully refreshed tokens for client_id: %s", req.ClientID)

	// 获取用户的权限范围
	scope := strings.Join(client.Scopes, " ")

	return &model.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(tokenPair.AccessTokenExpireIn.Seconds()),
		RefreshToken: tokenPair.RefreshToken,
		Scope:        scope,
	}, nil
}
