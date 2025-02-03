package repository

import (
	"context"
	"time"

	"lauth/internal/model"

	"gorm.io/gorm"
)

// LoginLocationRepository 登录位置仓储接口
type LoginLocationRepository interface {
	Create(ctx context.Context, location *model.LoginLocation) error
	GetByID(ctx context.Context, id string) (*model.LoginLocation, error)
	GetLatestByUserID(ctx context.Context, userID string, limit int) ([]*model.LoginLocation, error)
	GetByUserIDAndTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*model.LoginLocation, error)
}

// loginLocationRepository 登录位置仓储实现
type loginLocationRepository struct {
	db *gorm.DB
}

// NewLoginLocationRepository 创建登录位置仓储实例
func NewLoginLocationRepository(db *gorm.DB) LoginLocationRepository {
	return &loginLocationRepository{db: db}
}

// Create 创建登录位置记录
func (r *loginLocationRepository) Create(ctx context.Context, location *model.LoginLocation) error {
	return r.db.WithContext(ctx).Create(location).Error
}

// GetByID 根据ID获取登录位置记录
func (r *loginLocationRepository) GetByID(ctx context.Context, id string) (*model.LoginLocation, error) {
	var location model.LoginLocation
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&location).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &location, err
}

// GetLatestByUserID 获取用户最近的登录位置记录
func (r *loginLocationRepository) GetLatestByUserID(ctx context.Context, userID string, limit int) ([]*model.LoginLocation, error) {
	var locations []*model.LoginLocation
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("login_time DESC").
		Limit(limit).
		Find(&locations).Error
	return locations, err
}

// GetByUserIDAndTimeRange 获取用户指定时间范围内的登录位置记录
func (r *loginLocationRepository) GetByUserIDAndTimeRange(ctx context.Context, userID string, start, end time.Time) ([]*model.LoginLocation, error) {
	var locations []*model.LoginLocation
	err := r.db.WithContext(ctx).
		Select("id, ip, country, province, city, isp, login_time").
		Where("user_id = ? AND login_time BETWEEN ? AND ?", userID, start, end).
		Order("login_time DESC").
		Find(&locations).Error
	return locations, err
}
