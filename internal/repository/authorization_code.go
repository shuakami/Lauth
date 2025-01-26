package repository

import (
	"context"
	"time"

	"lauth/internal/model"

	"gorm.io/gorm"
)

// AuthorizationCodeRepository 授权码仓储接口
type AuthorizationCodeRepository interface {
	// Create 创建授权码
	Create(ctx context.Context, code *model.AuthorizationCode) error
	// GetByCode 通过授权码获取
	GetByCode(ctx context.Context, code string) (*model.AuthorizationCode, error)
	// Delete 删除授权码
	Delete(ctx context.Context, code string) error
	// DeleteExpired 删除过期的授权码
	DeleteExpired(ctx context.Context) error
}

// authorizationCodeRepository 授权码仓储实现
type authorizationCodeRepository struct {
	db *gorm.DB
}

// NewAuthorizationCodeRepository 创建授权码仓储实例
func NewAuthorizationCodeRepository(db *gorm.DB) AuthorizationCodeRepository {
	return &authorizationCodeRepository{db: db}
}

// Create 创建授权码
func (r *authorizationCodeRepository) Create(ctx context.Context, code *model.AuthorizationCode) error {
	return r.db.WithContext(ctx).Create(code).Error
}

// GetByCode 通过授权码获取
func (r *authorizationCodeRepository) GetByCode(ctx context.Context, code string) (*model.AuthorizationCode, error) {
	var authCode model.AuthorizationCode
	err := r.db.WithContext(ctx).Where("code = ? AND expires_at > ?", code, time.Now()).First(&authCode).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &authCode, err
}

// Delete 删除授权码
func (r *authorizationCodeRepository) Delete(ctx context.Context, code string) error {
	return r.db.WithContext(ctx).Where("code = ?", code).Delete(&model.AuthorizationCode{}).Error
}

// DeleteExpired 删除过期的授权码
func (r *authorizationCodeRepository) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at <= ?", time.Now()).Delete(&model.AuthorizationCode{}).Error
}
