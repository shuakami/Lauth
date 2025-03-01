package service

import (
	"context"
	"fmt"
	"log"
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

	// 根据action筛选需要的插件
	var requirements []model.PluginRequirement
	for _, config := range configs {
		// 检查插件是否启用
		if !config.Enabled {
			continue
		}

		// 获取插件实例
		plugin, exists := s.pluginManager.GetPlugin(appID, config.Name)
		if !exists {
			continue
		}

		// 获取插件元数据
		metadata := plugin.GetMetadata()

		// 检查插件是否适用于当前action
		actionMatch := false
		for _, a := range metadata.Actions {
			if a == action {
				actionMatch = true
				break
			}
		}
		if !actionMatch {
			continue
		}

		// 如果是必需插件，直接添加到列表，不检查 NeedsVerification
		if metadata.Required {
			log.Printf("[Plugin] 必需插件 %s 添加到验证列表", config.Name)
			requirements = append(requirements, model.PluginRequirement{
				Name:     config.Name,
				Required: true,
				Stage:    metadata.Stage,
				Status:   model.PluginStatusPending,
			})
			continue
		}

		// 对于可选插件，检查是否需要验证
		needsVerify, err := s.checkPluginVerification(ctx, plugin, userID, action, verificationContext)
		if err != nil {
			return nil, err
		}

		if !needsVerify {
			continue
		}

		log.Printf("[Plugin] 可选插件 %s 需要验证，添加到验证列表", config.Name)
		requirements = append(requirements, model.PluginRequirement{
			Name:     config.Name,
			Required: false,
			Stage:    metadata.Stage,
			Status:   model.PluginStatusPending,
		})
	}

	if len(requirements) > 0 {
		log.Printf("[Plugin] 需要验证的插件列表: %+v", requirements)
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

// pluginValidator 插件验证器
type pluginValidator struct {
	appID           string
	userID          *string
	action          string
	plugin          types.Plugin
	verificationMap map[string]*model.PluginStatus
}

// newPluginValidator 创建插件验证器
func newPluginValidator(appID string, userID *string, action string, plugin types.Plugin, verificationMap map[string]*model.PluginStatus) *pluginValidator {
	return &pluginValidator{
		appID:           appID,
		userID:          userID,
		action:          action,
		plugin:          plugin,
		verificationMap: verificationMap,
	}
}

// validate 验证插件状态
func (v *pluginValidator) validate(ctx context.Context) (bool, error) {
	// 获取插件实例
	if v.plugin == nil {
		return false, nil
	}

	// 验证当前验证是否有效
	valid, err := v.validatePluginVerification(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to validate plugin verification: %v", err)
	}

	return valid, nil
}

// validatePluginVerification 验证插件验证状态
func (v *pluginValidator) validatePluginVerification(ctx context.Context) (bool, error) {
	verifiable, ok := v.plugin.(types.Verifiable)
	if !ok {
		return true, nil
	}

	// 获取当前验证记录
	verification := v.verificationMap[v.plugin.GetMetadata().Name]

	// 如果没有验证记录,说明未验证
	if verification == nil {
		return false, nil
	}

	var uid string
	if v.userID != nil {
		uid = *v.userID
	}

	// 验证记录是否有效
	valid, err := verifiable.ValidateVerification(ctx, uid, v.action, verification.ID)
	if err != nil {
		return false, fmt.Errorf("failed to validate verification: %v", err)
	}
	return valid, nil
}

// validateRequiredPlugins 验证必需插件
func (s *verificationPluginService) validateRequiredPlugins(
	ctx context.Context,
	session *model.VerificationSession,
	requirements []model.PluginRequirement,
	statusMap map[string]string,
	verificationMap map[string]*model.PluginStatus,
) (bool, *model.PluginRequirement, error) {
	allCompleted := true
	var nextPlugin *model.PluginRequirement

	for i := range requirements {
		if !requirements[i].Required {
			continue
		}

		completed, next, err := s.validateSinglePlugin(ctx, session, &requirements[i], statusMap, verificationMap)
		if err != nil {
			return false, nil, err
		}

		if !completed {
			allCompleted = false
			if nextPlugin == nil {
				nextPlugin = next
			}
		}
	}

	return allCompleted, nextPlugin, nil
}

// validateOptionalPlugins 验证可选插件
func (s *verificationPluginService) validateOptionalPlugins(
	ctx context.Context,
	session *model.VerificationSession,
	requirements []model.PluginRequirement,
	statusMap map[string]string,
	verificationMap map[string]*model.PluginStatus,
) (*model.PluginRequirement, error) {
	var nextPlugin *model.PluginRequirement

	for i := range requirements {
		if requirements[i].Required {
			continue
		}

		completed, next, err := s.validateSinglePlugin(ctx, session, &requirements[i], statusMap, verificationMap)
		if err != nil {
			return nil, err
		}

		if !completed && nextPlugin == nil {
			nextPlugin = next
		}
	}

	return nextPlugin, nil
}

// validateSinglePlugin 验证单个插件
func (s *verificationPluginService) validateSinglePlugin(
	ctx context.Context,
	session *model.VerificationSession,
	requirement *model.PluginRequirement,
	statusMap map[string]string,
	verificationMap map[string]*model.PluginStatus,
) (bool, *model.PluginRequirement, error) {
	// 检查插件状态
	status, exists := statusMap[requirement.Name]
	if exists && status == model.PluginStatusCompleted {
		requirement.Status = model.PluginStatusCompleted
		log.Printf("[Plugin] 插件 %s 验证通过", requirement.Name)
		return true, nil, nil
	}

	// 获取插件实例
	plugin, exists := s.pluginManager.GetPlugin(session.AppID, requirement.Name)
	if !exists {
		return true, nil, nil
	}

	// 验证当前验证是否有效
	validator := newPluginValidator(session.AppID, session.UserID, session.Action, plugin, verificationMap)
	valid, err := validator.validate(ctx)
	if err != nil {
		return false, nil, err
	}

	if valid {
		requirement.Status = model.PluginStatusCompleted
		log.Printf("[Plugin] 插件 %s 验证通过", requirement.Name)
		return true, nil, nil
	}

	log.Printf("[Plugin] 插件 %s 需要验证", requirement.Name)
	return false, requirement, nil
}

// ValidatePluginStatusBySession 通过会话验证插件状态
func (s *verificationPluginService) ValidatePluginStatusBySession(ctx context.Context, sessionID string) (*VerificationStatus, error) {
	// 获取会话
	session, err := s.getSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 获取所有插件的验证状态
	statuses, err := s.getStatusesBySession(ctx, session)
	if err != nil {
		return nil, err
	}

	// 构建状态映射
	statusMap, verificationMap := s.buildStatusMaps(statuses)

	// 获取需要验证的插件列表
	var userID string
	if session.UserID != nil {
		userID = *session.UserID
	} else if session.Identifier != "" {
		userID = session.Identifier
	}
	requirements, err := s.GetRequiredPlugins(ctx, session.AppID, session.Action, session.Context, userID)
	if err != nil {
		return nil, err
	}

	log.Printf("Required plugins: %+v", requirements)

	// 先验证必需插件
	allCompleted, nextPlugin, err := s.validateRequiredPlugins(ctx, session, requirements, statusMap, verificationMap)
	if err != nil {
		return nil, err
	}

	// 如果所有必需插件都已完成，再验证可选插件
	if allCompleted && nextPlugin == nil {
		nextPlugin, err = s.validateOptionalPlugins(ctx, session, requirements, statusMap, verificationMap)
		if err != nil {
			return nil, err
		}
	}

	// 创建验证状态响应
	var status *VerificationStatus
	if allCompleted && nextPlugin == nil {
		status = s.createCompletedStatus()
	} else {
		status = s.createPendingStatus(nextPlugin)
	}
	status.Plugins = requirements

	log.Printf("Verification status: completed=%v, status=%s, nextPlugin=%+v",
		status.Completed, status.Status, status.NextPlugin)

	return status, nil
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
