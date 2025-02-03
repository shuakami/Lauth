package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lauth/internal/model"
)

// pluginStatusRepository 插件状态存储实现
type pluginStatusRepository struct {
	db *gorm.DB
}

// NewPluginStatusRepository 创建插件状态存储实例
func NewPluginStatusRepository(db *gorm.DB) PluginStatusRepository {
	return &pluginStatusRepository{
		db: db,
	}
}

// SaveStatus 保存插件状态
func (r *pluginStatusRepository) SaveStatus(ctx context.Context, status *model.PluginStatus) error {
	if status.ID == "" {
		status.ID = uuid.New().String()
	}
	if status.CreatedAt.IsZero() {
		status.CreatedAt = time.Now()
	}
	status.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).Save(status).Error
}

// GetStatus 获取插件状态
func (r *pluginStatusRepository) GetStatus(ctx context.Context, appID, userID, action, plugin string) (*model.PluginStatus, error) {
	var status model.PluginStatus
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND user_id = ? AND action = ? AND plugin = ?", appID, userID, action, plugin).
		First(&status).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &status, err
}

// ListStatus 获取用户在指定操作下的所有插件状态
func (r *pluginStatusRepository) ListStatus(ctx context.Context, appID, userID, action string) ([]*model.PluginStatus, error) {
	var statuses []*model.PluginStatus
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND user_id = ? AND action = ?", appID, userID, action).
		Find(&statuses).Error
	return statuses, err
}

// DeleteStatus 删除插件状态
func (r *pluginStatusRepository) DeleteStatus(ctx context.Context, appID, userID, action, plugin string) error {
	return r.db.WithContext(ctx).
		Where("app_id = ? AND user_id = ? AND action = ? AND plugin = ?", appID, userID, action, plugin).
		Delete(&model.PluginStatus{}).Error
}

// GetStatusByIdentifier 通过标识符获取插件状态
func (r *pluginStatusRepository) GetStatusByIdentifier(ctx context.Context, appID, identifier, identifierType, action, plugin string) (*model.PluginStatus, error) {
	var status model.PluginStatus
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND identifier = ? AND identifier_type = ? AND action = ? AND plugin = ?",
			appID, identifier, identifierType, action, plugin).
		First(&status).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &status, err
}

// ListStatusByIdentifier 通过标识符获取指定操作下的所有插件状态
func (r *pluginStatusRepository) ListStatusByIdentifier(ctx context.Context, appID, identifier, identifierType, action string) ([]*model.PluginStatus, error) {
	var statuses []*model.PluginStatus
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND identifier = ? AND identifier_type = ? AND action = ?",
			appID, identifier, identifierType, action).
		Find(&statuses).Error
	return statuses, err
}

// DeleteStatusByIdentifier 通过标识符删除插件状态
func (r *pluginStatusRepository) DeleteStatusByIdentifier(ctx context.Context, appID, identifier, identifierType, action, plugin string) error {
	return r.db.WithContext(ctx).
		Where("app_id = ? AND identifier = ? AND identifier_type = ? AND action = ? AND plugin = ?",
			appID, identifier, identifierType, action, plugin).
		Delete(&model.PluginStatus{}).Error
}
