package repository

import (
	"context"
	"lauth/internal/model"
)

// VerificationSessionRepository 验证会话仓储接口
type VerificationSessionRepository interface {
	// Create 创建会话
	Create(ctx context.Context, session *model.VerificationSession) error

	// GetByID 获取会话
	GetByID(ctx context.Context, id string) (*model.VerificationSession, error)

	// GetActiveSession 获取用户当前活动的会话
	GetActiveSession(ctx context.Context, appID, userID string) (*model.VerificationSession, error)

	// Update 更新会话
	Update(ctx context.Context, session *model.VerificationSession) error

	// Delete 删除会话
	Delete(ctx context.Context, id string) error

	// DeleteExpired 删除过期会话
	DeleteExpired(ctx context.Context) error
}
