package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"lauth/internal/model"
	"lauth/internal/repository"
)

// authAccountService 账号服务
type authAccountService struct {
	userRepo          repository.UserRepository
	appRepo           repository.AppRepository
	tokenService      TokenService
	verificationSvc   VerificationService
	profileSvc        ProfileService
	locationSvc       LoginLocationService
	superAdminService SuperAdminService
	db                *gorm.DB
}

// newAuthAccountService 创建认证账号服务实例
func newAuthAccountService(
	userRepo repository.UserRepository,
	appRepo repository.AppRepository,
	tokenService TokenService,
	verificationSvc VerificationService,
	profileSvc ProfileService,
	locationSvc LoginLocationService,
	superAdminService SuperAdminService,
	db *gorm.DB,
) *authAccountService {
	return &authAccountService{
		userRepo:          userRepo,
		appRepo:           appRepo,
		tokenService:      tokenService,
		verificationSvc:   verificationSvc,
		profileSvc:        profileSvc,
		locationSvc:       locationSvc,
		superAdminService: superAdminService,
		db:                db,
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
	var lastLoginStr *string
	if user.LastLoginAt != nil {
		formatted := user.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
		lastLoginStr = &formatted
	}

	// 转换密码过期时间
	var passwordExpiresStr *string
	if user.PasswordExpiresAt != nil {
		formatted := user.PasswordExpiresAt.Format("2006-01-02T15:04:05Z07:00")
		passwordExpiresStr = &formatted
	}

	// 只检查是否是首次登录，不再检查密码是否过期
	// 密码过期信息由前端根据PasswordExpiresAt自行判断
	needChangePassword := user.IsFirstLogin

	response := &model.ExtendedLoginResponse{
		User: model.UserResponse{
			ID:                 user.ID,
			AppID:              user.AppID,
			Username:           user.Username,
			Nickname:           user.Nickname,
			Email:              user.Email,
			Phone:              user.Phone,
			Status:             user.Status,
			IsFirstLogin:       user.IsFirstLogin,
			LastLoginAt:        lastLoginStr,
			PasswordExpiresAt:  passwordExpiresStr,
			NeedChangePassword: needChangePassword,
			CreatedAt:          user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:          user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		Plugins:   plugins,
		SessionID: sessionID,
	}

	// 安全地设置NextPlugin，避免nil指针引用
	if verifyStatus != nil {
		response.NextPlugin = verifyStatus.NextPlugin
	}

	if tokenPair != nil {
		response.AccessToken = tokenPair.AccessToken
		response.RefreshToken = tokenPair.RefreshToken
		response.ExpiresIn = int64(tokenPair.AccessTokenExpireIn.Seconds())
		response.AuthStatus = model.PluginStatusCompleted
	} else if verifyStatus != nil {
		response.AuthStatus = verifyStatus.Status
	} else {
		response.AuthStatus = model.PluginStatusCompleted
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
	log.Printf("[DEBUG] 尝试登录用户: %s, AppID: %s", req.Username, appID)
	user, err := s.userRepo.GetByUsername(ctx, appID, req.Username)
	if err != nil {
		log.Printf("[ERROR] 获取用户时出错: %v", err)
		return nil, err
	}
	if user == nil {
		log.Printf("[ERROR] 用户不存在: %s", req.Username)
		return nil, ErrInvalidCredentials
	}

	log.Printf("[DEBUG] 找到用户: ID=%s, 用户名=%s, 状态=%d", user.ID, user.Username, user.Status)

	// 验证密码
	log.Printf("[DEBUG] 正在验证密码，存储的密码哈希长度: %d", len(user.Password))
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		log.Printf("[ERROR] 密码验证失败: %v", err)
		return nil, ErrInvalidCredentials
	}
	log.Printf("[DEBUG] 密码验证成功")

	// 检查用户状态
	if user.Status == model.UserStatusDisabled {
		log.Printf("[ERROR] 用户已禁用")
		return nil, ErrUserDisabled
	}

	// 检查是否是超级管理员
	user.IsSuperAdmin = false
	if userID := user.ID; userID != "" {
		// 使用超级管理员服务检查用户是否是超级管理员
		isSuperAdmin, err := s.superAdminService.IsSuperAdmin(ctx, userID)
		if err != nil {
			log.Printf("[ERROR] 检查超级管理员状态失败: %v", err)
		} else if isSuperAdmin {
			user.IsSuperAdmin = true
			log.Printf("[DEBUG] 用户是超级管理员")
		}
	}

	// 更新用户的最后登录时间
	now := time.Now()
	// 同时更新内存中的用户对象
	user.LastLoginAt = &now
	// 在数据库中更新最后登录时间，但不触发密码哈希
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID, now); err != nil {
		// 仅记录错误，不影响登录流程
		log.Printf("[ERROR] 更新最后登录时间失败: %v", err)
	}

	// 处理验证流程
	// 如果是首次登录的超级管理员，跳过验证
	skipVerification := user.IsSuperAdmin && user.IsFirstLogin
	log.Printf("[DEBUG] 是否跳过验证: %v (IsSuperAdmin=%v, IsFirstLogin=%v)",
		skipVerification, user.IsSuperAdmin, user.IsFirstLogin)

	if !skipVerification {
		vCtx := s.buildVerificationContext(appID, user.ID, "login", req)
		session, plugins, verifyStatus, err := s.handleVerification(ctx, vCtx)
		if err != nil {
			log.Printf("[ERROR] 处理验证失败: %v", err)
			return nil, err
		}

		// 如果需要验证，返回验证状态
		if len(plugins) > 0 && !verifyStatus.Completed {
			log.Printf("[DEBUG] 需要额外验证，插件数量: %d", len(plugins))
			return s.buildUserResponse(user, nil, plugins, verifyStatus, session.ID), ErrPluginRequired
		}
	}

	// 验证完成，生成token
	log.Printf("[DEBUG] 验证完成，正在生成token")
	tokenPair, err := s.tokenService.GenerateTokenPair(ctx, user, "read")
	if err != nil {
		log.Printf("[ERROR] 生成token失败: %v", err)
		return nil, err
	}

	// 清理验证状态
	if !skipVerification {
		if err := s.verificationSvc.ClearVerification(ctx, appID, user.ID, "login"); err != nil {
			log.Printf("Failed to clear verification: %v", err)
		}
	}

	// 记录登录位置
	if req.ClientIP != "" {
		if err := s.locationSvc.RecordLoginLocation(ctx, appID, user.ID, req.ClientIP); err != nil {
			log.Printf("Failed to record login location: %v", err)
		}
	}

	return s.buildUserResponse(user, tokenPair, nil, nil, ""), nil
}
