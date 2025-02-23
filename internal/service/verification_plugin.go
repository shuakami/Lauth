package service

import (
	"context"
	"fmt"
	"time"

	"lauth/internal/model"
	"lauth/internal/plugin/types"
	"lauth/internal/repository"
)

// verificationPluginService 插件验证服务
type verificationPluginService struct {
	pluginManager    types.Manager
	pluginStatusRepo repository.PluginStatusRepository
	sessionRepo      repository.VerificationSessionRepository
}

// newVerificationPluginService 创建插件验证服务实例
func newVerificationPluginService(
	pluginManager types.Manager,
	pluginStatusRepo repository.PluginStatusRepository,
	sessionRepo repository.VerificationSessionRepository,
) *verificationPluginService {
	return &verificationPluginService{
		pluginManager:    pluginManager,
		pluginStatusRepo: pluginStatusRepo,
		sessionRepo:      sessionRepo,
	}
}

// checkPluginStatus 检查插件状态是否已完成
func (s *verificationPluginService) checkPluginStatus(statusMap map[string]*model.PluginStatus, pluginName string) bool {
	if status, exists := statusMap[pluginName]; exists && status.Status == model.PluginStatusCompleted {
		fmt.Printf("插件 %s 已完成验证,跳过\n", pluginName)
		return true
	}
	return false
}

// checkPluginActionMatch 检查插件是否适用于当前action
func (s *verificationPluginService) checkPluginActionMatch(actions []string, targetAction string) bool {
	for _, a := range actions {
		if a == targetAction {
			return true
		}
	}
	return false
}

// getPluginStatusMap 获取插件状态映射
func (s *verificationPluginService) getPluginStatusMap(ctx context.Context, appID, userID, action string) (map[string]*model.PluginStatus, error) {
	if userID == "" {
		return make(map[string]*model.PluginStatus), nil
	}

	statuses, err := s.pluginStatusRepo.ListStatus(ctx, appID, userID, action)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin statuses: %v", err)
	}

	statusMap := make(map[string]*model.PluginStatus)
	for _, status := range statuses {
		statusMap[status.Plugin] = status
	}
	return statusMap, nil
}

// checkPluginVerification 检查插件是否需要验证
func (s *verificationPluginService) checkPluginVerification(ctx context.Context, plugin types.Plugin, userID, action string, verificationContext map[string]interface{}) (bool, error) {
	verifiable, ok := plugin.(types.Verifiable)
	if !ok {
		return false, nil
	}

	needsVerify, err := verifiable.NeedsVerification(ctx, userID, action, verificationContext)
	if err != nil {
		return false, fmt.Errorf("failed to check if plugin needs verification: %v", err)
	}

	return needsVerify, nil
}

// GetRequiredPlugins 获取指定操作需要的插件
func (s *verificationPluginService) GetRequiredPlugins(ctx context.Context, appID string, action string, verificationContext map[string]interface{}, userID string) ([]model.PluginRequirement, error) {
	// 获取App已安装的插件列表
	installedPlugins := s.pluginManager.ListPlugins(appID)
	if len(installedPlugins) == 0 {
		return nil, nil
	}

	// 获取插件配置
	configs, err := s.pluginManager.GetPluginConfigs(ctx, appID)
	if err != nil {
		return nil, err
	}

	// 获取插件状态映射
	statusMap, err := s.getPluginStatusMap(ctx, appID, userID, action)
	if err != nil {
		return nil, err
	}

	// 根据action筛选需要的插件
	var requirements []model.PluginRequirement
	for _, config := range configs {
		// 检查插件是否启用且适用于当前action
		if !config.Enabled {
			continue
		}

		if !s.checkPluginActionMatch(config.Actions, action) {
			continue
		}

		// 检查插件状态是否已完成
		if s.checkPluginStatus(statusMap, config.Name) {
			continue
		}

		// 获取插件实例
		plugin, exists := s.pluginManager.GetPlugin(appID, config.Name)
		if !exists {
			continue
		}

		// 检查插件是否需要验证
		needsVerify, err := s.checkPluginVerification(ctx, plugin, userID, action, verificationContext)
		if err != nil {
			return nil, err
		}

		// 如果插件不需要验证，跳过
		if !needsVerify {
			continue
		}

		// 添加到需求列表
		requirements = append(requirements, model.PluginRequirement{
			Name:     config.Name,
			Required: config.Required,
			Stage:    config.Stage,
			Status:   model.PluginStatusPending,
		})
	}

	return requirements, nil
}

// getActiveSession 获取活动会话
func (s *verificationPluginService) getActiveSession(ctx context.Context, appID string, userID string) (*model.VerificationSession, error) {
	session, err := s.sessionRepo.GetActiveSession(ctx, appID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %v", err)
	}
	if session == nil {
		return nil, fmt.Errorf("no active session found")
	}
	return session, nil
}

// buildStatusMaps 构建状态映射
func (s *verificationPluginService) buildStatusMaps(statuses []*model.PluginStatus) (map[string]string, map[string]*model.PluginStatus) {
	statusMap := make(map[string]string)
	verificationMap := make(map[string]*model.PluginStatus)
	for _, status := range statuses {
		statusMap[status.Plugin] = status.Status
		verificationMap[status.Plugin] = status
	}
	return statusMap, verificationMap
}

// validatePluginVerification 验证插件验证状态
func (s *verificationPluginService) validatePluginVerification(ctx context.Context, userID *string, action string, plugin types.Plugin, verification *model.PluginStatus) (bool, error) {
	verifiable, ok := plugin.(types.Verifiable)
	if !ok {
		return true, nil
	}

	if verification != nil {
		var uid string
		if userID != nil {
			uid = *userID
		}
		valid, err := verifiable.ValidateVerification(ctx, uid, action, verification.ID)
		if err != nil {
			return false, fmt.Errorf("failed to validate verification: %v", err)
		}
		return valid, nil
	}
	return true, nil
}

// getSessionByID 获取并验证会话
func (s *verificationPluginService) getSessionByID(ctx context.Context, sessionID string) (*model.VerificationSession, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %v", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	// 检查会话是否过期
	if session.ExpiredAt.Before(time.Now()) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// getStatusesBySession 根据会话获取插件状态列表
func (s *verificationPluginService) getStatusesBySession(ctx context.Context, session *model.VerificationSession) ([]*model.PluginStatus, error) {
	if session.UserID != nil {
		return s.pluginStatusRepo.ListStatus(ctx, session.AppID, *session.UserID, session.Action)
	}
	return s.pluginStatusRepo.ListStatusByIdentifier(ctx, session.AppID, session.Identifier, session.IdentifierType, session.Action)
}

// createCompletedStatus 创建已完成的验证状态
func (s *verificationPluginService) createCompletedStatus() *VerificationStatus {
	return &VerificationStatus{
		Completed: true,
		Status:    model.PluginStatusCompleted,
		UpdatedAt: time.Now(),
	}
}

// createPendingStatus 创建待处理的验证状态
func (s *verificationPluginService) createPendingStatus(nextPlugin *model.PluginRequirement) *VerificationStatus {
	return &VerificationStatus{
		Completed:  false,
		Status:     model.PluginStatusPending,
		NextPlugin: nextPlugin,
		UpdatedAt:  time.Now(),
	}
}

// ValidatePluginStatusBySession 通过会话验证插件状态
func (s *verificationPluginService) ValidatePluginStatusBySession(ctx context.Context, sessionID string) (*VerificationStatus, error) {
	// 获取并验证会话
	session, err := s.getSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 获取需要的插件
	requirements, err := s.GetRequiredPlugins(ctx, session.AppID, session.Action, session.Context, "")
	if err != nil {
		return nil, err
	}

	// 如果没有需要的插件，直接返回完成
	if len(requirements) == 0 {
		return s.createCompletedStatus(), nil
	}

	// 获取所有插件的状态
	statuses, err := s.getStatusesBySession(ctx, session)
	if err != nil {
		return nil, err
	}

	// 构建状态映射
	statusMap, verificationMap := s.buildStatusMaps(statuses)

	// 检查每个必需插件的状态
	for _, req := range requirements {
		status, exists := statusMap[req.Name]
		if !exists || status != model.PluginStatusCompleted {
			return s.createPendingStatus(&req), nil
		}

		// 获取插件实例
		plugin, exists := s.pluginManager.GetPlugin(session.AppID, req.Name)
		if !exists {
			continue
		}

		// 验证当前验证是否有效
		valid, err := s.validatePluginVerification(ctx, session.UserID, session.Action, plugin, verificationMap[req.Name])
		if err != nil {
			return nil, err
		}
		if !valid {
			return s.createPendingStatus(&req), nil
		}
	}

	return s.createCompletedStatus(), nil
}

// ValidatePluginStatus 验证插件状态
func (s *verificationPluginService) ValidatePluginStatus(ctx context.Context, appID string, userID string, action string) (*VerificationStatus, error) {
	// 获取活动会话
	session, err := s.getActiveSession(ctx, appID, userID)
	if err != nil {
		return nil, err
	}

	// 使用会话ID验证插件状态
	return s.ValidatePluginStatusBySession(ctx, session.ID)
}
