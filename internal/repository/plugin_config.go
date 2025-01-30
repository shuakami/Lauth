package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lauth/internal/model"
)

// PluginConfigRepository 插件配置存储接口
type PluginConfigRepository interface {
	// SaveConfig 保存插件配置
	SaveConfig(ctx context.Context, config *model.PluginConfig) error

	// GetConfig 获取插件配置
	GetConfig(ctx context.Context, appID, name string) (*model.PluginConfig, error)

	// ListConfigs 获取应用的所有插件配置
	ListConfigs(ctx context.Context, appID string) ([]*model.PluginConfig, error)

	// DeleteConfig 删除插件配置
	DeleteConfig(ctx context.Context, appID, name string) error
}

// pluginConfigRepository 插件配置存储实现
type pluginConfigRepository struct {
	db *gorm.DB
}

// NewPluginConfigRepository 创建插件配置存储实例
func NewPluginConfigRepository(db *gorm.DB) PluginConfigRepository {
	return &pluginConfigRepository{
		db: db,
	}
}

// SaveConfig 保存插件配置
func (r *pluginConfigRepository) SaveConfig(ctx context.Context, config *model.PluginConfig) error {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}
	config.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).Save(config).Error
}

// GetConfig 获取插件配置
func (r *pluginConfigRepository) GetConfig(ctx context.Context, appID, name string) (*model.PluginConfig, error) {
	var config model.PluginConfig
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND name = ?", appID, name).
		First(&config).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &config, err
}

// ListConfigs 获取应用的所有插件配置
func (r *pluginConfigRepository) ListConfigs(ctx context.Context, appID string) ([]*model.PluginConfig, error) {
	var configs []*model.PluginConfig
	err := r.db.WithContext(ctx).
		Where("app_id = ?", appID).
		Find(&configs).Error
	return configs, err
}

// DeleteConfig 删除插件配置
func (r *pluginConfigRepository) DeleteConfig(ctx context.Context, appID, name string) error {
	return r.db.WithContext(ctx).
		Where("app_id = ? AND name = ?", appID, name).
		Delete(&model.PluginConfig{}).Error
}
