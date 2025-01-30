package repository

import (
	"context"
	"errors"

	"lauth/internal/model"

	"gorm.io/gorm"
)

// RoleRepository 角色仓储接口
type RoleRepository interface {
	// 基础CRUD
	Create(ctx context.Context, role *model.Role) error
	GetByID(ctx context.Context, id string) (*model.Role, error)
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, id string) error

	// 查询方法
	GetByName(ctx context.Context, appID, name string) (*model.Role, error)
	List(ctx context.Context, appID string, offset, limit int) ([]model.Role, int64, error)

	// 角色-权限关联
	AddPermissions(ctx context.Context, roleID string, permissionIDs []string) error
	RemovePermissions(ctx context.Context, roleID string, permissionIDs []string) error
	GetPermissions(ctx context.Context, roleID string) ([]model.Permission, error)

	// 用户-角色关联
	AddUsers(ctx context.Context, roleID string, userIDs []string) error
	RemoveUsers(ctx context.Context, roleID string, userIDs []string) error
	GetUsers(ctx context.Context, roleID string) ([]model.User, error)
	GetUserRoles(ctx context.Context, userID, appID string) ([]model.Role, error)
}

// roleRepository 角色仓储实现
type roleRepository struct {
	db *gorm.DB
}

// NewRoleRepository 创建角色仓储实例
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

// Create 创建角色
func (r *roleRepository) Create(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// GetByID 通过ID获取角色
func (r *roleRepository) GetByID(ctx context.Context, id string) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// GetByName 通过名称获取角色
func (r *roleRepository) GetByName(ctx context.Context, appID, name string) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Where("app_id = ? AND name = ?", appID, name).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// Update 更新角色
func (r *roleRepository) Update(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

// Delete 删除角色
func (r *roleRepository) Delete(ctx context.Context, id string) error {
	// 开启事务
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除角色的权限关联
		if err := tx.Where("role_id = ?", id).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}

		// 删除角色的用户关联
		if err := tx.Where("role_id = ?", id).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}

		// 删除角色
		return tx.Delete(&model.Role{}, "id = ?", id).Error
	})
}

// List 获取角色列表
func (r *roleRepository) List(ctx context.Context, appID string, offset, limit int) ([]model.Role, int64, error) {
	var roles []model.Role
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.Role{}).Where("app_id = ?", appID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Where("app_id = ?", appID).Offset(offset).Limit(limit).Find(&roles).Error; err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}

// AddPermissions 为角色添加权限
func (r *roleRepository) AddPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	rolePermissions := make([]model.RolePermission, 0, len(permissionIDs))
	for _, permID := range permissionIDs {
		rolePermissions = append(rolePermissions, model.RolePermission{
			RoleID:       roleID,
			PermissionID: permID,
		})
	}
	return r.db.WithContext(ctx).Create(&rolePermissions).Error
}

// RemovePermissions 移除角色的权限
func (r *roleRepository) RemovePermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	return r.db.WithContext(ctx).Where("role_id = ? AND permission_id IN ?", roleID, permissionIDs).Delete(&model.RolePermission{}).Error
}

// GetPermissions 获取角色的权限列表
func (r *roleRepository) GetPermissions(ctx context.Context, roleID string) ([]model.Permission, error) {
	var permissions []model.Permission
	err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&permissions).Error
	return permissions, err
}

// AddUsers 为角色添加用户
func (r *roleRepository) AddUsers(ctx context.Context, roleID string, userIDs []string) error {
	// 先获取角色信息以获取 app_id
	role, err := r.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New("role not found")
	}

	userRoles := make([]model.UserRole, 0, len(userIDs))
	for _, userID := range userIDs {
		userRoles = append(userRoles, model.UserRole{
			RoleID: roleID,
			UserID: userID,
			AppID:  role.AppID, // 设置 app_id
		})
	}
	return r.db.WithContext(ctx).Create(&userRoles).Error
}

// RemoveUsers 移除角色的用户
func (r *roleRepository) RemoveUsers(ctx context.Context, roleID string, userIDs []string) error {
	return r.db.WithContext(ctx).Where("role_id = ? AND user_id IN ?", roleID, userIDs).Delete(&model.UserRole{}).Error
}

// GetUsers 获取角色的用户列表
func (r *roleRepository) GetUsers(ctx context.Context, roleID string) ([]model.User, error) {
	var users []model.User
	err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON user_roles.user_id = users.id").
		Where("user_roles.role_id = ?", roleID).
		Find(&users).Error
	return users, err
}

// GetUserRoles 获取用户在指定应用下的角色列表
func (r *roleRepository) GetUserRoles(ctx context.Context, userID, appID string) ([]model.Role, error) {
	var roles []model.Role
	query := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID)

	// 只有在提供了appID时才添加应用过滤条件
	if appID != "" {
		query = query.Where("roles.app_id = ?", appID)
	}

	err := query.Find(&roles).Error
	return roles, err
}
