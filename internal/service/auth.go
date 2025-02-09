package service

import (
	"context"
	"errors"

	"lauth/internal/model"
	"lauth/internal/repository"
)

var (
	// ErrInvalidCredentials 无效的凭证
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrPluginRequired 需要完成插件验证
	ErrPluginRequired = errors.New("plugin verification required")
)

// AuthService 认证服务接口
type AuthService interface {
	// Register 用户注册
	Register(ctx context.Context, appID string, req *model.CreateUserRequest) (*model.ExtendedLoginResponse, error)

	// Login 用户登录
	Login(ctx context.Context, appID string, req *model.LoginRequest) (*model.ExtendedLoginResponse, error)

	// ContinueLogin 继续登录（通过会话ID）
	ContinueLogin(ctx context.Context, sessionID string) (*model.ExtendedLoginResponse, error)

	// RefreshToken 刷新访问令牌
	RefreshToken(ctx context.Context, refreshToken string) (*model.ExtendedLoginResponse, error)

	// Logout 用户登出
	Logout(ctx context.Context, accessToken string) error

	// ValidateTokenAndGetUser 验证Token并获取用户信息（快速接口）
	ValidateTokenAndGetUser(ctx context.Context, token string) (*model.TokenUserInfo, error)

	// ValidateTokenAndRuleWithUser 组合验证令牌和规则并返回用户信息
	ValidateTokenAndRuleWithUser(ctx context.Context, token string, data map[string]interface{}) (*ValidateTokenAndRuleResponse, error)
}

// authService 认证服务实现
type authService struct {
	accountService    *authAccountService
	tokenService      *authTokenService
	validationService *authValidationService
}

// NewAuthService 创建认证服务实例
func NewAuthService(
	userRepo repository.UserRepository,
	appRepo repository.AppRepository,
	tokenService TokenService,
	ruleService RuleService,
	verificationSvc VerificationService,
	profileSvc ProfileService,
	locationSvc LoginLocationService,
) AuthService {
	// 创建子服务实例
	accountService := newAuthAccountService(userRepo, appRepo, tokenService, verificationSvc, profileSvc, locationSvc)
	tokenSvc := newAuthTokenService(userRepo, tokenService)
	validationService := newAuthValidationService(userRepo, tokenService, ruleService)

	return &authService{
		accountService:    accountService,
		tokenService:      tokenSvc,
		validationService: validationService,
	}
}

// Register 用户注册
func (s *authService) Register(ctx context.Context, appID string, req *model.CreateUserRequest) (*model.ExtendedLoginResponse, error) {
	return s.accountService.Register(ctx, appID, req)
}

// Login 用户登录
func (s *authService) Login(ctx context.Context, appID string, req *model.LoginRequest) (*model.ExtendedLoginResponse, error) {
	return s.accountService.Login(ctx, appID, req)
}

// ContinueLogin 继续登录（通过会话ID）
func (s *authService) ContinueLogin(ctx context.Context, sessionID string) (*model.ExtendedLoginResponse, error) {
	return s.accountService.ContinueLogin(ctx, sessionID)
}

// RefreshToken 刷新访问令牌
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*model.ExtendedLoginResponse, error) {
	return s.tokenService.RefreshToken(ctx, refreshToken)
}

// Logout 用户登出
func (s *authService) Logout(ctx context.Context, accessToken string) error {
	return s.tokenService.Logout(ctx, accessToken)
}

// ValidateTokenAndGetUser 验证Token并获取用户信息（快速接口）
func (s *authService) ValidateTokenAndGetUser(ctx context.Context, token string) (*model.TokenUserInfo, error) {
	return s.validationService.ValidateTokenAndGetUser(ctx, token)
}

// ValidateTokenAndRuleWithUser 组合验证令牌和规则并返回用户信息
func (s *authService) ValidateTokenAndRuleWithUser(ctx context.Context, token string, data map[string]interface{}) (*ValidateTokenAndRuleResponse, error) {
	return s.validationService.ValidateTokenAndRuleWithUser(ctx, token, data)
}
