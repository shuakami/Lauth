package email

import (
	"fmt"
	hookenum "lauth/internal/plugin/hook/email"
	"lauth/pkg/config"
)

// EmailSender 邮件发送接口
type EmailSender interface {
	// Send 发送纯文本邮件
	Send(to string, subject string, content string) error
	// SendWithTemplate 发送HTML模板邮件
	SendWithTemplate(to string, subject string, templateName string, data interface{}) error
}

// DefaultEmailSender 默认的邮件发送器实现
type DefaultEmailSender struct {
	service hookenum.EmailService
}

// NewDefaultEmailSender 创建默认邮件发送器
func NewDefaultEmailSender(smtpConfig *config.SMTPConfig) EmailSender {
	// 创建邮件服务
	service, err := hookenum.NewEmailService(&hookenum.Options{
		SMTPConfig: smtpConfig,
	})
	if err != nil {
		// 如果创建服务失败，使用日志发送器
		fmt.Printf("Failed to create email service: %v, fallback to log sender\n", err)
		return &LogEmailSender{}
	}

	return &DefaultEmailSender{
		service: service,
	}
}

// Send 发送纯文本邮件
func (s *DefaultEmailSender) Send(to string, subject string, content string) error {
	return s.service.SendText([]string{to}, subject, content)
}

// SendWithTemplate 发送HTML模板邮件
func (s *DefaultEmailSender) SendWithTemplate(to string, subject string, templateName string, data interface{}) error {
	return s.service.SendHTML([]string{to}, subject, templateName, data)
}

// LogEmailSender 日志发送器（用作后备）
type LogEmailSender struct{}

// Send 发送纯文本邮件（仅打印日志）
func (s *LogEmailSender) Send(to string, subject string, content string) error {
	fmt.Printf("[TEXT] Send email to %s\nSubject: %s\nContent: %s\n", to, subject, content)
	return nil
}

// SendWithTemplate 发送HTML模板邮件（仅打印日志）
func (s *LogEmailSender) SendWithTemplate(to string, subject string, templateName string, data interface{}) error {
	fmt.Printf("[HTML] Send email to %s\nSubject: %s\nTemplate: %s\nData: %+v\n", to, subject, templateName, data)
	return nil
}
