package repository

import (
	"context"
	"lauth/internal/model"
)

// PluginStatusRepository 插件状态存储接口
type PluginStatusRepository interface {
	// SaveStatus 保存插件状态
	SaveStatus(ctx context.Context, status *model.PluginStatus) error

	// GetStatus 获取插件状态
	GetStatus(ctx context.Context, appID, userID, action, plugin string) (*model.PluginStatus, error)

	// GetStatusByIdentifier 通过标识符获取插件状态
	GetStatusByIdentifier(ctx context.Context, appID, identifier, identifierType, action, plugin string) (*model.PluginStatus, error)

	// ListStatus 获取用户在指定操作下的所有插件状态
	ListStatus(ctx context.Context, appID, userID, action string) ([]*model.PluginStatus, error)

	// ListStatusByIdentifier 通过标识符获取指定操作下的所有插件状态
	ListStatusByIdentifier(ctx context.Context, appID, identifier, identifierType, action string) ([]*model.PluginStatus, error)

	// DeleteStatus 删除插件状态
	DeleteStatus(ctx context.Context, appID, userID, action, plugin string) error

	// DeleteStatusByIdentifier 通过标识符删除插件状态
	DeleteStatusByIdentifier(ctx context.Context, appID, identifier, identifierType, action, plugin string) error
}
