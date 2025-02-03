package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"

	"lauth/internal/model"
	"lauth/internal/repository"
)

// authAccountService 账号服务
type authAccountService struct {
	userRepo        repository.UserRepository
	appRepo         repository.AppRepository
	tokenService    TokenService
	verificationSvc VerificationService
	profileSvc      ProfileService
	locationSvc     LoginLocationService
}

// newAuthAccountService 创建账号服务实例
func newAuthAccountService(
	userRepo repository.UserRepository,
	appRepo repository.AppRepository,
	tokenService TokenService,
	verificationSvc VerificationService,
	profileSvc ProfileService,
	locationSvc LoginLocationService,
) *authAccountService {
	return &authAccountService{
		userRepo:        userRepo,
		appRepo:         appRepo,
		tokenService:    tokenService,
		verificationSvc: verificationSvc,
		profileSvc:      profileSvc,
		locationSvc:     locationSvc,
	}
}

// Register 用户注册
func (s *authAccountService) Register(ctx context.Context, appID string, req *model.CreateUserRequest) (*model.User, error) {
	// 验证应用是否存在
	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrAppNotFound
	}

	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetByUsername(ctx, appID, req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}

	user := &model.User{
		AppID:    appID,
		Username: req.Username,
		Password: req.Password,
		Nickname: req.Nickname,
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   model.UserStatusEnabled,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 如果请求中包含Profile信息,创建用户档案
	if req.Profile != nil {
		if _, err := s.profileSvc.CreateProfile(ctx, user.ID, appID, req.Profile); err != nil {
			// 如果创建档案失败,记录错误但不影响用户创建
			log.Printf("创建用户档案失败: %v", err)
		}
	}

	return user, nil
}

// Login 用户登录
func (s *authAccountService) Login(ctx context.Context, appID string, req *model.LoginRequest) (*model.ExtendedLoginResponse, error) {
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

	// 构建验证上下文
	verificationContext := map[string]interface{}{
		"ip":          req.ClientIP,
		"device_id":   req.DeviceID,
		"device_type": req.DeviceType,
		"user_agent":  req.UserAgent,
	}

	// 创建验证会话
	session, err := s.verificationSvc.CreateSession(ctx, appID, user.ID, "login", verificationContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create verification session: %v", err)
	}

	// 获取需要的插件
	plugins, err := s.verificationSvc.GetRequiredPlugins(ctx, appID, "login", verificationContext, user.ID)
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
			SessionID:  session.ID,
		}, ErrPluginRequired
	}

	// 如果验证完成，则清理验证状态和会话
	if err := s.verificationSvc.ClearVerification(ctx, appID, user.ID, "login"); err != nil {
		log.Printf("Failed to clear verification: %v", err)
	}

	// 如果验证完成，则生成token
	tokenPair, err := s.tokenService.GenerateTokenPair(ctx, user, "read")
	if err != nil {
		return nil, err
	}

	// 记录登录位置
	if req.ClientIP != "" {
		if err := s.locationSvc.RecordLoginLocation(ctx, appID, user.ID, req.ClientIP); err != nil {
			log.Printf("Failed to record login location: %v", err)
			// 不影响登录流程，只记录错误
		}
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

// ContinueLogin 继续登录（通过会话ID）
func (s *authAccountService) ContinueLogin(ctx context.Context, sessionID string) (*model.ExtendedLoginResponse, error) {
	// 获取会话信息
	session, err := s.verificationSvc.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %v", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	// 验证会话是否过期
	if session.ExpiredAt.Before(time.Now()) {
		return nil, fmt.Errorf("session expired")
	}

	// 获取用户信息
	var user *model.User
	if session.UserID != nil {
		user, err = s.userRepo.GetByID(ctx, *session.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %v", err)
		}
		if user == nil {
			return nil, fmt.Errorf("user not found")
		}
	} else {
		return nil, fmt.Errorf("invalid session: no user ID")
	}

	// 检查用户状态
	if user.Status == model.UserStatusDisabled {
		return nil, ErrUserDisabled
	}

	// 获取需要的插件
	plugins, err := s.verificationSvc.GetRequiredPlugins(ctx, session.AppID, "login", session.Context, *session.UserID)
	if err != nil {
		return nil, err
	}

	// 检查验证状态
	verifyStatus, err := s.verificationSvc.ValidatePluginStatusBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 如果验证未完成，返回当前状态
	if len(plugins) > 0 && !verifyStatus.Completed {
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
			SessionID:  sessionID,
		}, ErrPluginRequired
	}

	// 如果验证完成，生成token
	tokenPair, err := s.tokenService.GenerateTokenPair(ctx, user, "read")
	if err != nil {
		return nil, err
	}

	// 记录登录位置（从会话上下文中获取IP）
	if ip, ok := session.Context["ip"].(string); ok && ip != "" {
		if err := s.locationSvc.RecordLoginLocation(ctx, session.AppID, user.ID, ip); err != nil {
			log.Printf("Failed to record login location: %v", err)
			// 不影响登录流程，只记录错误
		}
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
