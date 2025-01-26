package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"time"

	"lauth/internal/model"
	"lauth/internal/repository"

	"github.com/google/uuid"
)

var (
	// ErrClientExists 客户端已存在
	ErrClientExists = errors.New("client already exists")
	// ErrClientNotFound 客户端不存在
	ErrClientNotFound = errors.New("client not found")
)

// OAuthClientService OAuth客户端服务接口
type OAuthClientService interface {
	// CreateClient 创建OAuth客户端
	CreateClient(ctx context.Context, appID string, req *model.CreateOAuthClientRequest) (*model.OAuthClientResponse, error)

	// UpdateClient 更新OAuth客户端
	UpdateClient(ctx context.Context, id string, req *model.UpdateOAuthClientRequest) (*model.OAuthClientResponse, error)

	// DeleteClient 删除OAuth客户端
	DeleteClient(ctx context.Context, id string) error

	// GetClient 获取OAuth客户端
	GetClient(ctx context.Context, id string) (*model.OAuthClientResponse, error)

	// ListClients 获取OAuth客户端列表
	ListClients(ctx context.Context, appID string, page, size int) ([]*model.OAuthClientResponse, int64, error)

	// ValidateClient 验证客户端凭证
	ValidateClient(ctx context.Context, clientID, clientSecret string) (*model.OAuthClient, error)

	// CreateClientSecret 创建客户端秘钥
	CreateClientSecret(ctx context.Context, clientID string, req *model.CreateClientSecretRequest) (*model.ClientSecretResponse, error)

	// DeleteClientSecret 删除客户端秘钥
	DeleteClientSecret(ctx context.Context, clientID, secretID string) error

	// ListClientSecrets 获取客户端秘钥列表
	ListClientSecrets(ctx context.Context, clientID string) ([]*model.ClientSecretResponse, error)
}

// oauthClientService OAuth客户端服务实现
type oauthClientService struct {
	clientRepo repository.OAuthClientRepository
	secretRepo repository.OAuthClientSecretRepository
}

// NewOAuthClientService 创建OAuth客户端服务实例
func NewOAuthClientService(
	clientRepo repository.OAuthClientRepository,
	secretRepo repository.OAuthClientSecretRepository,
) OAuthClientService {
	return &oauthClientService{
		clientRepo: clientRepo,
		secretRepo: secretRepo,
	}
}

// generateClientCredentials 生成客户端凭证
func generateClientCredentials() (clientID, clientSecret string, err error) {
	// 生成32字节的随机数作为客户端ID
	idBytes := make([]byte, 32)
	if _, err := rand.Read(idBytes); err != nil {
		return "", "", err
	}
	clientID = base64.URLEncoding.EncodeToString(idBytes)

	// 生成64字节的随机数作为客户端密钥
	secretBytes := make([]byte, 64)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", err
	}
	clientSecret = base64.URLEncoding.EncodeToString(secretBytes)

	return clientID, clientSecret, nil
}

// toOAuthClientResponse 转换为OAuth客户端响应
func toOAuthClientResponse(client *model.OAuthClient) *model.OAuthClientResponse {
	return &model.OAuthClientResponse{
		ID:           client.ID,
		AppID:        client.AppID,
		Name:         client.Name,
		ClientID:     client.ClientID,
		Type:         client.Type,
		GrantTypes:   client.GrantTypes,
		RedirectURIs: client.RedirectURIs,
		Scopes:       client.Scopes,
		Status:       client.Status,
		CreatedAt:    client.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    client.UpdatedAt.Format(time.RFC3339),
	}
}

// toClientSecretResponse 转换为客户端秘钥响应
func toClientSecretResponse(secret *model.OAuthClientSecret) *model.ClientSecretResponse {
	return &model.ClientSecretResponse{
		ID:          secret.ID,
		Secret:      secret.Secret,
		Description: secret.Description,
		LastUsedAt:  secret.LastUsedAt.Format(time.RFC3339),
		ExpiresAt:   secret.ExpiresAt.Format(time.RFC3339),
		CreatedAt:   secret.CreatedAt.Format(time.RFC3339),
	}
}

// CreateClient 创建OAuth客户端
func (s *oauthClientService) CreateClient(ctx context.Context, appID string, req *model.CreateOAuthClientRequest) (*model.OAuthClientResponse, error) {
	// 生成客户端ID
	clientID := uuid.New().String()

	// 检查客户端ID是否已存在
	existingClient, err := s.clientRepo.GetByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if existingClient != nil {
		return nil, ErrClientExists
	}

	// 创建客户端
	now := time.Now()
	client := &model.OAuthClient{
		ID:           uuid.New().String(),
		AppID:        appID,
		Name:         req.Name,
		ClientID:     clientID,
		Type:         req.Type,
		GrantTypes:   req.GrantTypes,
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
		Status:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.clientRepo.Create(ctx, client); err != nil {
		return nil, err
	}

	return toOAuthClientResponse(client), nil
}

// UpdateClient 更新OAuth客户端
func (s *oauthClientService) UpdateClient(ctx context.Context, id string, req *model.UpdateOAuthClientRequest) (*model.OAuthClientResponse, error) {
	// 获取现有客户端
	client, err := s.clientRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrClientNotFound
	}

	// 更新字段
	if req.Name != "" {
		client.Name = req.Name
	}
	if len(req.GrantTypes) > 0 {
		client.GrantTypes = req.GrantTypes
	}
	if len(req.RedirectURIs) > 0 {
		client.RedirectURIs = req.RedirectURIs
	}
	if len(req.Scopes) > 0 {
		client.Scopes = req.Scopes
	}
	if req.Status != nil {
		client.Status = *req.Status
	}
	client.UpdatedAt = time.Now()

	if err := s.clientRepo.Update(ctx, client); err != nil {
		return nil, err
	}

	return toOAuthClientResponse(client), nil
}

// DeleteClient 删除OAuth客户端
func (s *oauthClientService) DeleteClient(ctx context.Context, id string) error {
	return s.clientRepo.Delete(ctx, id)
}

// GetClient 获取OAuth客户端
func (s *oauthClientService) GetClient(ctx context.Context, id string) (*model.OAuthClientResponse, error) {
	client, err := s.clientRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrClientNotFound
	}

	return toOAuthClientResponse(client), nil
}

// ListClients 获取OAuth客户端列表
func (s *oauthClientService) ListClients(ctx context.Context, appID string, page, size int) ([]*model.OAuthClientResponse, int64, error) {
	// 获取总数
	total, err := s.clientRepo.Count(ctx, appID)
	if err != nil {
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * size

	// 获取列表
	clients, err := s.clientRepo.List(ctx, appID, offset, size)
	if err != nil {
		return nil, 0, err
	}

	// 转换响应
	var responses []*model.OAuthClientResponse
	for _, client := range clients {
		responses = append(responses, toOAuthClientResponse(client))
	}

	return responses, total, nil
}

// ValidateClient 验证客户端凭证
func (s *oauthClientService) ValidateClient(ctx context.Context, clientID, clientSecret string) (*model.OAuthClient, error) {
	// 获取客户端
	client, err := s.clientRepo.GetByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrClientNotFound
	}

	// 验证状态
	if !client.Status {
		return nil, errors.New("client is disabled")
	}

	// 验证秘钥
	secret, err := s.secretRepo.ValidateSecret(ctx, clientID, clientSecret)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, errors.New("invalid client credentials")
	}

	// 更新最后使用时间
	if err := s.secretRepo.UpdateLastUsedAt(ctx, secret.ID); err != nil {
		log.Printf("Failed to update secret last used time: %v", err)
	}

	return client, nil
}

// CreateClientSecret 创建客户端秘钥
func (s *oauthClientService) CreateClientSecret(ctx context.Context, clientID string, req *model.CreateClientSecretRequest) (*model.ClientSecretResponse, error) {
	// 检查客户端是否存在
	client, err := s.clientRepo.GetByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrClientNotFound
	}

	// 生成秘钥
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, err
	}
	secret := base64.URLEncoding.EncodeToString(secretBytes)

	// 创建秘钥记录
	now := time.Now()
	clientSecret := &model.OAuthClientSecret{
		ClientID:    clientID,
		Secret:      secret,
		Description: req.Description,
		LastUsedAt:  now,
		ExpiresAt:   now.Add(time.Duration(req.ExpiresIn) * time.Second),
		CreatedAt:   now,
	}

	if err := s.secretRepo.Create(ctx, clientSecret); err != nil {
		return nil, err
	}

	return toClientSecretResponse(clientSecret), nil
}

// DeleteClientSecret 删除客户端秘钥
func (s *oauthClientService) DeleteClientSecret(ctx context.Context, clientID, secretID string) error {
	// 检查秘钥是否存在
	secret, err := s.secretRepo.GetByID(ctx, secretID)
	if err != nil {
		return err
	}
	if secret == nil {
		return errors.New("secret not found")
	}

	// 检查秘钥是否属于该客户端
	if secret.ClientID != clientID {
		return errors.New("secret does not belong to client")
	}

	return s.secretRepo.Delete(ctx, secretID)
}

// ListClientSecrets 获取客户端秘钥列表
func (s *oauthClientService) ListClientSecrets(ctx context.Context, clientID string) ([]*model.ClientSecretResponse, error) {
	// 获取所有有效秘钥
	secrets, err := s.secretRepo.GetByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// 转换响应
	var responses []*model.ClientSecretResponse
	for _, secret := range secrets {
		resp := toClientSecretResponse(secret)
		resp.Secret = "" // 不返回秘钥内容
		responses = append(responses, resp)
	}

	return responses, nil
}
