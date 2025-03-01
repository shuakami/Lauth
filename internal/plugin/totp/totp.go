package totp

import (
	"context"
	"errors"
	"fmt"
	"image/png"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"lauth/internal/model"
	"lauth/internal/plugin/types"
	"lauth/internal/plugin/verification"
	"lauth/internal/repository"
	"lauth/pkg/container"
)

var (
	// ErrTOTPNotEnabled TOTP未启用
	ErrTOTPNotEnabled = errors.New("totp not enabled for user")
	// ErrTOTPAlreadyEnabled TOTP已启用
	ErrTOTPAlreadyEnabled = errors.New("totp already enabled for user")
	// ErrTOTPVerificationFailed TOTP验证失败
	ErrTOTPVerificationFailed = errors.New("totp verification failed")
	// ErrTOTPInvalidCode TOTP验证码无效
	ErrTOTPInvalidCode = errors.New("invalid totp code")
)

// Config TOTP插件配置
type Config struct {
	Issuer       string `json:"issuer"`        // 发行方
	SecretLength int    `json:"secret_length"` // 密钥长度
	Period       uint   `json:"period"`        // TOTP周期(秒)
	Digits       uint   `json:"digits"`        // 验证码位数
	QRCodeSize   int    `json:"qr_code_size"`  // 二维码尺寸
	Description  string `json:"description"`   // TOTP描述
	AppName      string `json:"app_name"`      // 应用名称
}

// UserConfig 用户TOTP配置
type UserConfig struct {
	Enabled     bool   `json:"enabled"`     // 是否启用
	Secret      string `json:"secret"`      // TOTP密钥
	Description string `json:"description"` // TOTP描述
	AppName     string `json:"app_name"`    // 应用名称
}

// TOTPPlugin TOTP二因素认证插件
type TOTPPlugin struct {
	metadata      *types.PluginMetadata
	config        Config
	userConfigSvc types.UserConfigManager
	sessionRepo   repository.VerificationSessionRepository
	appID         string
}

// NewTOTPPlugin 创建TOTP插件实例
func NewTOTPPlugin() *TOTPPlugin {
	return &TOTPPlugin{
		metadata: &types.PluginMetadata{
			Name:        "totp",
			Description: "基于时间的一次性密码(TOTP)二因素认证",
			Version:     "1.0.0",
			Author:      "AuthSystem",
			Required:    false,
			Stage:       model.PluginStagePostLogin,
			Actions:     []string{"login"},
			Operations: []types.OperationMetadata{
				{
					Name:        "setup",
					Description: "设置TOTP二因素认证",
					Parameters: map[string]string{
						"user_id": "用户ID",
					},
					Returns: map[string]string{
						"secret":      "TOTP密钥",
						"qr_code_url": "二维码URL",
					},
				},
				{
					Name:        "verify",
					Description: "验证TOTP验证码",
					Parameters: map[string]string{
						"user_id": "用户ID",
						"code":    "TOTP验证码",
					},
					Returns: map[string]string{
						"success": "验证结果",
					},
				},
				{
					Name:        "disable",
					Description: "禁用TOTP二因素认证",
					Parameters: map[string]string{
						"user_id": "用户ID",
						"code":    "TOTP验证码",
					},
					Returns: map[string]string{
						"success": "操作结果",
					},
				},
			},
		},
	}
}

// Name 返回插件名称
func (p *TOTPPlugin) Name() string {
	return p.metadata.Name
}

// GetMetadata 返回插件元数据
func (p *TOTPPlugin) GetMetadata() *types.PluginMetadata {
	return p.metadata
}

// Load 加载插件
func (p *TOTPPlugin) Load(config map[string]interface{}) error {
	// 设置默认配置
	p.config = Config{
		Issuer:       "AuthSystem",
		SecretLength: 16,
		Period:       30,
		Digits:       6,
		QRCodeSize:   200,
	}

	// 覆盖自定义配置
	if issuer, ok := config["issuer"].(string); ok && issuer != "" {
		p.config.Issuer = issuer
	}
	if secretLength, ok := config["secret_length"].(float64); ok && secretLength > 0 {
		p.config.SecretLength = int(secretLength)
	}
	if period, ok := config["period"].(float64); ok && period > 0 {
		p.config.Period = uint(period)
	}
	if digits, ok := config["digits"].(float64); ok && digits > 0 {
		p.config.Digits = uint(digits)
	}
	if qrCodeSize, ok := config["qr_code_size"].(float64); ok && qrCodeSize > 0 {
		p.config.QRCodeSize = int(qrCodeSize)
	}

	return nil
}

// Unload 卸载插件
func (p *TOTPPlugin) Unload() error {
	return nil
}

// Start 启动插件
func (p *TOTPPlugin) Start() error {
	return nil
}

// Stop 停止插件
func (p *TOTPPlugin) Stop() error {
	return nil
}

// GetConfig 获取配置
func (p *TOTPPlugin) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"issuer":        p.config.Issuer,
		"secret_length": p.config.SecretLength,
		"period":        p.config.Period,
		"digits":        p.config.Digits,
		"qr_code_size":  p.config.QRCodeSize,
	}
}

// UpdateConfig 更新配置
func (p *TOTPPlugin) UpdateConfig(config map[string]interface{}) error {
	return p.Load(config)
}

// ValidateConfig 验证配置
func (p *TOTPPlugin) ValidateConfig(config map[string]interface{}) error {
	// 验证配置是否合法
	if issuer, ok := config["issuer"].(string); ok && issuer == "" {
		return types.NewPluginError(types.ErrConfigInvalid, "issuer cannot be empty", nil)
	}
	if secretLength, ok := config["secret_length"].(float64); ok && secretLength <= 0 {
		return types.NewPluginError(types.ErrConfigInvalid, "secret_length must be positive", nil)
	}
	if period, ok := config["period"].(float64); ok && period <= 0 {
		return types.NewPluginError(types.ErrConfigInvalid, "period must be positive", nil)
	}
	if digits, ok := config["digits"].(float64); ok && digits <= 0 {
		return types.NewPluginError(types.ErrConfigInvalid, "digits must be positive", nil)
	}
	if qrCodeSize, ok := config["qr_code_size"].(float64); ok && qrCodeSize <= 0 {
		return types.NewPluginError(types.ErrConfigInvalid, "qr_code_size must be positive", nil)
	}
	return nil
}

// GetDependencies 获取依赖
func (p *TOTPPlugin) GetDependencies() []string {
	return []string{
		"user_config_repo",
		"verification_session_repo",
	}
}

// Configure 配置插件
func (p *TOTPPlugin) Configure(container container.PluginContainer) error {
	// 获取用户配置仓储
	userConfigRepo, err := container.Resolve("user_config_repo")
	if err != nil {
		return fmt.Errorf("failed to get user_config_repo: %v", err)
	}

	// 获取会话仓储
	sessionRepo, err := container.Resolve("verification_session_repo")
	if err != nil {
		return fmt.Errorf("failed to get verification_session_repo: %v", err)
	}

	// 类型断言
	repo, ok := sessionRepo.(repository.VerificationSessionRepository)
	if !ok {
		return fmt.Errorf("verification_session_repo is not a VerificationSessionRepository")
	}
	p.sessionRepo = repo

	// 创建用户配置服务
	p.userConfigSvc = verification.NewDefaultUserConfigManager(userConfigRepo.(repository.PluginUserConfigRepository), p.appID, p.Name())

	return nil
}

// OnInstall 安装插件
func (p *TOTPPlugin) OnInstall(appID string) error {
	p.appID = appID
	return nil
}

// OnUninstall 卸载插件
func (p *TOTPPlugin) OnUninstall(appID string) error {
	return nil
}

// NeedsVerification 判断是否需要验证
func (p *TOTPPlugin) NeedsVerification(ctx context.Context, userID string, action string, context map[string]interface{}) (bool, error) {
	// 仅对登录操作进行验证
	if action != "login" {
		fmt.Printf("[TOTP] 非登录操作，不需要验证: action=%s\n", action)
		return false, nil
	}

	// 如果没有userID，尝试从session中获取
	if userID == "" {
		if sessionID, ok := context["session_id"].(string); ok && sessionID != "" {
			fmt.Printf("[TOTP] 从session获取用户信息: session_id=%s\n", sessionID)
			// 获取session
			session, err := p.getSession(ctx, sessionID)
			if err != nil {
				fmt.Printf("[TOTP] 获取session失败: error=%v\n", err)
				return false, err
			}
			if session != nil && session.UserID != nil {
				userID = *session.UserID
				fmt.Printf("[TOTP] 从session获取到userID: user_id=%s\n", userID)
			}
		}
	}

	// 如果仍然没有userID，说明用户未登录，不需要验证
	if userID == "" {
		fmt.Printf("[TOTP] 未获取到userID，不需要验证\n")
		return false, nil
	}

	// 获取用户TOTP配置
	config, err := p.getUserConfig(ctx, userID)
	if err != nil {
		return false, err
	}

	return config != nil && config.Enabled, nil
}

// getSession 获取验证会话
func (p *TOTPPlugin) getSession(ctx context.Context, sessionID string) (*model.VerificationSession, error) {
	// 增加判空，避免空指针
	if p.sessionRepo == nil {
		return nil, fmt.Errorf("[TOTPPlugin] sessionRepo is nil; plugin not configured or not installed properly")
	}
	return p.sessionRepo.GetByID(ctx, sessionID)
}

// ValidateVerification 验证当前验证是否有效
func (p *TOTPPlugin) ValidateVerification(ctx context.Context, userID string, action string, verificationID string) (bool, error) {
	// 获取用户TOTP配置
	userConfig, err := p.getUserConfig(ctx, userID)
	if err != nil {
		return false, err
	}

	// 如果TOTP未启用，则不需要验证
	if !userConfig.Enabled {
		return true, nil
	}

	// 验证TOTP验证码
	return totp.Validate(verificationID, userConfig.Secret), nil
}

// OnVerificationSuccess 验证成功回调
func (p *TOTPPlugin) OnVerificationSuccess(ctx context.Context, userID string, action string, context map[string]interface{}) error {
	// 无需额外操作
	return nil
}

// GetLastVerification 获取上次验证信息
func (p *TOTPPlugin) GetLastVerification(ctx context.Context, userID string, action string) (*model.PluginStatus, error) {
	// TOTP没有上次验证信息
	return nil, nil
}

// RegisterRoutes 注册路由
func (p *TOTPPlugin) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/setup", p.handleSetup)
	group.POST("/verify", p.handleVerify)
	group.POST("/disable", p.handleDisable)
}

// GetAPIInfo 获取API信息
func (p *TOTPPlugin) GetAPIInfo() []types.APIInfo {
	return []types.APIInfo{
		{
			Method:      "POST",
			Path:        "/setup",
			Description: "设置TOTP二因素认证",
		},
		{
			Method:      "POST",
			Path:        "/verify",
			Description: "验证TOTP验证码",
		},
		{
			Method:      "POST",
			Path:        "/disable",
			Description: "禁用TOTP二因素认证",
		},
	}
}

// GetRoutesRequireAuth 获取需要认证的路由列表
func (p *TOTPPlugin) GetRoutesRequireAuth() []string {
	return []string{
		"/setup",   // 设置TOTP需要认证
		"/disable", // 禁用TOTP需要认证
	}
}

// NeedsVerificationSession 判断指定操作是否需要验证会话
func (p *TOTPPlugin) NeedsVerificationSession(operation string) bool {
	// 只有setup和verify操作需要验证会话
	return operation == "setup" || operation == "verify"
}

func (p *TOTPPlugin) handleSetup(c *gin.Context) {
	p.handleOperation(c, "setup")
}

func (p *TOTPPlugin) handleVerify(c *gin.Context) {
	p.handleOperation(c, "verify")
}

func (p *TOTPPlugin) handleDisable(c *gin.Context) {
	p.handleOperation(c, "disable")
}

// handleOperation 将三个路由的JSON解析与Execute逻辑统一处理
func (p *TOTPPlugin) handleOperation(c *gin.Context, op string) {
	// 从context获取已安装的插件实例
	pluginInterface, exists := c.Get("installed_plugin")
	if !exists {
		c.JSON(500, gin.H{"error": "插件实例未找到"})
		return
	}

	realPlugin, ok := pluginInterface.(*TOTPPlugin)
	if !ok {
		c.JSON(500, gin.H{"error": "插件类型错误"})
		return
	}

	// 解析请求体
	// {
	//    "operation": "setup",
	//    "params": {...},
	//    "session_id": "xxxx"
	// }
	var body struct {
		Operation string                 `json:"operation"`
		Params    map[string]interface{} `json:"params"`
		SessionID string                 `json:"session_id"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "无效的请求参数"})
		return
	}
	// 如果请求中没带 operation，则使用路由给定的 op
	if body.Operation == "" {
		body.Operation = op
	}
	// 如果原请求没有 params，初始化一个
	if body.Params == nil {
		body.Params = make(map[string]interface{})
	}
	// 最终强制令 "operation" 为 op
	body.Params["operation"] = body.Operation

	// 若有 session_id，则从session获取userID
	if body.SessionID != "" {
		body.Params["session_id"] = body.SessionID
		// 获取session
		session, err := realPlugin.getSession(c.Request.Context(), body.SessionID)
		if err != nil {
			c.JSON(400, gin.H{"error": "无效的会话ID"})
			return
		}
		// 检查session和userID是否为nil
		if session == nil {
			c.JSON(400, gin.H{"error": "会话不存在"})
			return
		}
		if session.UserID == nil {
			c.JSON(400, gin.H{"error": "会话中未找到用户ID"})
			return
		}
		// 从session获取userID并添加到params
		body.Params["user_id"] = *session.UserID
	}

	// Execute 调用
	err := realPlugin.Execute(c.Request.Context(), body.Params)
	if err != nil {
		// 如果执行出错，根据错误类型给出相应返回
		switch err {
		case ErrTOTPNotEnabled:
			c.JSON(400, gin.H{"error": "TOTP未启用", "success": false})
			return
		case ErrTOTPAlreadyEnabled:
			c.JSON(400, gin.H{"error": "TOTP已启用", "success": false})
			return
		case ErrTOTPInvalidCode:
			c.JSON(400, gin.H{"error": "验证码无效", "success": false})
			return
		case ErrTOTPVerificationFailed:
			c.JSON(400, gin.H{"error": "验证失败", "success": false})
			return
		default:
			c.JSON(500, gin.H{"error": err.Error(), "success": false})
			return
		}
	}

	// 如果是setup操作，返回secret和二维码URL
	if body.Operation == "setup" {
		userConfig, err := realPlugin.getUserConfig(c.Request.Context(), body.Params["user_id"].(string))
		if err != nil {
			c.JSON(500, gin.H{"error": "获取用户配置失败", "success": false})
			return
		}
		c.JSON(200, gin.H{
			"success": true,
			"data": gin.H{
				"secret":     userConfig.Secret,
				"qr_code":    "/api/v1/files/totp/" + body.Params["user_id"].(string) + ".png",
				"app_name":   userConfig.AppName,
				"enabled":    userConfig.Enabled,
				"operation":  body.Operation,
				"session_id": body.SessionID,
			},
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"operation":  body.Operation,
			"session_id": body.SessionID,
		},
	})
}

// Execute 执行插件
func (p *TOTPPlugin) Execute(ctx context.Context, params map[string]interface{}) error {
	// 验证参数
	operation, userID, err := p.validateExecuteParams(params)
	if err != nil {
		return err
	}

	// 根据操作类型执行不同的逻辑
	switch operation {
	case "setup":
		return p.handleSetupOperation(ctx, userID, params)
	case "verify":
		return p.handleVerifyOperation(ctx, userID, params)
	case "disable":
		return p.handleDisableOperation(ctx, userID, params)
	default:
		return types.NewPluginError(types.ErrInvalidState, "unsupported operation", nil)
	}
}

// validateExecuteParams 验证执行参数
func (p *TOTPPlugin) validateExecuteParams(params map[string]interface{}) (operation string, userID string, err error) {
	// 获取操作类型
	operation, ok := params["operation"].(string)
	if !ok {
		return "", "", fmt.Errorf("[INVALID_PARAMETER] missing operation parameter")
	}

	// 获取用户ID
	userID, ok = params["user_id"].(string)
	if !ok || userID == "" {
		// 尝试从session获取userID
		if sessionID, ok := params["session_id"].(string); ok && sessionID != "" {
			session, err := p.getSession(context.Background(), sessionID)
			if err != nil {
				return "", "", fmt.Errorf("[INTERNAL_ERROR] failed to get session: %v", err)
			}
			if session != nil && session.UserID != nil {
				userID = *session.UserID
			}
		}

		if userID == "" {
			return "", "", fmt.Errorf("[INVALID_STATE] missing user_id parameter")
		}
	}

	return operation, userID, nil
}

// handleSetupOperation 处理设置TOTP操作
func (p *TOTPPlugin) handleSetupOperation(ctx context.Context, userID string, params map[string]interface{}) error {
	// 获取描述和应用名称参数
	description, _ := params["description"].(string)
	appName, _ := params["app_name"].(string)

	// 生成新的TOTP密钥
	key, url, err := p.generateTOTP(userID)
	if err != nil {
		return types.NewPluginError(types.ErrExecuteFailed, "failed to generate TOTP", err)
	}

	// 获取用户配置
	userConfig, err := p.getUserConfig(ctx, userID)
	if err != nil {
		return types.NewPluginError(types.ErrExecuteFailed, "failed to get user config", err)
	}

	// 检查是否已启用
	if userConfig.Enabled {
		return ErrTOTPAlreadyEnabled
	}

	// 更新用户配置
	userConfig.Secret = key.Secret()
	userConfig.Enabled = true
	userConfig.Description = description
	userConfig.AppName = appName

	// 保存配置
	if err := p.saveUserConfig(ctx, userID, userConfig); err != nil {
		return types.NewPluginError(types.ErrExecuteFailed, "failed to save user config", err)
	}

	// 将结果添加到响应参数中（路由层将会返回）
	params["secret"] = key.Secret()
	params["url"] = url
	return nil
}

// handleVerifyOperation 处理验证TOTP操作
func (p *TOTPPlugin) handleVerifyOperation(ctx context.Context, userID string, params map[string]interface{}) error {
	code, ok := params["code"].(string)
	if !ok {
		return types.NewPluginError(types.ErrInvalidState, "missing code parameter", nil)
	}
	return p.verifyTOTP(ctx, userID, code)
}

// handleDisableOperation 处理禁用TOTP操作
func (p *TOTPPlugin) handleDisableOperation(ctx context.Context, userID string, params map[string]interface{}) error {
	code, ok := params["code"].(string)
	if !ok {
		return types.NewPluginError(types.ErrInvalidState, "missing code parameter", nil)
	}
	return p.disableTOTP(ctx, userID, code)
}

// verifyTOTP 验证TOTP
func (p *TOTPPlugin) verifyTOTP(ctx context.Context, userID string, code string) error {
	// 获取用户TOTP配置
	userConfig, err := p.getUserConfig(ctx, userID)
	if err != nil {
		return err
	}

	// 如果TOTP未启用，则不需要验证
	if !userConfig.Enabled {
		return nil
	}

	// 验证TOTP验证码
	valid := totp.Validate(code, userConfig.Secret)
	if !valid {
		return ErrTOTPVerificationFailed
	}

	return nil
}

// disableTOTP 禁用TOTP
func (p *TOTPPlugin) disableTOTP(ctx context.Context, userID string, code string) error {
	// 获取用户TOTP配置
	userConfig, err := p.getUserConfig(ctx, userID)
	if err != nil {
		return err
	}

	// 检查是否已启用
	if !userConfig.Enabled {
		return ErrTOTPNotEnabled
	}

	// 验证TOTP验证码
	valid := totp.Validate(code, userConfig.Secret)
	if !valid {
		return ErrTOTPInvalidCode
	}

	// 禁用TOTP
	userConfig.Enabled = false
	return p.saveUserConfig(ctx, userID, userConfig)
}

// getUserConfig 获取用户TOTP配置
func (p *TOTPPlugin) getUserConfig(ctx context.Context, userID string) (*UserConfig, error) {
	config, err := p.userConfigSvc.GetConfig(ctx, userID)
	fmt.Printf("[TOTP] 获取用户配置: userID=%s, appID=%s, err=%v\n", userID, p.appID, err)
	if err != nil {
		return &UserConfig{Enabled: false}, nil
	}

	// 查找插件特定配置
	pluginConfig, ok := config[p.Name()].(map[string]interface{})
	fmt.Printf("[TOTP] 插件配置: userID=%s, found=%v, config=%+v\n", userID, ok, config)
	if !ok {
		return &UserConfig{Enabled: false}, nil
	}

	userConfig := &UserConfig{Enabled: false}

	if totpConfig, ok := pluginConfig["totp"].(map[string]interface{}); ok {
		fmt.Printf("[TOTP] 发现配置: %+v\n", totpConfig)
		if enabled, ok := totpConfig["enabled"].(bool); ok {
			userConfig.Enabled = enabled
		}
		if secret, ok := totpConfig["secret"].(string); ok {
			userConfig.Secret = secret
		}
		if description, ok := totpConfig["description"].(string); ok {
			userConfig.Description = description
		}
		if appName, ok := totpConfig["app_name"].(string); ok {
			userConfig.AppName = appName
		}
	}

	fmt.Printf("[TOTP] 解析后的用户配置: enabled=%v, secret=%s, description=%s, app_name=%s\n",
		userConfig.Enabled, userConfig.Secret, userConfig.Description, userConfig.AppName)
	return userConfig, nil
}

// saveUserConfig 保存用户TOTP配置
func (p *TOTPPlugin) saveUserConfig(ctx context.Context, userID string, config *UserConfig) error {
	// 获取现有配置
	existingConfig, err := p.userConfigSvc.GetConfig(ctx, userID)
	if err != nil {
		existingConfig = make(map[string]interface{})
	}

	// 只使用新格式保存配置
	existingConfig[p.Name()] = map[string]interface{}{
		"totp": map[string]interface{}{
			"enabled":     config.Enabled,
			"secret":      config.Secret,
			"description": config.Description,
			"app_name":    config.AppName,
		},
	}

	// 保存整个配置
	return p.userConfigSvc.SaveConfig(ctx, userID, existingConfig)
}

// generateTOTP 生成TOTP密钥和二维码
func (p *TOTPPlugin) generateTOTP(userID string) (*otp.Key, string, error) {
	issuer := p.config.Issuer
	if p.config.AppName != "" {
		issuer = fmt.Sprintf("%s-%s", p.config.AppName, issuer)
	}

	opts := totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: userID,
		SecretSize:  uint(p.config.SecretLength),
		Period:      p.config.Period,
		Digits:      otp.Digits(p.config.Digits),
		Algorithm:   otp.AlgorithmSHA1,
	}

	key, err := totp.Generate(opts)
	if err != nil {
		return nil, "", err
	}

	return key, key.URL(), nil
}

// SaveQRCode 保存二维码到文件
func (p *TOTPPlugin) SaveQRCode(key *otp.Key, filePath string) error {
	// 生成二维码图像
	img, err := key.Image(p.config.QRCodeSize, p.config.QRCodeSize)
	if err != nil {
		return err
	}

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 编码为PNG
	return png.Encode(file, img)
}
