package repository

import (
	"context"
	"errors"

	"lauth/internal/model"

	"gorm.io/gorm"
)

// PermissionRepository 权限仓储接口
type PermissionRepository interface {
	// 基础CRUD
	Create(ctx context.Context, permission *model.Permission) error
	GetByID(ctx context.Context, id string) (*model.Permission, error)
	Update(ctx context.Context, permission *model.Permission) error
	Delete(ctx context.Context, id string) error

	// 查询方法
	GetByName(ctx context.Context, appID, name string) (*model.Permission, error)
	List(ctx context.Context, appID string, offset, limit int) ([]model.Permission, int64, error)

	// 高级查询
	GetByResourceAndAction(ctx context.Context, appID string, resourceType model.ResourceType, action model.ActionType) (*model.Permission, error)
	ListByResourceType(ctx context.Context, appID string, resourceType model.ResourceType) ([]model.Permission, error)
	ListByRole(ctx context.Context, roleID string) ([]model.Permission, error)
}

// permissionRepository 权限仓储实现
type permissionRepository struct {
	db *gorm.DB
}

// NewPermissionRepository 创建权限仓储实例
func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

// Create 创建权限
func (r *permissionRepository) Create(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Create(permission).Error
}

// GetByID 通过ID获取权限
func (r *permissionRepository) GetByID(ctx context.Context, id string) (*model.Permission, error) {
	var permission model.Permission
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&permission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &permission, nil
}

// GetByName 通过名称获取权限
func (r *permissionRepository) GetByName(ctx context.Context, appID, name string) (*model.Permission, error) {
	var permission model.Permission
	if err := r.db.WithContext(ctx).Where("app_id = ? AND name = ?", appID, name).First(&permission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &permission, nil
}

// Update 更新权限
func (r *permissionRepository) Update(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Save(permission).Error
}

// Delete 删除权限
func (r *permissionRepository) Delete(ctx context.Context, id string) error {
	// 开启事务
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除权限的角色关联
		if err := tx.Where("permission_id = ?", id).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}

		// 删除权限
		return tx.Delete(&model.Permission{}, "id = ?", id).Error
	})
}

// List 获取权限列表
func (r *permissionRepository) List(ctx context.Context, appID string, offset, limit int) ([]model.Permission, int64, error) {
	var permissions []model.Permission
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.Permission{}).Where("app_id = ?", appID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Where("app_id = ?", appID).Offset(offset).Limit(limit).Find(&permissions).Error; err != nil {
		return nil, 0, err
	}

	return permissions, total, nil
}

// GetByResourceAndAction 通过资源类型和操作类型获取权限
func (r *permissionRepository) GetByResourceAndAction(ctx context.Context, appID string, resourceType model.ResourceType, action model.ActionType) (*model.Permission, error) {
	var permission model.Permission
	if err := r.db.WithContext(ctx).
		Where("app_id = ? AND resource_type = ? AND action = ?", appID, resourceType, action).
		First(&permission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &permission, nil
}

// ListByResourceType 获取指定资源类型的所有权限
func (r *permissionRepository) ListByResourceType(ctx context.Context, appID string, resourceType model.ResourceType) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND resource_type = ?", appID, resourceType).
		Find(&permissions).Error
	return permissions, err
}

// ListByRole 获取角色的所有权限
func (r *permissionRepository) ListByRole(ctx context.Context, roleID string) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&permissions).Error
	return permissions, err
}
