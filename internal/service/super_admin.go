package service

import (
	"context"
	"errors"
	"lauth/internal/model"
	"lauth/internal/repository"
	"log"
)

var (
	// ErrUserNotExists 用户不存在
	ErrUserNotExists = errors.New("用户不存在")
	// ErrSuperAdminExists 超级管理员已存在
	ErrSuperAdminExists = errors.New("该用户已经是超级管理员")
	// ErrSuperAdminNotExists 超级管理员不存在
	ErrSuperAdminNotExists = errors.New("该用户不是超级管理员")
	// ErrLastSuperAdmin 不能删除最后一个超级管理员
	ErrLastSuperAdmin = errors.New("不能删除最后一个超级管理员")
)

// SuperAdminService 超级管理员服务接口
type SuperAdminService interface {
	// 添加超级管理员
	AddSuperAdmin(ctx context.Context, userID string) error
	// 删除超级管理员
	RemoveSuperAdmin(ctx context.Context, userID string) error
	// 获取所有超级管理员
	ListSuperAdmins(ctx context.Context) ([]*model.SuperAdmin, error)
	// 判断用户是否是超级管理员
	IsSuperAdmin(ctx context.Context, userID string) (bool, error)
	// 检查是否需要初始化超级管理员
	CheckAndInitSuperAdmin(ctx context.Context) (string, string, bool, error)
}

// superAdminService 超级管理员服务实现
type superAdminService struct {
	superAdminRepo repository.SuperAdminRepository
	userRepo       repository.UserRepository
	appRepo        repository.AppRepository
}

// NewSuperAdminService 创建超级管理员服务实例
func NewSuperAdminService(
	superAdminRepo repository.SuperAdminRepository,
	userRepo repository.UserRepository,
	appRepo repository.AppRepository,
) SuperAdminService {
	return &superAdminService{
		superAdminRepo: superAdminRepo,
		userRepo:       userRepo,
		appRepo:        appRepo,
	}
}

// AddSuperAdmin 添加超级管理员
func (s *superAdminService) AddSuperAdmin(ctx context.Context, userID string) error {
	// 检查用户是否存在
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotExists
	}

	// 检查是否已经是超级管理员
	isSuperAdmin, err := s.superAdminRepo.IsSuperAdmin(ctx, userID)
	if err != nil {
		return err
	}
	if isSuperAdmin {
		return ErrSuperAdminExists
	}

	// 创建超级管理员
	superAdmin := &model.SuperAdmin{
		UserID: userID,
	}
	return s.superAdminRepo.CreateSuperAdmin(ctx, superAdmin)
}

// RemoveSuperAdmin 删除超级管理员
func (s *superAdminService) RemoveSuperAdmin(ctx context.Context, userID string) error {
	// 检查是否是超级管理员
	isSuperAdmin, err := s.superAdminRepo.IsSuperAdmin(ctx, userID)
	if err != nil {
		return err
	}
	if !isSuperAdmin {
		return ErrSuperAdminNotExists
	}

	// 检查是否是最后一个超级管理员
	admins, err := s.superAdminRepo.ListSuperAdmins(ctx)
	if err != nil {
		return err
	}
	if len(admins) <= 1 {
		return ErrLastSuperAdmin
	}

	// 删除超级管理员
	return s.superAdminRepo.DeleteSuperAdmin(ctx, userID)
}

// ListSuperAdmins 获取所有超级管理员
func (s *superAdminService) ListSuperAdmins(ctx context.Context) ([]*model.SuperAdmin, error) {
	return s.superAdminRepo.ListSuperAdmins(ctx)
}

// IsSuperAdmin 判断用户是否是超级管理员
func (s *superAdminService) IsSuperAdmin(ctx context.Context, userID string) (bool, error) {
	log.Printf("[DEBUG] 正在检查用户是否为超级管理员: %s", userID)
	isSuperAdmin, err := s.superAdminRepo.IsSuperAdmin(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] 查询超级管理员状态失败: %v", err)
		return false, err
	}
	log.Printf("[DEBUG] 用户 %s 是否为超级管理员: %v", userID, isSuperAdmin)
	return isSuperAdmin, nil
}

// CheckAndInitSuperAdmin 检查是否需要初始化超级管理员
func (s *superAdminService) CheckAndInitSuperAdmin(ctx context.Context) (string, string, bool, error) {
	// 检查是否已有超级管理员
	admins, err := s.superAdminRepo.ListSuperAdmins(ctx)
	if err != nil {
		return "", "", false, err
	}

	// 如果已有超级管理员，直接返回
	if len(admins) > 0 {
		// 确保已有的admin用户密码不会因为重启而被重置
		return "", "", false, nil
	}

	// 创建默认应用（如果不存在）
	apps, count, err := s.appRepo.List(ctx, 0, 1)
	if err != nil {
		return "", "", false, err
	}

	var appID string
	if count == 0 {
		// 创建默认应用
		defaultApp := &model.App{
			Name:        "系统默认应用",
			Description: "初始化自动创建的系统默认应用",
			Status:      model.AppStatusEnabled,
		}
		if err := s.appRepo.Create(ctx, defaultApp); err != nil {
			return "", "", false, err
		}
		appID = defaultApp.ID
	} else {
		appID = apps[0].ID
	}

	// 使用专用的管理员账号，而不是查找已有用户
	adminUsername := "admin"
	adminPassword := "Admin@123" // 仅用于新建管理员

	// 先检查超级管理员表中是否已有对应用户
	allUsers, total, err := s.userRepo.List(ctx, appID, 0, 100)
	if err != nil {
		log.Printf("[ERROR] 获取所有用户列表时出错: %v", err)
		return "", "", false, err
	}

	log.Printf("[INFO] 找到 %d 个用户", total)

	// 查找管理员用户
	var adminUser *model.User
	for _, user := range allUsers {
		if user.Username == adminUsername {
			userCopy := user // 创建一个拷贝
			adminUser = &userCopy
			break
		}
	}

	// 如果已找到管理员用户
	if adminUser != nil && adminUser.ID != "" {
		log.Printf("[INFO] 用户 %s 已存在，ID: %s", adminUsername, adminUser.ID)

		// 创建超级管理员关联（如果之前不存在）
		isSuperAdmin, err := s.superAdminRepo.IsSuperAdmin(ctx, adminUser.ID)
		if err != nil {
			return "", "", false, err
		}

		if !isSuperAdmin {
			// 创建超级管理员关联
			superAdmin := &model.SuperAdmin{
				UserID: adminUser.ID,
			}
			err = s.superAdminRepo.CreateSuperAdmin(ctx, superAdmin)
			if err != nil {
				return "", "", false, err
			}
			log.Printf("[INFO] 为已存在用户 %s 创建了超级管理员关联", adminUsername)
		}

		// 返回空密码，表示使用现有密码
		return adminUsername, "", true, nil
	}

	// 创建专用的超级管理员用户
	newAdminUser := &model.User{
		AppID:        appID,
		Username:     adminUsername,
		Password:     adminPassword,
		Status:       model.UserStatusEnabled,
		Name:         "系统管理员",
		Email:        "admin@example.com",
		IsFirstLogin: true, // 标记为首次登录
	}
	if err := s.userRepo.Create(ctx, newAdminUser); err != nil {
		return "", "", false, err
	}

	// 创建超级管理员
	superAdmin := &model.SuperAdmin{
		UserID: newAdminUser.ID,
	}
	err = s.superAdminRepo.CreateSuperAdmin(ctx, superAdmin)
	if err != nil {
		return "", "", false, err
	}

	// 返回超级管理员初始凭据
	return adminUsername, adminPassword, true, nil
}
