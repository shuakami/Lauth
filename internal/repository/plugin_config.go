package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// PluginUserConfigRepository 插件用户配置存储接口
type PluginUserConfigRepository interface {
	// GetUserConfig 获取用户配置
	GetUserConfig(ctx context.Context, appID, userID, plugin string) (*model.PluginUserConfig, error)

	// SaveUserConfig 保存用户配置
	SaveUserConfig(ctx context.Context, config *model.PluginUserConfig) error

	// DeleteUserConfig 删除用户配置
	DeleteUserConfig(ctx context.Context, appID, userID, plugin string) error
}

// PluginVerificationRecordRepository 插件验证记录存储接口
type PluginVerificationRecordRepository interface {
	// SaveRecord 保存验证记录
	SaveRecord(ctx context.Context, record *model.PluginVerificationRecord) error

	// GetLastRecord 获取最近的验证记录
	GetLastRecord(ctx context.Context, appID, userID, plugin, action string) (*model.PluginVerificationRecord, error)

	// ListRecords 获取验证记录列表
	ListRecords(ctx context.Context, appID, userID, plugin string, limit int) ([]*model.PluginVerificationRecord, error)
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
	// 生成ID（如果没有）
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	// 更新时间戳
	now := time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	// 使用 ON CONFLICT (app_id, name) DO UPDATE 更新现有记录
	return r.db.WithContext(ctx).
		Table("plugin_configs").
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "app_id"},
				{Name: "name"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"config",
				"required",
				"stage",
				"actions",
				"enabled",
				"updated_at",
			}),
		}).
		Create(config).Error
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

// pluginUserConfigRepository 插件用户配置存储实现
type pluginUserConfigRepository struct {
	db *gorm.DB
}

// NewPluginUserConfigRepository 创建插件用户配置存储实例
func NewPluginUserConfigRepository(db *gorm.DB) PluginUserConfigRepository {
	return &pluginUserConfigRepository{
		db: db,
	}
}

// GetUserConfig 获取用户配置
func (r *pluginUserConfigRepository) GetUserConfig(ctx context.Context, appID, userID, plugin string) (*model.PluginUserConfig, error) {
	var config model.PluginUserConfig
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND user_id = ? AND plugin = ?", appID, userID, plugin).
		First(&config).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &config, err
}

// SaveUserConfig 保存用户配置
func (r *pluginUserConfigRepository) SaveUserConfig(ctx context.Context, config *model.PluginUserConfig) error {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}
	config.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "app_id"},
				{Name: "user_id"},
				{Name: "plugin"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"config",
				"updated_at",
			}),
		}).
		Create(config).Error
}

// DeleteUserConfig 删除用户配置
func (r *pluginUserConfigRepository) DeleteUserConfig(ctx context.Context, appID, userID, plugin string) error {
	return r.db.WithContext(ctx).
		Where("app_id = ? AND user_id = ? AND plugin = ?", appID, userID, plugin).
		Delete(&model.PluginUserConfig{}).Error
}

// pluginVerificationRecordRepository 插件验证记录存储实现
type pluginVerificationRecordRepository struct {
	db *gorm.DB
}

// NewPluginVerificationRecordRepository 创建插件验证记录存储实例
func NewPluginVerificationRecordRepository(db *gorm.DB) PluginVerificationRecordRepository {
	return &pluginVerificationRecordRepository{
		db: db,
	}
}

// SaveRecord 保存验证记录
func (r *pluginVerificationRecordRepository) SaveRecord(ctx context.Context, record *model.PluginVerificationRecord) error {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	record.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).Save(record).Error
}

// GetLastRecord 获取最近的验证记录
func (r *pluginVerificationRecordRepository) GetLastRecord(ctx context.Context, appID, userID, plugin, action string) (*model.PluginVerificationRecord, error) {
	var record model.PluginVerificationRecord
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND user_id = ? AND plugin = ? AND action = ?", appID, userID, plugin, action).
		Order("verified_at DESC").
		First(&record).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &record, err
}

// ListRecords 获取验证记录列表
func (r *pluginVerificationRecordRepository) ListRecords(ctx context.Context, appID, userID, plugin string, limit int) ([]*model.PluginVerificationRecord, error) {
	var records []*model.PluginVerificationRecord
	query := r.db.WithContext(ctx).
		Where("app_id = ? AND user_id = ? AND plugin = ?", appID, userID, plugin).
		Order("verified_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&records).Error
	return records, err
}
