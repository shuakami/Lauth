package email

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"lauth/internal/model"
	"lauth/internal/plugin/types"
	"lauth/internal/plugin/verification"
	"lauth/internal/repository"
	"lauth/pkg/config"
	"lauth/pkg/container"

	"github.com/gin-gonic/gin"
)

// emailCodeSender 邮件验证码发送器
type emailCodeSender struct {
	sender EmailSender
}

// Send 实现types.CodeSender接口
func (s *emailCodeSender) Send(to string, code string, expireMinutes int) error {
	// 准备模板数据
	data := map[string]interface{}{
		"Code":          code,
		"ExpireMinutes": expireMinutes,
	}

	// 使用HTML模板发送邮件
	return s.sender.SendWithTemplate(
		to,
		"验证码",
		"verification_code",
		data,
	)
}

// emailLinkSender 邮件验证链接发送器
type emailLinkSender struct {
	sender EmailSender
}

// Send 实现types.LinkSender接口
func (s *emailLinkSender) Send(to string, link string, expireMinutes int) error {
	// 准备模板数据
	data := map[string]interface{}{
		"Link":          link,
		"ExpireMinutes": expireMinutes,
	}

	// 使用HTML模板发送邮件
	return s.sender.SendWithTemplate(
		to,
		"Verification Email",
		"verification_link",
		data,
	)
}

// VerificationPolicy 验证策略配置
type VerificationPolicy struct {
	AlwaysVerify   bool          `json:"always_verify"`   // 是否每次都验证
	VerifyInterval time.Duration `json:"verify_interval"` // 两次验证的最小间隔
	ExemptIPs      []string      `json:"exempt_ips"`      // 豁免的IP列表
	ExemptDevices  []string      `json:"exempt_devices"`  // 豁免的设备列表
}

// VerificationMode 验证模式
type VerificationMode string

const (
	// VerificationModeCode 验证码模式
	VerificationModeCode VerificationMode = "code"
	// VerificationModeLink 链接模式
	VerificationModeLink VerificationMode = "link"
)

// EmailConfig 邮件插件配置
type EmailConfig struct {
	CodeLength         int                `json:"code_length"`         // 验证码长度
	ExpireTime         time.Duration      `json:"expire_time"`         // 验证码过期时间
	VerificationPolicy VerificationPolicy `json:"verification_policy"` // 验证策略
	VerificationMode   VerificationMode   `json:"verification_mode"`   // 验证模式
	LinkConfig         *types.LinkConfig  `json:"link_config"`         // 链接验证配置
}

// EmailPlugin 邮件验证插件
type EmailPlugin struct {
	*types.SmartPluginBase

	config        *EmailConfig
	codeManager   types.VerificationCodeManager
	linkManager   types.VerificationLinkManager
	codeSender    *emailCodeSender
	linkSender    *emailLinkSender
	configManager types.UserConfigManager
	verifyRepo    repository.PluginVerificationRecordRepository
	appID         string
	exemptManager *types.ExemptionManager
}

// NewEmailPlugin 创建邮件插件实例
func NewEmailPlugin() types.Plugin {
	p := &EmailPlugin{
		SmartPluginBase: types.NewSmartPlugin(
			types.WithName("email_verify"),
			types.WithVersion("1.0.0"),
			types.WithMetadata(&types.PluginMetadata{
				Name:        "email_verify",
				Description: "通过邮箱验证码进行身份验证",
				Author:      "lauth team",
				Required:    true,
				Stage:       "post_login",
				Actions:     []string{"login", "register", "reset_password"},
				Operations: []types.OperationMetadata{
					{
						Name:        "send",
						Description: "发送验证码到指定邮箱",
						Parameters: map[string]string{
							"email": "接收验证码的邮箱地址",
						},
					},
					{
						Name:        "verify",
						Description: "验证邮箱验证码",
						Parameters: map[string]string{
							"email": "接收验证码的邮箱地址",
							"code":  "用户输入的验证码",
						},
					},
				},
			}),
			types.WithHooks(&emailHooks{}),
		),
		exemptManager: types.NewExemptionManager(),
	}

	// 设置hooks的plugin字段
	if hooks, ok := p.SmartPluginBase.GetHooks().(*emailHooks); ok {
		hooks.plugin = p
	}

	// 添加配置验证器
	p.AddValidator(types.NewBaseConfigValidator(
		types.RequiredValidator("code_length", "expire_time", "verification_policy"),
		types.TypeValidator("code_length", 0),
		types.TypeValidator("expire_time", ""),
		types.TypeValidator("verification_policy", map[string]interface{}{}),
	))

	// 添加豁免规则
	p.exemptManager.AddRule(types.NewIPRule(100))    // IP规则优先级最高
	p.exemptManager.AddRule(types.NewDeviceRule(90)) // 设备规则次之

	// 添加日志中间件
	if p.GetLogger() != nil {
		p.exemptManager.AddMiddleware(types.LoggingMiddleware(p.GetLogger()))
	}

	return p
}

// GetDependencies 获取插件依赖的服务
func (p *EmailPlugin) GetDependencies() []string {
	return []string{
		"user_config_repo",
		"verification_repo",
		"smtp_config",
		"app_id",
	}
}

// Configure 配置插件（注入依赖）
func (p *EmailPlugin) Configure(c container.PluginContainer) error {
	// 解析依赖
	userConfigRepo, err := c.Resolve("user_config_repo")
	if err != nil {
		return fmt.Errorf("failed to resolve user_config_repo: %v", err)
	}

	verifyRepo, err := c.Resolve("verification_repo")
	if err != nil {
		return fmt.Errorf("failed to resolve verification_repo: %v", err)
	}

	smtpConfig, err := c.Resolve("smtp_config")
	if err != nil {
		return fmt.Errorf("failed to resolve smtp_config: %v", err)
	}

	appID, err := c.ResolvePluginService(p.Name(), "app_id")
	if err != nil {
		return fmt.Errorf("failed to resolve app_id: %v", err)
	}

	// 类型断言
	p.appID = appID.(string)
	p.verifyRepo = verifyRepo.(repository.PluginVerificationRecordRepository)

	// 创建邮件发送器
	emailSender := NewDefaultEmailSender(smtpConfig.(*config.SMTPConfig))
	p.codeSender = &emailCodeSender{
		sender: emailSender,
	}
	p.linkSender = &emailLinkSender{
		sender: emailSender,
	}

	// 创建验证码管理器
	p.codeManager = verification.NewDefaultCodeManager(
		&types.VerificationConfig{
			CodeLength: 6,               // 验证码长度
			ExpireTime: 5 * time.Minute, // 验证码过期时间
		},
		p.codeSender,
	)

	// 创建链接验证管理器
	p.linkManager = verification.NewDefaultLinkManager(
		&types.VerificationConfig{
			ExpireTime: 30 * time.Minute, // 链接默认30分钟过期
		},
		&types.LinkConfig{
			BaseURL:     "http://localhost:8080/verify", // 默认验证URL
			TokenLength: 32,                             // 默认token长度
			ExpireTime:  30 * time.Minute,               // 默认30分钟过期
		},
		p.linkSender,
	)

	// 创建配置管理器
	p.configManager = verification.NewDefaultUserConfigManager(
		userConfigRepo.(repository.PluginUserConfigRepository),
		p.appID,
		p.Name(),
	)

	return nil
}

// emailHooks 邮件插件钩子
type emailHooks struct {
	types.BaseHooks
	plugin *EmailPlugin
}

// OnLoad 加载配置
func (h *emailHooks) OnLoad(config map[string]interface{}) error {
	// 解析配置
	cfg := &EmailConfig{
		CodeLength:       6,                    // 验证码长度
		ExpireTime:       5 * time.Minute,      // 验证码过期时间
		VerificationMode: VerificationModeCode, // 默认使用验证码模式
		VerificationPolicy: VerificationPolicy{
			VerifyInterval: 24 * time.Hour, // 两次验证的最小间隔
		},
		LinkConfig: &types.LinkConfig{
			BaseURL:     "http://localhost:8080/verify", // 默认验证URL
			TokenLength: 32,                             // 默认token长度
			ExpireTime:  30 * time.Minute,               // 默认30分钟过期
		},
	}

	// 读取验证码长度
	if codeLength, ok := config["code_length"].(int); ok {
		cfg.CodeLength = codeLength
	}

	// 读取过期时间
	if expireTime, ok := config["expire_time"].(string); ok {
		duration, err := time.ParseDuration(expireTime)
		if err != nil {
			return fmt.Errorf("invalid expire_time format: %v", err)
		}
		cfg.ExpireTime = duration
	}

	// 读取验证策略
	if policyConfig, ok := config["verification_policy"].(map[string]interface{}); ok {
		if alwaysVerify, ok := policyConfig["always_verify"].(bool); ok {
			cfg.VerificationPolicy.AlwaysVerify = alwaysVerify
		}

		if verifyInterval, ok := policyConfig["verify_interval"].(string); ok {
			duration, err := time.ParseDuration(verifyInterval)
			if err != nil {
				return fmt.Errorf("invalid verify_interval format: %v", err)
			}
			cfg.VerificationPolicy.VerifyInterval = duration
		}

		if exemptIPs, ok := policyConfig["exempt_ips"].([]interface{}); ok {
			fmt.Printf("exempt_ips: %+v\n", exemptIPs)
			for _, ip := range exemptIPs {
				if ipStr, ok := ip.(string); ok {
					cfg.VerificationPolicy.ExemptIPs = append(cfg.VerificationPolicy.ExemptIPs, ipStr)
				}
			}
		}

		if exemptDevices, ok := policyConfig["exempt_devices"].([]interface{}); ok {
			fmt.Printf("exempt_devices: %+v\n", exemptDevices)
			for _, device := range exemptDevices {
				if deviceStr, ok := device.(string); ok {
					cfg.VerificationPolicy.ExemptDevices = append(cfg.VerificationPolicy.ExemptDevices, deviceStr)
				}
			}
		}
	}

	// 读取验证模式
	if mode, ok := config["verification_mode"].(string); ok {
		cfg.VerificationMode = VerificationMode(mode)
	}

	// 读取链接配置
	if linkConfig, ok := config["link_config"].(map[string]interface{}); ok {
		if baseURL, ok := linkConfig["base_url"].(string); ok {
			cfg.LinkConfig.BaseURL = baseURL
		}
		if tokenLength, ok := linkConfig["token_length"].(int); ok {
			cfg.LinkConfig.TokenLength = tokenLength
		}
		if expireTime, ok := linkConfig["expire_time"].(string); ok {
			duration, err := time.ParseDuration(expireTime)
			if err != nil {
				return fmt.Errorf("invalid link expire_time format: %v", err)
			}
			cfg.LinkConfig.ExpireTime = duration
		}
	}

	h.plugin.config = cfg

	// 更新验证码管理器配置
	h.plugin.codeManager = verification.NewDefaultCodeManager(
		&types.VerificationConfig{
			CodeLength: cfg.CodeLength,
			ExpireTime: cfg.ExpireTime,
		},
		h.plugin.codeSender,
	)

	// 更新链接验证管理器配置
	h.plugin.linkManager = verification.NewDefaultLinkManager(
		&types.VerificationConfig{
			ExpireTime: cfg.LinkConfig.ExpireTime,
		},
		cfg.LinkConfig,
		h.plugin.linkSender,
	)

	return nil
}

// OnExecute 执行插件逻辑
func (h *emailHooks) OnExecute(ctx context.Context, params map[string]interface{}) error {
	email, ok := params["email"].(string)
	if !ok {
		return fmt.Errorf("email parameter is required")
	}

	operation, ok := params["operation"].(string)
	if !ok {
		return fmt.Errorf("operation parameter is required")
	}

	// 获取session_id（如果存在）
	sessionID, _ := params["session_id"].(string)

	// 根据验证模式选择不同的处理逻辑
	switch h.plugin.config.VerificationMode {
	case VerificationModeCode:
		return h.handleCodeMode(email, operation, params)
	case VerificationModeLink:
		return h.handleLinkMode(email, operation, sessionID, params)
	default:
		return fmt.Errorf("unsupported verification mode: %s", h.plugin.config.VerificationMode)
	}
}

// handleCodeMode 处理验证码模式
func (h *emailHooks) handleCodeMode(email string, operation string, params map[string]interface{}) error {
	switch operation {
	case "send":
		return h.plugin.codeManager.Send(email)
	case "verify":
		code, ok := params["code"].(string)
		if !ok {
			return fmt.Errorf("code parameter is required for verify operation")
		}
		return h.plugin.codeManager.Verify(email, code)
	default:
		return fmt.Errorf("unsupported operation for code mode: %s", operation)
	}
}

// handleLinkMode 处理链接模式
func (h *emailHooks) handleLinkMode(email string, operation string, sessionID string, params map[string]interface{}) error {
	switch operation {
	case "send":
		if sessionID == "" {
			return fmt.Errorf("session_id parameter is required for send operation")
		}
		return h.plugin.linkManager.Send(email, sessionID)
	case "verify":
		token, ok := params["token"].(string)
		if !ok {
			return fmt.Errorf("token parameter is required for verify operation")
		}
		// 验证token并获取session_id
		verifiedSessionID, err := h.plugin.linkManager.Verify(email, token)
		if err != nil {
			return err
		}
		// 将session_id添加到params中
		params["session_id"] = verifiedSessionID
		return nil
	default:
		return fmt.Errorf("unsupported operation for link mode: %s", operation)
	}
}

// NeedsVerification 判断是否需要验证
func (p *EmailPlugin) NeedsVerification(ctx context.Context, userID string, action string, context map[string]interface{}) (bool, error) {
	if p.GetState() != types.StateRunning {
		fmt.Printf("插件状态不是running,返回false\n")
		return false, nil
	}

	fmt.Printf("开始验证检查:\n")
	fmt.Printf("- userID: %s\n", userID)
	fmt.Printf("- action: %s\n", action)
	fmt.Printf("- context: %+v\n", context)
	fmt.Printf("- always_verify: %v\n", p.config.VerificationPolicy.AlwaysVerify)
	fmt.Printf("- exempt_ips: %v\n", p.config.VerificationPolicy.ExemptIPs)
	fmt.Printf("- exempt_devices: %v\n", p.config.VerificationPolicy.ExemptDevices)

	// 获取用户配置
	userConfig, err := p.GetUserConfig(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user config: %v", err)
	}
	fmt.Printf("用户配置: %+v\n", userConfig)

	// 构建豁免配置
	globalExempts := map[string]interface{}{
		"exempt_ips":     p.config.VerificationPolicy.ExemptIPs,
		"exempt_devices": p.config.VerificationPolicy.ExemptDevices,
	}

	// 检查IP豁免
	if clientIP, ok := context["ip"].(string); ok {
		fmt.Printf("检查IP豁免: %s\n", clientIP)
		result, err := p.exemptManager.CheckExemption(ctx, types.ExemptionTypeIP, clientIP, userConfig, globalExempts)
		if err != nil {
			return false, fmt.Errorf("failed to check ip exemption: %v", err)
		}
		fmt.Printf("IP豁免检查结果: exempt=%v, reason=%s\n", result.Exempt, result.Reason)
		if result.Exempt {
			fmt.Printf("IP在豁免列表中,返回false\n")
			return false, nil
		}
	}

	// 检查设备豁免
	if deviceID, ok := context["device_id"].(string); ok {
		fmt.Printf("检查设备豁免: %s\n", deviceID)
		result, err := p.exemptManager.CheckExemption(ctx, types.ExemptionTypeDevice, deviceID, userConfig, globalExempts)
		if err != nil {
			return false, fmt.Errorf("failed to check device exemption: %v", err)
		}
		fmt.Printf("设备豁免检查结果: exempt=%v, reason=%s\n", result.Exempt, result.Reason)
		if result.Exempt {
			fmt.Printf("设备在豁免列表中,返回false\n")
			return false, nil
		}
	}

	// 如果设置了总是验证，且不在豁免列表中，返回true
	if p.config.VerificationPolicy.AlwaysVerify {
		fmt.Printf("设置了always_verify=true且不在豁免列表中,返回true\n")
		return true, nil
	}

	// 检查验证间隔
	if userConfig != nil {
		if lastVerifyTimeStr, ok := userConfig["last_verify_time"].(string); ok {
			lastVerifyTime, err := time.Parse(time.RFC3339, lastVerifyTimeStr)
			if err == nil {
				interval := time.Since(lastVerifyTime)
				fmt.Printf("上次验证时间: %v, 间隔: %v, 最小间隔: %v\n",
					lastVerifyTime, interval, p.config.VerificationPolicy.VerifyInterval)
				if interval < p.config.VerificationPolicy.VerifyInterval {
					fmt.Printf("在验证间隔内,返回false\n")
					return false, nil
				}
			}
		}
	}

	fmt.Printf("默认返回true\n")
	return true, nil
}

// GetUserConfig 获取用户配置
func (p *EmailPlugin) GetUserConfig(ctx context.Context, userID string) (map[string]interface{}, error) {
	return p.configManager.GetConfig(ctx, userID)
}

// UpdateUserConfig 更新用户配置
func (p *EmailPlugin) UpdateUserConfig(ctx context.Context, userID string, config map[string]interface{}) error {
	return p.configManager.SaveConfig(ctx, userID, config)
}

// OnVerificationSuccess 验证成功回调
func (p *EmailPlugin) OnVerificationSuccess(ctx context.Context, userID string, action string, context map[string]interface{}) error {
	// 保存验证记录
	record := &model.PluginVerificationRecord{
		AppID:      p.appID,
		UserID:     userID,
		Plugin:     p.Name(),
		Action:     action,
		Context:    context,
		VerifiedAt: time.Now(),
	}
	if err := p.verifyRepo.SaveRecord(ctx, record); err != nil {
		return fmt.Errorf("failed to save verification record: %v", err)
	}

	// 获取当前用户配置
	currentConfig, err := p.GetUserConfig(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user config: %v", err)
	}
	if currentConfig == nil {
		currentConfig = make(map[string]interface{})
	}

	// 更新可信IP
	if clientIP, ok := context["ip"].(string); ok {
		var exemptIPs []string
		if existingIPs, ok := currentConfig["exempt_ips"].([]interface{}); ok {
			for _, ip := range existingIPs {
				if ipStr, ok := ip.(string); ok {
					exemptIPs = append(exemptIPs, ipStr)
				}
			}
		}
		if !contains(exemptIPs, clientIP) {
			exemptIPs = append(exemptIPs, clientIP)
			currentConfig["exempt_ips"] = exemptIPs
		}
	}

	// 更新可信设备
	if deviceID, ok := context["device_id"].(string); ok {
		var exemptDevices []string
		if existingDevices, ok := currentConfig["exempt_devices"].([]interface{}); ok {
			for _, device := range existingDevices {
				if deviceStr, ok := device.(string); ok {
					exemptDevices = append(exemptDevices, deviceStr)
				}
			}
		}
		if !contains(exemptDevices, deviceID) {
			exemptDevices = append(exemptDevices, deviceID)
			currentConfig["exempt_devices"] = exemptDevices
		}
	}

	// 更新最后验证时间
	currentConfig["last_verify_time"] = time.Now().Format(time.RFC3339)

	// 保存更新后的配置
	return p.UpdateUserConfig(ctx, userID, currentConfig)
}

// GetLastVerification 获取上次验证信息
func (p *EmailPlugin) GetLastVerification(ctx context.Context, userID string, action string) (*model.PluginStatus, error) {
	record, err := p.verifyRepo.GetLastRecord(ctx, p.appID, userID, p.Name(), action)
	if err != nil {
		return nil, fmt.Errorf("failed to get last verification record: %v", err)
	}

	if record == nil {
		return nil, nil
	}

	return &model.PluginStatus{
		AppID:     record.AppID,
		UserID:    &record.UserID,
		Action:    record.Action,
		Plugin:    record.Plugin,
		Status:    model.PluginStatusCompleted,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}, nil
}

// ValidateVerification 验证当前验证是否有效
func (p *EmailPlugin) ValidateVerification(ctx context.Context, userID string, action string, verificationID string) (bool, error) {
	return p.codeManager.IsValid(verificationID), nil
}

// SetAppID 设置应用ID
func (p *EmailPlugin) SetAppID(appID string) {
	p.appID = appID
	// 更新配置管理器的appID
	if manager, ok := p.configManager.(*verification.DefaultUserConfigManager); ok {
		manager.SetAppID(appID)
	}
}

// contains 检查字符串是否在切片中
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RegisterRoutes 注册插件路由
func (p *EmailPlugin) RegisterRoutes(group *gin.RouterGroup) {
	// 发送验证码或链接
	group.POST("/send", func(c *gin.Context) {
		var req struct {
			Email     string `json:"email" binding:"required,email"`
			SessionID string `json:"session_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 根据验证模式选择发送方式
		var err error
		switch p.config.VerificationMode {
		case VerificationModeCode:
			err = p.codeManager.Send(req.Email)
		case VerificationModeLink:
			err = p.linkManager.Send(req.Email, req.SessionID)
		default:
			err = fmt.Errorf("unsupported verification mode: %s", p.config.VerificationMode)
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusOK)
	})

	// 验证验证码或链接
	group.POST("/verify", func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
			Code  string `json:"code"`  // 验证码模式使用
			Token string `json:"token"` // 链接模式使用
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 根据验证模式选择验证方式
		var err error
		var sessionID string
		switch p.config.VerificationMode {
		case VerificationModeCode:
			if req.Code == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "code is required for code mode"})
				return
			}
			err = p.codeManager.Verify(req.Email, req.Code)
		case VerificationModeLink:
			if req.Token == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "token is required for link mode"})
				return
			}
			sessionID, err = p.linkManager.Verify(req.Email, req.Token)
		default:
			err = fmt.Errorf("unsupported verification mode: %s", p.config.VerificationMode)
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 如果是链接模式，返回session_id
		if p.config.VerificationMode == VerificationModeLink {
			c.JSON(http.StatusOK, gin.H{"session_id": sessionID})
		} else {
			c.Status(http.StatusOK)
		}
	})

	// 链接验证的重定向处理（用于点击邮件中的链接）
	group.GET("/verify", func(c *gin.Context) {
		// 只在链接模式下处理
		if p.config.VerificationMode != VerificationModeLink {
			c.String(http.StatusBadRequest, "Verification link mode is not enabled")
			return
		}

		email := c.Query("email")
		token := c.Query("token")

		if email == "" || token == "" {
			c.String(http.StatusBadRequest, "Missing email or token")
			return
		}

		sessionID, err := p.linkManager.Verify(email, token)
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("Verification failed: %v", err))
			return
		}

		// 验证成功，重定向到成功页面，并带上session_id
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/success?session_id=%s", p.config.LinkConfig.BaseURL, sessionID))
	})
}

// OnInstall 插件安装时的回调
func (p *EmailPlugin) OnInstall(appID string) error {
	// 设置AppID
	p.SetAppID(appID)
	return nil
}

// OnUninstall 插件卸载时的回调
func (p *EmailPlugin) OnUninstall(appID string) error {
	return nil
}
