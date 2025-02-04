package service

import (
	"context"
	"fmt"
	"time"

	"lauth/internal/model"
	"lauth/internal/plugin/types"
	"lauth/internal/repository"
)

// verificationStatusService 状态管理服务
type verificationStatusService struct {
	pluginManager    types.Manager
	pluginStatusRepo repository.PluginStatusRepository
	sessionRepo      repository.VerificationSessionRepository
}

// newVerificationStatusService 创建状态管理服务实例
func newVerificationStatusService(
	pluginManager types.Manager,
	pluginStatusRepo repository.PluginStatusRepository,
	sessionRepo repository.VerificationSessionRepository,
) *verificationStatusService {
	return &verificationStatusService{
		pluginManager:    pluginManager,
		pluginStatusRepo: pluginStatusRepo,
		sessionRepo:      sessionRepo,
	}
}

// UpdatePluginStatus 更新插件状态
func (s *verificationStatusService) UpdatePluginStatus(ctx context.Context, appID string, userID string, action string, pluginName string, status string) error {
	// 创建或更新状态记录
	pluginStatus := &model.PluginStatus{
		AppID:  appID,
		UserID: &userID,
		Action: action,
		Plugin: pluginName,
		Status: status,
	}

	// 保存状态
	if err := s.pluginStatusRepo.SaveStatus(ctx, pluginStatus); err != nil {
		return fmt.Errorf("failed to save plugin status: %v", err)
	}

	// 如果状态是已完成，更新插件的验证记录
	if status == model.PluginStatusCompleted {
		plugin, exists := s.pluginManager.GetPlugin(appID, pluginName)
		if exists {
			// 这里可以添加验证上下文信息
			context := map[string]interface{}{
				"verified_at": time.Now(),
			}
			if verifiable, ok := plugin.(types.Verifiable); ok {
				if _, err := verifiable.NeedsVerification(ctx, userID, action, context); err != nil {
					return fmt.Errorf("failed to update verification record: %v", err)
				}
			}
		}
	}

	return nil
}

// UpdatePluginStatusBySession 通过会话更新插件状态
func (s *verificationStatusService) UpdatePluginStatusBySession(ctx context.Context, sessionID string, pluginName string, status string) error {
	// 获取会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %v", err)
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// 如果会话已过期
	if session.ExpiredAt.Before(time.Now()) {
		return fmt.Errorf("session expired")
	}

	// 创建或更新状态记录
	pluginStatus := &model.PluginStatus{
		AppID:  session.AppID,
		Action: session.Action,
		Plugin: pluginName,
		Status: status,
	}

	// 根据会话类型设置标识信息
	if session.UserID != nil {
		pluginStatus.UserID = session.UserID
	} else {
		pluginStatus.Identifier = session.Identifier
		pluginStatus.IdentifierType = session.IdentifierType
	}

	// 保存状态
	if err := s.pluginStatusRepo.SaveStatus(ctx, pluginStatus); err != nil {
		return fmt.Errorf("failed to save plugin status: %v", err)
	}

	// 如果状态是已完成，调用插件的成功回调
	if status == model.PluginStatusCompleted {
		plugin, exists := s.pluginManager.GetPlugin(session.AppID, pluginName)
		if exists && session.UserID != nil {
			if verifiable, ok := plugin.(types.Verifiable); ok {
				if err := verifiable.OnVerificationSuccess(ctx, *session.UserID, session.Action, session.Context); err != nil {
					return fmt.Errorf("failed to handle verification success: %v", err)
				}
			}
		}
	}

	return nil
}

// ClearVerification 清理验证状态和会话
func (s *verificationStatusService) ClearVerification(ctx context.Context, appID string, userID string, action string) error {
	// 删除会话
	session, err := s.sessionRepo.GetActiveSession(ctx, appID, userID)
	if err != nil {
		return fmt.Errorf("failed to get session: %v", err)
	}
	if session != nil {
		if err := s.sessionRepo.Delete(ctx, session.ID); err != nil {
			return fmt.Errorf("failed to delete session: %v", err)
		}
	}

	// 获取并删除验证状态
	statuses, err := s.pluginStatusRepo.ListStatus(ctx, appID, userID, action)
	if err != nil {
		return fmt.Errorf("failed to list statuses: %v", err)
	}
	for _, status := range statuses {
		if err := s.pluginStatusRepo.DeleteStatus(ctx, appID, userID, action, status.Plugin); err != nil {
			return fmt.Errorf("failed to delete status: %v", err)
		}
	}

	return nil
}
