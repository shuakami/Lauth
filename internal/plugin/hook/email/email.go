package email

import (
	"fmt"
	"lauth/pkg/config"
)

// EmailService 邮件服务接口
type EmailService interface {
	// SendText 发送纯文本邮件
	SendText(to []string, subject, content string) error
	// SendHTML 发送HTML邮件
	SendHTML(to []string, subject, templateName string, data interface{}) error
	// AddTemplate 添加HTML模板
	AddTemplate(name, content string) error
	// GetTemplateNames 获取所有已注册的模板名称
	GetTemplateNames() []string
}

// Options 邮件服务配置选项
type Options struct {
	// SMTP配置，如果为nil则使用config.yaml中的配置
	SMTPConfig *config.SMTPConfig
	// 模板目录，如果为空则使用config.yaml中的配置
	TemplatePath string
	// 是否使用日志发送器（用于测试）
	UseLogSender bool
}

// NewEmailService 创建邮件服务实例
// 使用默认配置：
//
//	service := email.NewEmailService(nil)
//
// 使用自定义配置：
//
//	service := email.NewEmailService(&email.Options{
//	    SMTPConfig: &config.SMTPConfig{...},
//	    TemplatePath: "path/to/templates",
//	})
//
// 使用日志发送器（用于测试）：
//
//	service := email.NewEmailService(&email.Options{UseLogSender: true})
func NewEmailService(opts *Options) (EmailService, error) {
	if opts == nil {
		opts = &Options{}
	}

	// 如果没有提供SMTP配置，使用默认配置
	smtpConfig := opts.SMTPConfig
	if smtpConfig == nil {
		smtpConfig = config.NewSMTPConfig()
	}

	// 如果指定使用日志发送器
	if opts.UseLogSender {
		return newLogSender(), nil
	}

	// 创建SMTP发送器
	sender, err := NewSMTPSender(smtpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMTP sender: %v", err)
	}

	return sender, nil
}
