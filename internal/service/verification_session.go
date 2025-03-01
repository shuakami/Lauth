package service

import (
	"context"
	"fmt"
	"time"

	"lauth/internal/model"
	"lauth/internal/plugin/types"
	"lauth/internal/repository"
)

// VerificationStatus 验证状态
type VerificationStatus struct {
	Completed  bool                      `json:"completed"` // 是否完成
	Status     string                    `json:"status"`    // 当前状态
	NextPlugin *model.PluginRequirement  `json:"next_plugin,omitempty"`
	UpdatedAt  time.Time                 `json:"updated_at"` // 更新时间
	Plugins    []model.PluginRequirement `json:"plugins,omitempty"`
}

// verificationSessionService 会话管理服务
type verificationSessionService struct {
	pluginManager    types.Manager
	pluginStatusRepo repository.PluginStatusRepository
	sessionRepo      repository.VerificationSessionRepository
}

// newVerificationSessionService 创建会话管理服务实例
func newVerificationSessionService(
	pluginManager types.Manager,
	pluginStatusRepo repository.PluginStatusRepository,
	sessionRepo repository.VerificationSessionRepository,
) *verificationSessionService {
	return &verificationSessionService{
		pluginManager:    pluginManager,
		pluginStatusRepo: pluginStatusRepo,
		sessionRepo:      sessionRepo,
	}
}

// CreateSession 创建验证会话（兼容旧接口）
func (s *verificationSessionService) CreateSession(ctx context.Context, appID string, userID string, action string, verificationContext map[string]interface{}) (*model.VerificationSession, error) {
	// 先检查最近的验证状态
	statuses, err := s.pluginStatusRepo.ListStatus(ctx, appID, userID, action)
	if err != nil {
		return nil, fmt.Errorf("failed to check verification status: %v", err)
	}

	// 检查是否有最近的成功验证
	var recentCompletedStatus *model.PluginStatus
	for _, status := range statuses {
		if status.Status == model.PluginStatusCompleted {
			if recentCompletedStatus == nil || status.UpdatedAt.After(recentCompletedStatus.UpdatedAt) {
				recentCompletedStatus = status
			}
		}
	}

	// 如果有最近的成功验证,且在30分钟内,则继承该状态
	if recentCompletedStatus != nil && time.Since(recentCompletedStatus.UpdatedAt) < 30*time.Minute {
		// 创建新的验证状态记录
		newStatus := &model.PluginStatus{
			AppID:  appID,
			UserID: &userID,
			Action: action,
			Plugin: recentCompletedStatus.Plugin,
			Status: model.PluginStatusCompleted,
		}
		if err := s.pluginStatusRepo.SaveStatus(ctx, newStatus); err != nil {
			return nil, fmt.Errorf("failed to save inherited status: %v", err)
		}
	}

	// 先删除已有的会话
	existingSession, err := s.sessionRepo.GetActiveSession(ctx, appID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing session: %v", err)
	}
	if existingSession != nil {
		if err := s.sessionRepo.Delete(ctx, existingSession.ID); err != nil {
			return nil, fmt.Errorf("failed to delete existing session: %v", err)
		}
	}

	// 创建新会话
	session := &model.VerificationSession{
		AppID:   appID,
		UserID:  &userID,
		Action:  action,
		Status:  model.PluginStatusPending,
		Context: verificationContext,
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	return session, nil
}

// CreateSessionWithIdentifier 使用标识符创建验证会话
func (s *verificationSessionService) CreateSessionWithIdentifier(ctx context.Context, appID string, identifier string, identifierType string, action string, verificationContext map[string]interface{}) (*model.VerificationSession, error) {
	// 先删除已有的会话
	existingSession, err := s.sessionRepo.GetActiveSessionByIdentifier(ctx, appID, identifier, identifierType)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing session: %v", err)
	}
	if existingSession != nil {
		if err := s.sessionRepo.Delete(ctx, existingSession.ID); err != nil {
			return nil, fmt.Errorf("failed to delete existing session: %v", err)
		}
	}

	// 创建新会话
	session := &model.VerificationSession{
		AppID:          appID,
		Identifier:     identifier,
		IdentifierType: identifierType,
		Action:         action,
		Status:         model.PluginStatusPending,
		Context:        verificationContext,
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	return session, nil
}

// GetSession 获取验证会话
func (s *verificationSessionService) GetSession(ctx context.Context, appID string, userID string) (*model.VerificationSession, error) {
	return s.sessionRepo.GetActiveSession(ctx, appID, userID)
}

// GetSessionByIdentifier 通过标识符获取验证会话
func (s *verificationSessionService) GetSessionByIdentifier(ctx context.Context, appID string, identifier string, identifierType string) (*model.VerificationSession, error) {
	return s.sessionRepo.GetActiveSessionByIdentifier(ctx, appID, identifier, identifierType)
}

// GetSessionByID 通过会话ID获取验证会话
func (s *verificationSessionService) GetSessionByID(ctx context.Context, sessionID string) (*model.VerificationSession, error) {
	return s.sessionRepo.GetByID(ctx, sessionID)
}

// UpdateSessionUserID 更新会话的用户ID
func (s *verificationSessionService) UpdateSessionUserID(ctx context.Context, sessionID string, userID string) error {
	// 获取会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %v", err)
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// 更新会话的userID
	session.UserID = &userID

	// 保存更新后的会话
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %v", err)
	}

	return nil
}
