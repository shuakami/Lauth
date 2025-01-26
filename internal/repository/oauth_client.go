package repository

import (
	"context"

	"lauth/internal/model"

	"gorm.io/gorm"
)

// OAuthClientRepository OAuth客户端仓储接口
type OAuthClientRepository interface {
	// Create 创建OAuth客户端
	Create(ctx context.Context, client *model.OAuthClient) error

	// Update 更新OAuth客户端
	Update(ctx context.Context, client *model.OAuthClient) error

	// Delete 删除OAuth客户端
	Delete(ctx context.Context, id string) error

	// GetByID 通过ID获取OAuth客户端
	GetByID(ctx context.Context, id string) (*model.OAuthClient, error)

	// GetByClientID 通过客户端ID获取OAuth客户端
	GetByClientID(ctx context.Context, clientID string) (*model.OAuthClient, error)

	// List 获取OAuth客户端列表
	List(ctx context.Context, appID string, offset, limit int) ([]*model.OAuthClient, error)

	// Count 获取OAuth客户端总数
	Count(ctx context.Context, appID string) (int64, error)
}

// oauthClientRepository OAuth客户端仓储实现
type oauthClientRepository struct {
	db *gorm.DB
}

// NewOAuthClientRepository 创建OAuth客户端仓储实例
func NewOAuthClientRepository(db *gorm.DB) OAuthClientRepository {
	return &oauthClientRepository{db: db}
}

// Create 创建OAuth客户端
func (r *oauthClientRepository) Create(ctx context.Context, client *model.OAuthClient) error {
	return r.db.WithContext(ctx).Create(client).Error
}

// Update 更新OAuth客户端
func (r *oauthClientRepository) Update(ctx context.Context, client *model.OAuthClient) error {
	return r.db.WithContext(ctx).Save(client).Error
}

// Delete 删除OAuth客户端
func (r *oauthClientRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.OAuthClient{}, "id = ?", id).Error
}

// GetByID 通过ID获取OAuth客户端
func (r *oauthClientRepository) GetByID(ctx context.Context, id string) (*model.OAuthClient, error) {
	var client model.OAuthClient
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&client).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

// GetByClientID 通过客户端ID获取OAuth客户端
func (r *oauthClientRepository) GetByClientID(ctx context.Context, clientID string) (*model.OAuthClient, error) {
	var client model.OAuthClient
	err := r.db.WithContext(ctx).Where("client_id = ?", clientID).First(&client).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

// List 获取OAuth客户端列表
func (r *oauthClientRepository) List(ctx context.Context, appID string, offset, limit int) ([]*model.OAuthClient, error) {
	var clients []*model.OAuthClient
	err := r.db.WithContext(ctx).
		Where("app_id = ?", appID).
		Offset(offset).
		Limit(limit).
		Find(&clients).Error
	return clients, err
}

// Count 获取OAuth客户端总数
func (r *oauthClientRepository) Count(ctx context.Context, appID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.OAuthClient{}).Where("app_id = ?", appID).Count(&count).Error
	return count, err
}
