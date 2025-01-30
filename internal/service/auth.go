package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"

	"lauth/internal/model"
	"lauth/internal/repository"
	"lauth/pkg/engine"
)

var (
	// ErrInvalidCredentials 无效的凭证
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrPluginRequired 需要完成插件验证
	ErrPluginRequired = errors.New("plugin verification required")
)

// ValidateTokenAndRuleResponse 组合验证响应
type ValidateTokenAndRuleResponse struct {
	User         *model.User    `json:"user"`
	RuleResult   *engine.Result `json:"rule_result"`
	ValidateTime time.Time      `json:"validate_time"`
	Status       bool           `json:"status"`
}

// AuthService 认证服务接口
type AuthService interface {
	// Login 用户登录
	Login(ctx context.Context, appID string, req *model.LoginRequest) (*model.ExtendedLoginResponse, error)

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
	userRepo        repository.UserRepository
	tokenService    TokenService
	ruleService     RuleService
	verificationSvc VerificationService
}

// NewAuthService 创建认证服务实例
func NewAuthService(
	userRepo repository.UserRepository,
	tokenService TokenService,
	ruleService RuleService,
	verificationSvc VerificationService,
) AuthService {
	return &authService{
		userRepo:        userRepo,
		tokenService:    tokenService,
		ruleService:     ruleService,
		verificationSvc: verificationSvc,
	}
}

// Login 用户登录
func (s *authService) Login(ctx context.Context, appID string, req *model.LoginRequest) (*model.ExtendedLoginResponse, error) {
	// 验证用户名密码
	user, err := s.userRepo.GetByUsername(ctx, appID, req.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 检查用户状态
	if user.Status == model.UserStatusDisabled {
		return nil, ErrUserDisabled
	}

	// 创建验证会话
	if _, err := s.verificationSvc.CreateSession(ctx, appID, user.ID, "login"); err != nil {
		return nil, fmt.Errorf("failed to create verification session: %v", err)
	}

	// 获取需要的插件
	plugins, err := s.verificationSvc.GetRequiredPlugins(ctx, appID, "login")
	if err != nil {
		return nil, err
	}
	log.Printf("Required plugins: %+v", plugins)

	// 检查验证状态
	verifyStatus, err := s.verificationSvc.ValidatePluginStatus(ctx, appID, user.ID, "login")
	if err != nil {
		return nil, err
	}
	log.Printf("Verification status: completed=%v, status=%s, nextPlugin=%+v",
		verifyStatus.Completed, verifyStatus.Status, verifyStatus.NextPlugin)

	// 如果有需要验证的插件但未完成验证，直接返回pending状态
	if len(plugins) > 0 && !verifyStatus.Completed {
		log.Printf("Plugin verification required: plugins=%d, completed=%v",
			len(plugins), verifyStatus.Completed)
		return &model.ExtendedLoginResponse{
			User: model.UserResponse{
				ID:        user.ID,
				AppID:     user.AppID,
				Username:  user.Username,
				Nickname:  user.Nickname,
				Email:     user.Email,
				Phone:     user.Phone,
				Status:    user.Status,
				CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
			AuthStatus: verifyStatus.Status,
			Plugins:    plugins,
			NextPlugin: verifyStatus.NextPlugin,
		}, ErrPluginRequired
	}

	// 如果验证完成，则生成token
	tokenPair, err := s.tokenService.GenerateTokenPair(ctx, user, "read")
	if err != nil {
		return nil, err
	}

	return &model.ExtendedLoginResponse{
		User: model.UserResponse{
			ID:        user.ID,
			AppID:     user.AppID,
			Username:  user.Username,
			Nickname:  user.Nickname,
			Email:     user.Email,
			Phone:     user.Phone,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(tokenPair.AccessTokenExpireIn.Seconds()),
		AuthStatus:   model.PluginStatusCompleted,
		Plugins:      plugins,
		NextPlugin:   verifyStatus.NextPlugin,
	}, nil
}

// RefreshToken 刷新访问令牌
func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*model.ExtendedLoginResponse, error) {
	// 使用TokenService刷新令牌
	tokenPair, err := s.tokenService.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// 获取用户信息
	claims, err := s.tokenService.ValidateToken(ctx, tokenPair.RefreshToken, model.RefreshToken)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// 检查用户状态
	if user.Status == model.UserStatusDisabled {
		return nil, ErrUserDisabled
	}

	// 构造响应
	return &model.ExtendedLoginResponse{
		User: model.UserResponse{
			ID:        user.ID,
			AppID:     user.AppID,
			Username:  user.Username,
			Nickname:  user.Nickname,
			Email:     user.Email,
			Phone:     user.Phone,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(tokenPair.AccessTokenExpireIn.Seconds()),
		AuthStatus:   model.PluginStatusCompleted,
	}, nil
}

// Logout 用户登出
func (s *authService) Logout(ctx context.Context, accessToken string) error {
	// 验证并吊销访问令牌
	claims, err := s.tokenService.ValidateToken(ctx, accessToken, model.AccessToken)
	if err != nil {
		return err
	}

	// 吊销访问令牌
	if err := s.tokenService.RevokeToken(ctx, accessToken, model.AccessToken); err != nil {
		return err
	}

	// 获取并吊销关联的刷新令牌
	refreshKey := fmt.Sprintf("refresh_token:%s", claims.UserID)
	refreshToken, err := s.tokenService.(*tokenService).redis.Get(ctx, refreshKey)
	if err == nil && refreshToken != "" {
		_ = s.tokenService.RevokeToken(ctx, refreshToken, model.RefreshToken)
	}

	return nil
}

// ValidateTokenAndGetUser 验证Token并获取用户信息（快速接口）
func (s *authService) ValidateTokenAndGetUser(ctx context.Context, token string) (*model.TokenUserInfo, error) {
	// 验证Token
	claims, err := s.tokenService.ValidateToken(ctx, token, model.AccessToken)
	if err != nil {
		return nil, err
	}

	// 构造快速响应
	return &model.TokenUserInfo{
		UserID:   claims.UserID,
		AppID:    claims.AppID,
		Username: claims.Username,
	}, nil
}

// ValidateTokenAndRuleWithUser 组合验证令牌和规则并返回用户信息
func (s *authService) ValidateTokenAndRuleWithUser(ctx context.Context, token string, data map[string]interface{}) (*ValidateTokenAndRuleResponse, error) {
	// 先验证令牌并获取用户信息
	userInfo, err := s.ValidateTokenAndGetUser(ctx, token)
	if err != nil {
		return nil, err
	}

	// 将 token 中的用户信息添加到验证数据中
	data["token_user_id"] = userInfo.UserID
	data["token_app_id"] = userInfo.AppID
	data["token_username"] = userInfo.Username

	// 如果请求中包含 user_id，验证是否与 token 用户匹配
	if requestUserID, ok := data["user_id"].(string); ok {
		if requestUserID != userInfo.UserID {
			return nil, errors.New("user_id mismatch with token")
		}
	}

	// 验证规则
	ruleResult, ruleErr := s.ruleService.ValidateRule(ctx, userInfo.AppID, data)

	// 获取完整的用户信息
	user, err := s.userRepo.GetByID(ctx, userInfo.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// 构造响应
	response := &ValidateTokenAndRuleResponse{
		User:         user,
		RuleResult:   ruleResult,
		ValidateTime: time.Now(),
		Status:       ruleErr == nil,
	}

	return response, ruleErr
}
