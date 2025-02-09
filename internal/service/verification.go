package service

import (
	"context"

	"lauth/internal/model"
	"lauth/internal/plugin/types"
	"lauth/internal/repository"
)

// VerificationService 验证服务接口
type VerificationService interface {
	// CreateSession 创建验证会话（兼容旧接口）
	CreateSession(ctx context.Context, appID string, userID string, action string, verificationContext map[string]interface{}) (*model.VerificationSession, error)

	// CreateSessionWithIdentifier 使用标识符创建验证会话
	CreateSessionWithIdentifier(ctx context.Context, appID string, identifier string, identifierType string, action string, verificationContext map[string]interface{}) (*model.VerificationSession, error)

	// GetSession 获取验证会话（兼容旧接口）
	GetSession(ctx context.Context, appID string, userID string) (*model.VerificationSession, error)

	// GetSessionByIdentifier 通过标识符获取验证会话
	GetSessionByIdentifier(ctx context.Context, appID string, identifier string, identifierType string) (*model.VerificationSession, error)

	// GetSessionByID 通过会话ID获取验证会话
	GetSessionByID(ctx context.Context, sessionID string) (*model.VerificationSession, error)

	// GetRequiredPlugins 获取指定操作需要的插件
	GetRequiredPlugins(ctx context.Context, appID string, action string, verificationContext map[string]interface{}, userID string) ([]model.PluginRequirement, error)

	// ValidatePluginStatus 验证插件状态
	ValidatePluginStatus(ctx context.Context, appID string, userID string, action string) (*VerificationStatus, error)

	// ValidatePluginStatusBySession 通过会话验证插件状态
	ValidatePluginStatusBySession(ctx context.Context, sessionID string) (*VerificationStatus, error)

	// UpdatePluginStatus 更新插件状态
	UpdatePluginStatus(ctx context.Context, appID string, userID string, action string, pluginName string, status string) error

	// UpdatePluginStatusBySession 通过会话更新插件状态
	UpdatePluginStatusBySession(ctx context.Context, sessionID string, pluginName string, status string) error

	// ClearVerification 清理验证状态和会话
	ClearVerification(ctx context.Context, appID string, userID string, action string) error

	// UpdateSessionUserID 更新会话的用户ID
	UpdateSessionUserID(ctx context.Context, sessionID string, userID string) error
}

// verificationService 验证服务实现
type verificationService struct {
	sessionService *verificationSessionService
	pluginService  *verificationPluginService
	statusService  *verificationStatusService
}

// NewVerificationService 创建验证服务实例
func NewVerificationService(
	pluginManager types.Manager,
	pluginStatusRepo repository.PluginStatusRepository,
	sessionRepo repository.VerificationSessionRepository,
) VerificationService {
	// 创建子服务实例
	sessionService := newVerificationSessionService(pluginManager, pluginStatusRepo, sessionRepo)
	pluginService := newVerificationPluginService(pluginManager, pluginStatusRepo, sessionRepo)
	statusService := newVerificationStatusService(pluginManager, pluginStatusRepo, sessionRepo)

	return &verificationService{
		sessionService: sessionService,
		pluginService:  pluginService,
		statusService:  statusService,
	}
}

// CreateSession 创建验证会话（兼容旧接口）
func (s *verificationService) CreateSession(ctx context.Context, appID string, userID string, action string, verificationContext map[string]interface{}) (*model.VerificationSession, error) {
	return s.sessionService.CreateSession(ctx, appID, userID, action, verificationContext)
}

// CreateSessionWithIdentifier 使用标识符创建验证会话
func (s *verificationService) CreateSessionWithIdentifier(ctx context.Context, appID string, identifier string, identifierType string, action string, verificationContext map[string]interface{}) (*model.VerificationSession, error) {
	return s.sessionService.CreateSessionWithIdentifier(ctx, appID, identifier, identifierType, action, verificationContext)
}

// GetSession 获取验证会话
func (s *verificationService) GetSession(ctx context.Context, appID string, userID string) (*model.VerificationSession, error) {
	return s.sessionService.GetSession(ctx, appID, userID)
}

// GetSessionByIdentifier 通过标识符获取验证会话
func (s *verificationService) GetSessionByIdentifier(ctx context.Context, appID string, identifier string, identifierType string) (*model.VerificationSession, error) {
	return s.sessionService.GetSessionByIdentifier(ctx, appID, identifier, identifierType)
}

// GetSessionByID 通过会话ID获取验证会话
func (s *verificationService) GetSessionByID(ctx context.Context, sessionID string) (*model.VerificationSession, error) {
	return s.sessionService.GetSessionByID(ctx, sessionID)
}

// GetRequiredPlugins 获取指定操作需要的插件
func (s *verificationService) GetRequiredPlugins(ctx context.Context, appID string, action string, verificationContext map[string]interface{}, userID string) ([]model.PluginRequirement, error) {
	return s.pluginService.GetRequiredPlugins(ctx, appID, action, verificationContext, userID)
}

// ValidatePluginStatus 验证插件状态
func (s *verificationService) ValidatePluginStatus(ctx context.Context, appID string, userID string, action string) (*VerificationStatus, error) {
	return s.pluginService.ValidatePluginStatus(ctx, appID, userID, action)
}

// ValidatePluginStatusBySession 通过会话验证插件状态
func (s *verificationService) ValidatePluginStatusBySession(ctx context.Context, sessionID string) (*VerificationStatus, error) {
	return s.pluginService.ValidatePluginStatusBySession(ctx, sessionID)
}

// UpdatePluginStatus 更新插件状态
func (s *verificationService) UpdatePluginStatus(ctx context.Context, appID string, userID string, action string, pluginName string, status string) error {
	return s.statusService.UpdatePluginStatus(ctx, appID, userID, action, pluginName, status)
}

// UpdatePluginStatusBySession 通过会话更新插件状态
func (s *verificationService) UpdatePluginStatusBySession(ctx context.Context, sessionID string, pluginName string, status string) error {
	return s.statusService.UpdatePluginStatusBySession(ctx, sessionID, pluginName, status)
}

// ClearVerification 清理验证状态和会话
func (s *verificationService) ClearVerification(ctx context.Context, appID string, userID string, action string) error {
	return s.statusService.ClearVerification(ctx, appID, userID, action)
}

// UpdateSessionUserID 更新会话的用户ID
func (s *verificationService) UpdateSessionUserID(ctx context.Context, sessionID string, userID string) error {
	return s.sessionService.UpdateSessionUserID(ctx, sessionID, userID)
}
