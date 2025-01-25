package service

import (
	"context"
	"errors"

	"lauth/internal/model"
	"lauth/internal/repository"
)

var (
	ErrPermissionNotFound   = errors.New("权限不存在")
	ErrPermissionNameExists = errors.New("权限名称已存在")
	ErrNoPermission         = errors.New("没有权限执行此操作")
)

// PermissionService 权限服务接口
type PermissionService interface {
	// 基础CRUD
	Create(ctx context.Context, appID string, permission *model.Permission) error
	GetByID(ctx context.Context, id string) (*model.Permission, error)
	Update(ctx context.Context, permission *model.Permission) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, appID string, offset, limit int) ([]model.Permission, int64, error)

	// 权限验证
	ValidatePermission(ctx context.Context, userID, appID string, resourceType model.ResourceType, action model.ActionType) error
	HasPermission(ctx context.Context, userID, appID string, resourceType model.ResourceType, action model.ActionType) (bool, error)

	// 资源权限管理
	GetResourcePermissions(ctx context.Context, appID string, resourceType model.ResourceType) ([]model.Permission, error)
	GetUserPermissions(ctx context.Context, userID, appID string) ([]model.Permission, error)
}

// permissionService 权限服务实现
type permissionService struct {
	permissionRepo repository.PermissionRepository
	roleRepo       repository.RoleRepository
}

// NewPermissionService 创建权限服务实例
func NewPermissionService(permissionRepo repository.PermissionRepository, roleRepo repository.RoleRepository) PermissionService {
	return &permissionService{
		permissionRepo: permissionRepo,
		roleRepo:       roleRepo,
	}
}

// Create 创建权限
func (s *permissionService) Create(ctx context.Context, appID string, permission *model.Permission) error {
	// 检查权限名是否已存在
	existing, err := s.permissionRepo.GetByName(ctx, appID, permission.Name)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrPermissionNameExists
	}

	permission.AppID = appID
	return s.permissionRepo.Create(ctx, permission)
}

// GetByID 获取权限
func (s *permissionService) GetByID(ctx context.Context, id string) (*model.Permission, error) {
	permission, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if permission == nil {
		return nil, ErrPermissionNotFound
	}
	return permission, nil
}

// Update 更新权限
func (s *permissionService) Update(ctx context.Context, permission *model.Permission) error {
	existing, err := s.permissionRepo.GetByID(ctx, permission.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrPermissionNotFound
	}

	// 检查新名称是否与其他权限冲突
	if permission.Name != existing.Name {
		nameExists, err := s.permissionRepo.GetByName(ctx, permission.AppID, permission.Name)
		if err != nil {
			return err
		}
		if nameExists != nil && nameExists.ID != permission.ID {
			return ErrPermissionNameExists
		}
	}

	return s.permissionRepo.Update(ctx, permission)
}

// Delete 删除权限
func (s *permissionService) Delete(ctx context.Context, id string) error {
	existing, err := s.permissionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrPermissionNotFound
	}

	return s.permissionRepo.Delete(ctx, id)
}

// List 获取权限列表
func (s *permissionService) List(ctx context.Context, appID string, offset, limit int) ([]model.Permission, int64, error) {
	return s.permissionRepo.List(ctx, appID, offset, limit)
}

// ValidatePermission 验证用户是否有指定权限
func (s *permissionService) ValidatePermission(ctx context.Context, userID, appID string, resourceType model.ResourceType, action model.ActionType) error {
	hasPermission, err := s.HasPermission(ctx, userID, appID, resourceType, action)
	if err != nil {
		return err
	}
	if !hasPermission {
		return ErrNoPermission
	}
	return nil
}

// HasPermission 检查用户是否有指定权限
func (s *permissionService) HasPermission(ctx context.Context, userID, appID string, resourceType model.ResourceType, action model.ActionType) (bool, error) {
	// 获取用户的所有权限
	permissions, err := s.GetUserPermissions(ctx, userID, appID)
	if err != nil {
		return false, err
	}

	// 检查是否有匹配的权限
	for _, p := range permissions {
		if p.ResourceType == resourceType && p.Action == action {
			return true, nil
		}
	}

	return false, nil
}

// GetResourcePermissions 获取资源类型的所有权限
func (s *permissionService) GetResourcePermissions(ctx context.Context, appID string, resourceType model.ResourceType) ([]model.Permission, error) {
	return s.permissionRepo.ListByResourceType(ctx, appID, resourceType)
}

// GetUserPermissions 获取用户的所有权限
func (s *permissionService) GetUserPermissions(ctx context.Context, userID, appID string) ([]model.Permission, error) {
	var allPermissions []model.Permission
	permMap := make(map[string]bool)

	// 获取用户的所有角色
	roles, err := s.roleRepo.GetUserRoles(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	// 获取每个角色的权限
	for _, role := range roles {
		permissions, err := s.roleRepo.GetPermissions(ctx, role.ID)
		if err != nil {
			return nil, err
		}

		// 使用map去重
		for _, p := range permissions {
			if !permMap[p.ID] {
				permMap[p.ID] = true
				allPermissions = append(allPermissions, p)
			}
		}
	}

	return allPermissions, nil
}
