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

// verificationContext 验证上下文
type verificationContext struct {
	appID      string
	userID     string
	action     string
	clientIP   string
	deviceID   string
	deviceType string
	userAgent  string
}

// buildVerificationContext 构建验证上下文
func (s *authAccountService) buildVerificationContext(appID, userID, action string, req interface{}) *verificationContext {
	ctx := &verificationContext{
		appID:  appID,
		userID: userID,
		action: action,
	}

	// 根据请求类型设置上下文
	switch r := req.(type) {
	case *model.LoginRequest:
		ctx.clientIP = r.ClientIP
		ctx.deviceID = r.DeviceID
		ctx.deviceType = r.DeviceType
		ctx.userAgent = r.UserAgent
	case *model.CreateUserRequest:
		ctx.clientIP = r.ClientIP
		ctx.deviceID = r.DeviceID
		ctx.deviceType = r.DeviceType
		ctx.userAgent = r.UserAgent
	}

	return ctx
}

// handleVerification 处理验证流程
func (s *authAccountService) handleVerification(ctx context.Context, vCtx *verificationContext) (*model.VerificationSession, []model.PluginRequirement, *VerificationStatus, error) {
	// 构建验证上下文map
	contextMap := map[string]interface{}{
		"ip":          vCtx.clientIP,
		"device_id":   vCtx.deviceID,
		"device_type": vCtx.deviceType,
		"user_agent":  vCtx.userAgent,
	}

	// 创建验证会话
	session, err := s.verificationSvc.CreateSession(ctx, vCtx.appID, vCtx.userID, vCtx.action, contextMap)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create verification session: %v", err)
	}

	// 获取需要的插件
	plugins, err := s.verificationSvc.GetRequiredPlugins(ctx, vCtx.appID, vCtx.action, contextMap, vCtx.userID)
	if err != nil {
		return nil, nil, nil, err
	}
	log.Printf("Required plugins: %+v", plugins)

	// 检查验证状态
	verifyStatus, err := s.verificationSvc.ValidatePluginStatus(ctx, vCtx.appID, vCtx.userID, vCtx.action)
	if err != nil {
		return nil, nil, nil, err
	}
	log.Printf("Verification status: completed=%v, status=%s, nextPlugin=%+v",
		verifyStatus.Completed, verifyStatus.Status, verifyStatus.NextPlugin)

	return session, plugins, verifyStatus, nil
}

// buildUserResponse 构建用户响应
func (s *authAccountService) buildUserResponse(user *model.User, tokenPair *model.TokenPair, plugins []model.PluginRequirement, verifyStatus *VerificationStatus, sessionID string) *model.ExtendedLoginResponse {
	response := &model.ExtendedLoginResponse{
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
		Plugins:    plugins,
		NextPlugin: verifyStatus.NextPlugin,
		SessionID:  sessionID,
	}

	if tokenPair != nil {
		response.AccessToken = tokenPair.AccessToken
		response.RefreshToken = tokenPair.RefreshToken
		response.ExpiresIn = int64(tokenPair.AccessTokenExpireIn.Seconds())
		response.AuthStatus = model.PluginStatusCompleted
	} else {
		response.AuthStatus = verifyStatus.Status
	}

	return response
}

// Register 用户注册
func (s *authAccountService) Register(ctx context.Context, appID string, req *model.CreateUserRequest) (*model.ExtendedLoginResponse, error) {
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

	// 先处理验证流程
	vCtx := s.buildVerificationContext(appID, "pending", "register", req)
	session, plugins, verifyStatus, err := s.handleVerification(ctx, vCtx)
	if err != nil {
		return nil, err
	}

	// 如果需要验证，返回验证状态
	if len(plugins) > 0 && !verifyStatus.Completed {
		// 构建一个临时的用户响应
		tempUser := &model.User{
			AppID:    appID,
			Username: req.Username,
			Nickname: req.Nickname,
			Email:    req.Email,
			Phone:    req.Phone,
			Status:   model.UserStatusEnabled,
		}
		return s.buildUserResponse(tempUser, nil, plugins, verifyStatus, session.ID), ErrPluginRequired
	}

	// 验证通过或不需要验证，创建用户
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

	// 更新验证会话的userID
	if err := s.verificationSvc.UpdateSessionUserID(ctx, session.ID, user.ID); err != nil {
		log.Printf("Failed to update session user ID: %v", err)
	}

	// 立即清理验证状态
	if err := s.verificationSvc.ClearVerification(ctx, appID, user.ID, "register"); err != nil {
		return nil, fmt.Errorf("failed to clear verification status: %v", err)
	}

	// 如果请求中包含Profile信息,创建用户档案
	if req.Profile != nil {
		if _, err := s.profileSvc.CreateProfile(ctx, user.ID, appID, req.Profile); err != nil {
			log.Printf("创建用户档案失败: %v", err)
		}
	}

	// 验证完成，生成token
	tokenPair, err := s.tokenService.GenerateTokenPair(ctx, user, "read")
	if err != nil {
		return nil, err
	}

	return s.buildUserResponse(user, tokenPair, plugins, verifyStatus, session.ID), nil
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

	// 处理验证流程
	vCtx := s.buildVerificationContext(appID, user.ID, "login", req)
	session, plugins, verifyStatus, err := s.handleVerification(ctx, vCtx)
	if err != nil {
		return nil, err
	}

	// 如果需要验证，返回验证状态
	if len(plugins) > 0 && !verifyStatus.Completed {
		return s.buildUserResponse(user, nil, plugins, verifyStatus, session.ID), ErrPluginRequired
	}

	// 验证完成，生成token
	tokenPair, err := s.tokenService.GenerateTokenPair(ctx, user, "read")
	if err != nil {
		return nil, err
	}

	// 清理验证状态
	if err := s.verificationSvc.ClearVerification(ctx, appID, user.ID, "login"); err != nil {
		log.Printf("Failed to clear verification: %v", err)
	}

	// 记录登录位置
	if req.ClientIP != "" {
		if err := s.locationSvc.RecordLoginLocation(ctx, appID, user.ID, req.ClientIP); err != nil {
			log.Printf("Failed to record login location: %v", err)
		}
	}

	return s.buildUserResponse(user, tokenPair, plugins, verifyStatus, session.ID), nil
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
	plugins, err := s.verificationSvc.GetRequiredPlugins(ctx, session.AppID, session.Action, session.Context, *session.UserID)
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
		return s.buildUserResponse(user, nil, plugins, verifyStatus, sessionID), ErrPluginRequired
	}

	// 验证完成，生成token
	tokenPair, err := s.tokenService.GenerateTokenPair(ctx, user, "read")
	if err != nil {
		return nil, err
	}

	// 记录登录位置
	if ip, ok := session.Context["ip"].(string); ok && ip != "" {
		if err := s.locationSvc.RecordLoginLocation(ctx, session.AppID, user.ID, ip); err != nil {
			log.Printf("Failed to record login location: %v", err)
		}
	}

	return s.buildUserResponse(user, tokenPair, plugins, verifyStatus, sessionID), nil
}
