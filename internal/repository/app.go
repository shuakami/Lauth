package repository

import (
	"context"
	"errors"

	"lauth/internal/model"

	"gorm.io/gorm"
)

// AppRepository 应用仓储接口
type AppRepository interface {
	Create(ctx context.Context, app *model.App) error
	GetByID(ctx context.Context, id string) (*model.App, error)
	GetByAppKey(ctx context.Context, appKey string) (*model.App, error)
	Update(ctx context.Context, app *model.App) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]model.App, int64, error)
}

// appRepository 应用仓储实现
type appRepository struct {
	db *gorm.DB
}

// NewAppRepository 创建应用仓储实例
func NewAppRepository(db *gorm.DB) AppRepository {
	return &appRepository{db: db}
}

// Create 创建应用
func (r *appRepository) Create(ctx context.Context, app *model.App) error {
	return r.db.WithContext(ctx).Create(app).Error
}

// GetByID 通过ID获取应用
func (r *appRepository) GetByID(ctx context.Context, id string) (*model.App, error) {
	var app model.App
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&app).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &app, nil
}

// GetByAppKey 通过AppKey获取应用
func (r *appRepository) GetByAppKey(ctx context.Context, appKey string) (*model.App, error) {
	var app model.App
	if err := r.db.WithContext(ctx).Where("app_key = ?", appKey).First(&app).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &app, nil
}

// Update 更新应用
func (r *appRepository) Update(ctx context.Context, app *model.App) error {
	return r.db.WithContext(ctx).Save(app).Error
}

// Delete 删除应用
func (r *appRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.App{}, "id = ?", id).Error
}

// List 获取应用列表
func (r *appRepository) List(ctx context.Context, offset, limit int) ([]model.App, int64, error) {
	var apps []model.App
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.App{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&apps).Error; err != nil {
		return nil, 0, err
	}

	return apps, total, nil
}
