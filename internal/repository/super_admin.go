package repository

import (
	"context"
	"lauth/internal/model"
	"log"

	"gorm.io/gorm"
)

// SuperAdminRepository 超级管理员仓库接口
type SuperAdminRepository interface {
	// 创建超级管理员
	CreateSuperAdmin(ctx context.Context, superAdmin *model.SuperAdmin) error
	// 删除超级管理员
	DeleteSuperAdmin(ctx context.Context, userID string) error
	// 检查用户是否是超级管理员
	IsSuperAdmin(ctx context.Context, userID string) (bool, error)
	// 获取所有超级管理员
	ListSuperAdmins(ctx context.Context) ([]*model.SuperAdmin, error)
	// 获取超级管理员详情（包括用户信息）
	GetSuperAdmin(ctx context.Context, userID string) (*model.SuperAdmin, error)
}

// superAdminRepository 超级管理员仓库实现
type superAdminRepository struct {
	db *gorm.DB
}

// NewSuperAdminRepository 创建超级管理员仓库实例
func NewSuperAdminRepository(db *gorm.DB) SuperAdminRepository {
	return &superAdminRepository{db: db}
}

// CreateSuperAdmin 创建超级管理员
func (r *superAdminRepository) CreateSuperAdmin(ctx context.Context, superAdmin *model.SuperAdmin) error {
	return r.db.WithContext(ctx).Create(superAdmin).Error
}

// DeleteSuperAdmin 删除超级管理员
func (r *superAdminRepository) DeleteSuperAdmin(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.SuperAdmin{}).Error
}

// IsSuperAdmin 判断用户是否是超级管理员
func (r *superAdminRepository) IsSuperAdmin(ctx context.Context, userID string) (bool, error) {
	log.Printf("[DEBUG] 正在数据库中检查超级管理员: %s", userID)
	var count int64
	err := r.db.WithContext(ctx).Model(&model.SuperAdmin{}).Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		log.Printf("[ERROR] 查询超级管理员数据库失败: %v", err)
		return false, err
	}
	isSuperAdmin := count > 0
	log.Printf("[DEBUG] 数据库查询结果: 用户 %s 是否为超级管理员: %v (记录数: %d)", userID, isSuperAdmin, count)
	return isSuperAdmin, nil
}

// ListSuperAdmins 获取所有超级管理员
func (r *superAdminRepository) ListSuperAdmins(ctx context.Context) ([]*model.SuperAdmin, error) {
	var superAdmins []*model.SuperAdmin
	err := r.db.WithContext(ctx).Preload("User").Find(&superAdmins).Error
	return superAdmins, err
}

// GetSuperAdmin 获取超级管理员详情（包括用户信息）
func (r *superAdminRepository) GetSuperAdmin(ctx context.Context, userID string) (*model.SuperAdmin, error) {
	var superAdmin model.SuperAdmin
	err := r.db.WithContext(ctx).Preload("User").Where("user_id = ?", userID).First(&superAdmin).Error
	if err != nil {
		return nil, err
	}
	return &superAdmin, nil
}
