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

	// 如果有userID,获取已有的验证状态
	var statusMap map[string]*model.PluginStatus
	if userID != "" {
		statuses, err := s.pluginStatusRepo.ListStatus(ctx, appID, userID, action)
		if err != nil {
			return nil, fmt.Errorf("failed to get plugin statuses: %v", err)
		}
		statusMap = make(map[string]*model.PluginStatus)
		for _, status := range statuses {
			statusMap[status.Plugin] = status
		}
	}

	// 根据action筛选需要的插件
	var requirements []model.PluginRequirement
	for _, config := range configs {
		// 检查插件是否启用且适用于当前action
		if !config.Enabled {
			continue
		}

		actionMatch := false
		for _, a := range config.Actions {
			if a == action {
				actionMatch = true
				break
			}
		}
		if !actionMatch {
			continue
		}

		// 检查插件状态是否已完成
		if status, exists := statusMap[config.Name]; exists && status.Status == model.PluginStatusCompleted {
			fmt.Printf("插件 %s 已完成验证,跳过\n", config.Name)
			continue
		}

		// 获取插件实例
		plugin, exists := s.pluginManager.GetPlugin(appID, config.Name)
		if !exists {
			continue
		}

		// 检查插件是否需要验证
		verifiable, ok := plugin.(types.Verifiable)
		if !ok {
			continue
		}

		needsVerify, err := verifiable.NeedsVerification(ctx, userID, action, verificationContext)
		if err != nil {
			return nil, fmt.Errorf("failed to check if plugin needs verification: %v", err)
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

// ValidatePluginStatus 验证插件状态
func (s *verificationPluginService) ValidatePluginStatus(ctx context.Context, appID string, userID string, action string) (*VerificationStatus, error) {
	// 获取当前会话
	session, err := s.sessionRepo.GetActiveSession(ctx, appID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %v", err)
	}
	if session == nil {
		return nil, fmt.Errorf("no active session found")
	}

	// 获取需要的插件
	requirements, err := s.GetRequiredPlugins(ctx, appID, action, session.Context, userID)
	if err != nil {
		return nil, err
	}

	// 如果没有需要的插件，直接返回完成
	if len(requirements) == 0 {
		return &VerificationStatus{
			Completed: true,
			Status:    model.PluginStatusCompleted,
			UpdatedAt: time.Now(),
		}, nil
	}

	// 获取所有插件的状态
	statuses, err := s.pluginStatusRepo.ListStatus(ctx, appID, userID, action)
	if err != nil {
		return nil, err
	}

	// 构建状态映射
	statusMap := make(map[string]string)
	verificationMap := make(map[string]*model.PluginStatus)
	for _, status := range statuses {
		statusMap[status.Plugin] = status.Status
		verificationMap[status.Plugin] = status
	}

	// 检查每个必需插件的状态
	for _, req := range requirements {
		status, exists := statusMap[req.Name]
		if !exists || status != model.PluginStatusCompleted {
			return &VerificationStatus{
				Completed:  false,
				Status:     model.PluginStatusPending,
				NextPlugin: &req,
				UpdatedAt:  time.Now(),
			}, nil
		}

		// 获取插件实例
		plugin, exists := s.pluginManager.GetPlugin(appID, req.Name)
		if !exists {
			continue
		}

		// 验证当前验证是否有效
		verifiable, ok := plugin.(types.Verifiable)
		if !ok {
			continue
		}

		verification := verificationMap[req.Name]
		if verification != nil {
			valid, err := verifiable.ValidateVerification(ctx, userID, action, verification.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to validate verification: %v", err)
			}
			if !valid {
				return &VerificationStatus{
					Completed:  false,
					Status:     model.PluginStatusPending,
					NextPlugin: &req,
					UpdatedAt:  time.Now(),
				}, nil
			}
		}
	}

	return &VerificationStatus{
		Completed: true,
		Status:    model.PluginStatusCompleted,
		UpdatedAt: time.Now(),
	}, nil
}

// ValidatePluginStatusBySession 通过会话验证插件状态
func (s *verificationPluginService) ValidatePluginStatusBySession(ctx context.Context, sessionID string) (*VerificationStatus, error) {
	// 获取会话
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %v", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	// 如果会话已过期
	if session.ExpiredAt.Before(time.Now()) {
		return nil, fmt.Errorf("session expired")
	}

	// 获取需要的插件
	requirements, err := s.GetRequiredPlugins(ctx, session.AppID, session.Action, session.Context, "")
	if err != nil {
		return nil, err
	}

	// 如果没有需要的插件，直接返回完成
	if len(requirements) == 0 {
		return &VerificationStatus{
			Completed: true,
			Status:    model.PluginStatusCompleted,
			UpdatedAt: time.Now(),
		}, nil
	}

	// 获取所有插件的状态
	var statuses []*model.PluginStatus
	if session.UserID != nil {
		statuses, err = s.pluginStatusRepo.ListStatus(ctx, session.AppID, *session.UserID, session.Action)
	} else {
		statuses, err = s.pluginStatusRepo.ListStatusByIdentifier(ctx, session.AppID, session.Identifier, session.IdentifierType, session.Action)
	}
	if err != nil {
		return nil, err
	}

	// 构建状态映射
	statusMap := make(map[string]string)
	verificationMap := make(map[string]*model.PluginStatus)
	for _, status := range statuses {
		statusMap[status.Plugin] = status.Status
		verificationMap[status.Plugin] = status
	}

	// 检查每个必需插件的状态
	for _, req := range requirements {
		status, exists := statusMap[req.Name]
		if !exists || status != model.PluginStatusCompleted {
			return &VerificationStatus{
				Completed:  false,
				Status:     model.PluginStatusPending,
				NextPlugin: &req,
				UpdatedAt:  time.Now(),
			}, nil
		}

		// 获取插件实例
		plugin, exists := s.pluginManager.GetPlugin(session.AppID, req.Name)
		if !exists {
			continue
		}

		// 验证当前验证是否有效
		verifiable, ok := plugin.(types.Verifiable)
		if !ok {
			continue
		}

		verification := verificationMap[req.Name]
		if verification != nil {
			var userID string
			if session.UserID != nil {
				userID = *session.UserID
			}
			valid, err := verifiable.ValidateVerification(ctx, userID, session.Action, verification.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to validate verification: %v", err)
			}
			if !valid {
				return &VerificationStatus{
					Completed:  false,
					Status:     model.PluginStatusPending,
					NextPlugin: &req,
					UpdatedAt:  time.Now(),
				}, nil
			}
		}
	}

	return &VerificationStatus{
		Completed: true,
		Status:    model.PluginStatusCompleted,
		UpdatedAt: time.Now(),
	}, nil
}
