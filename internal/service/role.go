package service

import (
	"context"
	"errors"

	"lauth/internal/model"
	"lauth/internal/repository"
)

var (
	ErrRoleNotFound     = errors.New("角色不存在")
	ErrRoleNameExists   = errors.New("角色名称已存在")
	ErrRoleSystemLocked = errors.New("系统角色不可修改")
)

// RoleService 角色服务接口
type RoleService interface {
	// 基础CRUD
	Create(ctx context.Context, appID string, role *model.Role) error
	GetByID(ctx context.Context, id string) (*model.Role, error)
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, appID string, offset, limit int) ([]model.Role, int64, error)

	// 角色权限管理
	AddPermissions(ctx context.Context, roleID string, permissionIDs []string) error
	RemovePermissions(ctx context.Context, roleID string, permissionIDs []string) error
	GetPermissions(ctx context.Context, roleID string) ([]model.Permission, error)

	// 用户角色管理
	AddUsers(ctx context.Context, roleID string, userIDs []string) error
	RemoveUsers(ctx context.Context, roleID string, userIDs []string) error
	GetUsers(ctx context.Context, roleID string) ([]model.User, error)
}

// roleService 角色服务实现
type roleService struct {
	roleRepo       repository.RoleRepository
	permissionRepo repository.PermissionRepository
}

// NewRoleService 创建角色服务实例
func NewRoleService(roleRepo repository.RoleRepository, permissionRepo repository.PermissionRepository) RoleService {
	return &roleService{
		roleRepo:       roleRepo,
		permissionRepo: permissionRepo,
	}
}

// Create 创建角色
func (s *roleService) Create(ctx context.Context, appID string, role *model.Role) error {
	// 检查角色名是否已存在
	existing, err := s.roleRepo.GetByName(ctx, appID, role.Name)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrRoleNameExists
	}

	role.AppID = appID
	return s.roleRepo.Create(ctx, role)
}

// GetByID 获取角色
func (s *roleService) GetByID(ctx context.Context, id string) (*model.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}
	return role, nil
}

// Update 更新角色
func (s *roleService) Update(ctx context.Context, role *model.Role) error {
	existing, err := s.roleRepo.GetByID(ctx, role.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrRoleNotFound
	}

	// 系统角色不可修改
	if existing.IsSystem {
		return ErrRoleSystemLocked
	}

	// 检查新名称是否与其他角色冲突
	if role.Name != existing.Name {
		nameExists, err := s.roleRepo.GetByName(ctx, role.AppID, role.Name)
		if err != nil {
			return err
		}
		if nameExists != nil && nameExists.ID != role.ID {
			return ErrRoleNameExists
		}
	}

	return s.roleRepo.Update(ctx, role)
}

// Delete 删除角色
func (s *roleService) Delete(ctx context.Context, id string) error {
	existing, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrRoleNotFound
	}

	// 系统角色不可删除
	if existing.IsSystem {
		return ErrRoleSystemLocked
	}

	return s.roleRepo.Delete(ctx, id)
}

// List 获取角色列表
func (s *roleService) List(ctx context.Context, appID string, offset, limit int) ([]model.Role, int64, error) {
	return s.roleRepo.List(ctx, appID, offset, limit)
}

// AddPermissions 为角色添加权限
func (s *roleService) AddPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	// 检查角色是否存在
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}

	// 系统角色不可修改权限
	if role.IsSystem {
		return ErrRoleSystemLocked
	}

	return s.roleRepo.AddPermissions(ctx, roleID, permissionIDs)
}

// RemovePermissions 移除角色的权限
func (s *roleService) RemovePermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	// 检查角色是否存在
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}

	// 系统角色不可修改权限
	if role.IsSystem {
		return ErrRoleSystemLocked
	}

	return s.roleRepo.RemovePermissions(ctx, roleID, permissionIDs)
}

// GetPermissions 获取角色的权限列表
func (s *roleService) GetPermissions(ctx context.Context, roleID string) ([]model.Permission, error) {
	// 检查角色是否存在
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	return s.roleRepo.GetPermissions(ctx, roleID)
}

// AddUsers 为角色添加用户
func (s *roleService) AddUsers(ctx context.Context, roleID string, userIDs []string) error {
	// 检查角色是否存在
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}

	return s.roleRepo.AddUsers(ctx, roleID, userIDs)
}

// RemoveUsers 移除角色的用户
func (s *roleService) RemoveUsers(ctx context.Context, roleID string, userIDs []string) error {
	// 检查角色是否存在
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}

	return s.roleRepo.RemoveUsers(ctx, roleID, userIDs)
}

// GetUsers 获取角色的用户列表
func (s *roleService) GetUsers(ctx context.Context, roleID string) ([]model.User, error) {
	// 检查角色是否存在
	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	return s.roleRepo.GetUsers(ctx, roleID)
}
