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
	Completed  bool                     `json:"completed"`   // 是否完成
	Status     string                   `json:"status"`      // 当前状态
	NextPlugin *model.PluginRequirement `json:"next_plugin"` // 下一个需要的插件
	UpdatedAt  time.Time                `json:"updated_at"`  // 更新时间
}

// VerificationService 验证服务接口
type VerificationService interface {
	// CreateSession 创建验证会话
	CreateSession(ctx context.Context, appID string, userID string, action string) (*model.VerificationSession, error)

	// GetSession 获取验证会话
	GetSession(ctx context.Context, appID string, userID string) (*model.VerificationSession, error)

	// GetRequiredPlugins 获取指定操作需要的插件
	GetRequiredPlugins(ctx context.Context, appID string, action string) ([]model.PluginRequirement, error)

	// ValidatePluginStatus 验证插件状态
	ValidatePluginStatus(ctx context.Context, appID string, userID string, action string) (*VerificationStatus, error)

	// UpdatePluginStatus 更新插件状态
	UpdatePluginStatus(ctx context.Context, appID string, userID string, action string, pluginName string, status string) error
}

// verificationService 验证服务实现
type verificationService struct {
	pluginManager    types.Manager
	pluginStatusRepo repository.PluginStatusRepository
	sessionRepo      repository.VerificationSessionRepository
}

// NewVerificationService 创建验证服务实例
func NewVerificationService(
	pluginManager types.Manager,
	pluginStatusRepo repository.PluginStatusRepository,
	sessionRepo repository.VerificationSessionRepository,
) VerificationService {
	return &verificationService{
		pluginManager:    pluginManager,
		pluginStatusRepo: pluginStatusRepo,
		sessionRepo:      sessionRepo,
	}
}

// CreateSession 创建验证会话
func (s *verificationService) CreateSession(ctx context.Context, appID string, userID string, action string) (*model.VerificationSession, error) {
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
		AppID:  appID,
		UserID: userID,
		Action: action,
		Status: model.PluginStatusPending,
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	return session, nil
}

// GetSession 获取验证会话
func (s *verificationService) GetSession(ctx context.Context, appID string, userID string) (*model.VerificationSession, error) {
	return s.sessionRepo.GetActiveSession(ctx, appID, userID)
}

// GetRequiredPlugins 获取指定操作需要的插件
func (s *verificationService) GetRequiredPlugins(ctx context.Context, appID string, action string) ([]model.PluginRequirement, error) {
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
func (s *verificationService) ValidatePluginStatus(ctx context.Context, appID string, userID string, action string) (*VerificationStatus, error) {
	// 获取需要的插件
	requirements, err := s.GetRequiredPlugins(ctx, appID, action)
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
	for _, status := range statuses {
		statusMap[status.Plugin] = status.Status
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
	}

	return &VerificationStatus{
		Completed: true,
		Status:    model.PluginStatusCompleted,
		UpdatedAt: time.Now(),
	}, nil
}

// UpdatePluginStatus 更新插件状态
func (s *verificationService) UpdatePluginStatus(ctx context.Context, appID string, userID string, action string, pluginName string, status string) error {
	return s.pluginStatusRepo.SaveStatus(ctx, &model.PluginStatus{
		AppID:     appID,
		UserID:    userID,
		Action:    action,
		Plugin:    pluginName,
		Status:    status,
		UpdatedAt: time.Now(),
	})
}
