package repository

import (
	"context"
	"time"

	"lauth/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OAuthClientSecretRepository OAuth客户端秘钥仓储接口
type OAuthClientSecretRepository interface {
	// Create 创建秘钥
	Create(ctx context.Context, secret *model.OAuthClientSecret) error
	// Delete 删除秘钥
	Delete(ctx context.Context, id string) error
	// GetByID 通过ID获取秘钥
	GetByID(ctx context.Context, id string) (*model.OAuthClientSecret, error)
	// GetByClientID 获取客户端的所有有效秘钥
	GetByClientID(ctx context.Context, clientID string) ([]*model.OAuthClientSecret, error)
	// ValidateSecret 验证秘钥
	ValidateSecret(ctx context.Context, clientID, secret string) (*model.OAuthClientSecret, error)
	// UpdateLastUsedAt 更新最后使用时间
	UpdateLastUsedAt(ctx context.Context, id string) error
}

// oauthClientSecretRepository OAuth客户端秘钥仓储实现
type oauthClientSecretRepository struct {
	db *gorm.DB
}

// NewOAuthClientSecretRepository 创建OAuth客户端秘钥仓储实例
func NewOAuthClientSecretRepository(db *gorm.DB) OAuthClientSecretRepository {
	return &oauthClientSecretRepository{db: db}
}

// Create 创建秘钥
func (r *oauthClientSecretRepository) Create(ctx context.Context, secret *model.OAuthClientSecret) error {
	if secret.ID == "" {
		secret.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(secret).Error
}

// Delete 删除秘钥
func (r *oauthClientSecretRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.OAuthClientSecret{}, "id = ?", id).Error
}

// GetByID 通过ID获取秘钥
func (r *oauthClientSecretRepository) GetByID(ctx context.Context, id string) (*model.OAuthClientSecret, error) {
	var secret model.OAuthClientSecret
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&secret).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &secret, err
}

// GetByClientID 获取客户端的所有有效秘钥
func (r *oauthClientSecretRepository) GetByClientID(ctx context.Context, clientID string) ([]*model.OAuthClientSecret, error) {
	var secrets []*model.OAuthClientSecret
	err := r.db.WithContext(ctx).
		Where("client_id = ? AND expires_at > ?", clientID, time.Now()).
		Find(&secrets).Error
	return secrets, err
}

// ValidateSecret 验证秘钥
func (r *oauthClientSecretRepository) ValidateSecret(ctx context.Context, clientID, secret string) (*model.OAuthClientSecret, error) {
	var clientSecret model.OAuthClientSecret
	err := r.db.WithContext(ctx).
		Where("client_id = ? AND secret = ? AND expires_at > ?", clientID, secret, time.Now()).
		First(&clientSecret).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &clientSecret, err
}

// UpdateLastUsedAt 更新最后使用时间
func (r *oauthClientSecretRepository) UpdateLastUsedAt(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&model.OAuthClientSecret{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}
