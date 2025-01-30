package email

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"lauth/internal/plugin/types"
)

// EmailPlugin 邮箱验证插件
type EmailPlugin struct {
	enabled     bool
	codeLength  int
	expireTime  time.Duration
	verifyStore map[string]verifyCode // 简单起见，先用map存储
}

type verifyCode struct {
	code      string
	expiredAt time.Time
}

// NewEmailPlugin 创建邮箱验证插件实例
func NewEmailPlugin() types.Plugin {
	return &EmailPlugin{
		verifyStore: make(map[string]verifyCode),
	}
}

// Name 返回插件名称
func (p *EmailPlugin) Name() string {
	return "email_verify"
}

// GetMetadata 返回插件元数据
func (p *EmailPlugin) GetMetadata() *types.PluginMetadata {
	return &types.PluginMetadata{
		Name:        "email_verify",
		Description: "通过邮箱验证码进行身份验证",
		Version:     "1.0.0",
		Author:      "lauth team",
		Required:    true,
		Stage:       "post_login",
		Actions:     []string{"login", "register", "reset_password"},
	}
}

// Load 加载插件
func (p *EmailPlugin) Load(config map[string]interface{}) error {
	// 从配置中读取参数
	if codeLength, ok := config["code_length"].(int); ok {
		p.codeLength = codeLength
	} else {
		p.codeLength = 6 // 默认长度
	}

	if expireTime, ok := config["expire_time"].(string); ok {
		duration, err := time.ParseDuration(expireTime)
		if err != nil {
			return fmt.Errorf("invalid expire_time format: %v", err)
		}
		p.expireTime = duration
	} else {
		p.expireTime = 5 * time.Minute // 默认5分钟
	}

	p.enabled = true
	return nil
}

// Unload 卸载插件
func (p *EmailPlugin) Unload() error {
	p.enabled = false
	p.verifyStore = make(map[string]verifyCode)
	return nil
}

// Execute 执行插件逻辑
func (p *EmailPlugin) Execute(ctx context.Context, params map[string]interface{}) error {
	if !p.enabled {
		return fmt.Errorf("plugin is not enabled")
	}

	email, ok := params["email"].(string)
	if !ok {
		return fmt.Errorf("email parameter is required")
	}

	action, ok := params["action"].(string)
	if !ok {
		return fmt.Errorf("action parameter is required")
	}

	switch action {
	case "send":
		return p.sendVerifyCode(email)
	case "verify":
		code, ok := params["code"].(string)
		if !ok {
			return fmt.Errorf("code parameter is required for verify action")
		}
		return p.verifyCode(email, code)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// sendVerifyCode 发送验证码
func (p *EmailPlugin) sendVerifyCode(email string) error {
	// 生成验证码
	code := p.generateCode()

	// TODO: 实际发送邮件的逻辑
	fmt.Printf("Send verify code %s to email %s\n", code, email)

	// 保存验证码
	p.verifyStore[email] = verifyCode{
		code:      code,
		expiredAt: time.Now().Add(p.expireTime),
	}

	return nil
}

// verifyCode 验证验证码
func (p *EmailPlugin) verifyCode(email, code string) error {
	stored, exists := p.verifyStore[email]
	if !exists {
		return fmt.Errorf("no verify code found for email: %s", email)
	}

	if time.Now().After(stored.expiredAt) {
		delete(p.verifyStore, email)
		return fmt.Errorf("verify code expired")
	}

	if stored.code != code {
		return fmt.Errorf("invalid verify code")
	}

	// 验证成功后删除验证码
	delete(p.verifyStore, email)
	return nil
}

// generateCode 生成验证码
func (p *EmailPlugin) generateCode() string {
	const charset = "0123456789"
	code := make([]byte, p.codeLength)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}
