package service

import (
	"context"
	"errors"
	"log"

	"lauth/internal/model"
	"lauth/internal/repository"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUserDisabled    = errors.New("user is disabled")
)

// UserService 用户服务接口
type UserService interface {
	CreateUser(ctx context.Context, appID string, req *model.CreateUserRequest) (*model.User, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
	UpdateUser(ctx context.Context, id string, req *model.UpdateUserRequest) (*model.User, error)
	UpdatePassword(ctx context.Context, id string, req *model.UpdatePasswordRequest) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, appID string, page, pageSize int) ([]model.User, int64, error)
	ValidateUser(ctx context.Context, appID string, username, password string) (*model.User, error)
}

// userService 用户服务实现
type userService struct {
	userRepo repository.UserRepository
	appRepo  repository.AppRepository
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repository.UserRepository, appRepo repository.AppRepository) UserService {
	return &userService{
		userRepo: userRepo,
		appRepo:  appRepo,
	}
}

// CreateUser 创建用户
func (s *userService) CreateUser(ctx context.Context, appID string, req *model.CreateUserRequest) (*model.User, error) {
	// 验证应用是否存在
	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrAppNotFound
	}

	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetByUsername(ctx, appID, req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}

	user := &model.User{
		AppID:    appID,
		Username: req.Username,
		Password: req.Password,
		Nickname: req.Nickname,
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   model.UserStatusEnabled,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUser 获取用户
func (s *userService) GetUser(ctx context.Context, id string) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// UpdateUser 更新用户
func (s *userService) UpdateUser(ctx context.Context, id string, req *model.UpdateUserRequest) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	user.Nickname = req.Nickname
	user.Email = req.Email
	user.Phone = req.Phone
	user.Status = req.Status

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdatePassword 更新密码
func (s *userService) UpdatePassword(ctx context.Context, id string, req *model.UpdatePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		log.Printf("获取用户失败: %v", err)
		return err
	}
	if user == nil {
		log.Printf("用户不存在: %s", id)
		return ErrUserNotFound
	}

	log.Printf("找到用户: id=%s, username=%s", user.ID, user.Username)

	// 验证旧密码
	log.Printf("开始验证旧密码")
	if !user.ValidatePassword(req.OldPassword) {
		log.Printf("旧密码验证失败: id=%s, username=%s", user.ID, user.Username)
		return ErrInvalidPassword
	}
	log.Printf("旧密码验证成功")

	// 更新密码(让model层处理哈希)
	user.Password = req.NewPassword
	if err := s.userRepo.Update(ctx, user); err != nil {
		log.Printf("更新密码失败: %v", err)
		return err
	}
	log.Printf("密码更新成功")

	return nil
}

// DeleteUser 删除用户
func (s *userService) DeleteUser(ctx context.Context, id string) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	return s.userRepo.Delete(ctx, id)
}

// ListUsers 获取用户列表
func (s *userService) ListUsers(ctx context.Context, appID string, page, pageSize int) ([]model.User, int64, error) {
	offset := (page - 1) * pageSize
	return s.userRepo.List(ctx, appID, offset, pageSize)
}

// ValidateUser 验证用户凭证
func (s *userService) ValidateUser(ctx context.Context, appID string, username, password string) (*model.User, error) {
	user, err := s.userRepo.GetByUsername(ctx, appID, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if !user.ValidatePassword(password) {
		return nil, ErrInvalidPassword
	}

	if user.Status == model.UserStatusDisabled {
		return nil, ErrUserDisabled
	}

	return user, nil
}
