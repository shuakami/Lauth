package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lauth/internal/model"
)

// verificationSessionRepository 验证会话仓储实现
type verificationSessionRepository struct {
	db *gorm.DB
}

// NewVerificationSessionRepository 创建验证会话仓储实例
func NewVerificationSessionRepository(db *gorm.DB) VerificationSessionRepository {
	return &verificationSessionRepository{
		db: db,
	}
}

// Create 创建会话
func (r *verificationSessionRepository) Create(ctx context.Context, session *model.VerificationSession) error {
	if session.ID == "" {
		session.ID = uuid.New().String()
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	session.UpdatedAt = time.Now()
	if session.ExpiredAt.IsZero() {
		session.ExpiredAt = time.Now().Add(30 * time.Minute) // 默认30分钟过期
	}

	return r.db.WithContext(ctx).Create(session).Error
}

// GetByID 获取会话
func (r *verificationSessionRepository) GetByID(ctx context.Context, id string) (*model.VerificationSession, error) {
	var session model.VerificationSession
	err := r.db.WithContext(ctx).Where("id = ? AND expired_at > ?", id, time.Now()).First(&session).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &session, err
}

// GetActiveSession 获取用户当前活动的会话
func (r *verificationSessionRepository) GetActiveSession(ctx context.Context, appID, userID string) (*model.VerificationSession, error) {
	var session model.VerificationSession
	err := r.db.WithContext(ctx).
		Where("app_id = ? AND user_id = ? AND expired_at > ?", appID, userID, time.Now()).
		Order("created_at DESC").
		First(&session).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &session, err
}

// Update 更新会话
func (r *verificationSessionRepository) Update(ctx context.Context, session *model.VerificationSession) error {
	session.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(session).Error
}

// Delete 删除会话
func (r *verificationSessionRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.VerificationSession{}, "id = ?", id).Error
}

// DeleteExpired 删除过期会话
func (r *verificationSessionRepository) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expired_at <= ?", time.Now()).
		Delete(&model.VerificationSession{}).Error
}
