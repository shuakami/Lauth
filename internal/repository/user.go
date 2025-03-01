package repository

import (
	"context"
	"errors"

	"lauth/internal/model"

	"gorm.io/gorm"
)

// UserRepository 用户仓储接口
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, appID, username string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, appID string, offset, limit int) ([]model.User, int64, error)
	Count(ctx context.Context) (int64, error)
}

// userRepository 用户仓储实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID 通过ID获取用户
func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername 通过用户名获取用户
func (r *userRepository) GetByUsername(ctx context.Context, appID, username string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("app_id = ? AND username = ?", appID, username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Update 更新用户
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	if user.Password != "" {
		// 如果包含密码更新,先获取现有用户
		existingUser := &model.User{}
		if err := r.db.WithContext(ctx).Where("id = ?", user.ID).First(existingUser).Error; err != nil {
			return err
		}
		// 只更新密码字段,并使用Save以触发钩子
		existingUser.Password = user.Password
		return r.db.WithContext(ctx).Save(existingUser).Error
	}
	// 更新除密码外的其他字段
	return r.db.WithContext(ctx).Model(user).Select("nickname", "email", "phone", "status").Updates(user).Error
}

// Delete 删除用户
func (r *userRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, "id = ?", id).Error
}

// List 获取用户列表
func (r *userRepository) List(ctx context.Context, appID string, offset, limit int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("app_id = ?", appID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).Where("app_id = ?", appID).Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Count 获取用户总数
func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error
	return count, err
}
